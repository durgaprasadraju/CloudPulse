package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/proto/driver"
	"ride-sharing/shared/tracking"
)

// driverLocationMessage matches the driver.cmd.location WS payload from the frontend.
type driverLocationMessage struct {
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	Geohash string `json:"geohash"`
}

type riderLocationMessage struct {
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
}

var (
	connManager = messaging.NewConnectionManager()
)

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ, locations *tracking.LocationStore) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("No user ID provided")
		return
	}

	connManager.Add(userID, conn)
	defer connManager.Remove(userID)

	queues := []string{
		messaging.NotifyDriverNoDriversFoundQueue,
		messaging.NotifyDriverAssignQueue,
		messaging.NotifyPaymentSessionCreatedQueue,
		messaging.NotifyTripLifecycleQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)
		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Default Hyderabad (Madhapur) until the rider sends their location
	riderLat, riderLon := 17.4401, 78.3915

	// Periodically push nearby drivers to this rider
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pushNearbyDrivers(ctx, userID, locations, riderLat, riderLon)
			}
		}
	}()

	// Immediate first push
	pushNearbyDrivers(ctx, userID, locations, riderLat, riderLon)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		type riderMessage struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		var msg riderMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshaling rider message: %v", err)
			continue
		}

		switch msg.Type {
		case contracts.RiderCmdLocation, contracts.DriverCmdLocation:
			var loc riderLocationMessage
			if err := json.Unmarshal(msg.Data, &loc); err != nil {
				continue
			}
			if loc.Location.Latitude != 0 || loc.Location.Longitude != 0 {
				riderLat, riderLon = loc.Location.Latitude, loc.Location.Longitude
				pushNearbyDrivers(ctx, userID, locations, riderLat, riderLon)
			}
		case contracts.TripCmdCancel:
			var cancelData struct {
				TripID string `json:"tripID"`
			}
			_ = json.Unmarshal(msg.Data, &cancelData)
			payload, _ := json.Marshal(messaging.TripCancelData{
				TripID: cancelData.TripID,
				UserID: userID,
			})
			if err := rb.PublishMessage(ctx, contracts.TripCmdCancel, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    payload,
			}); err != nil {
				log.Printf("Error publishing cancel: %v", err)
			}
		default:
			log.Printf("Received rider message type=%s", msg.Type)
		}
	}
}

func pushNearbyDrivers(ctx context.Context, userID string, locations *tracking.LocationStore, lat, lon float64) {
	if locations == nil {
		return
	}

	nearby, err := locations.NearbyDrivers(ctx, lat, lon, 15)
	if err != nil {
		log.Printf("nearby drivers query failed: %v", err)
		return
	}

	// Shape expected by the frontend Driver type
	drivers := make([]map[string]any, 0, len(nearby))
	for _, d := range nearby {
		name := d.Name
		if name == "" {
			name = "Available driver"
		}
		drivers = append(drivers, map[string]any{
			"id": d.ID,
			"location": map[string]float64{
				"latitude":  d.Latitude,
				"longitude": d.Longitude,
			},
			"geohash":        d.Geohash,
			"name":           name,
			"profilePicture": d.ProfilePicture,
			"carPlate":       d.CarPlate,
		})
	}

	if err := connManager.SendMessage(userID, contracts.WSMessage{
		Type: contracts.DriverCmdLocation,
		Data: drivers,
	}); err != nil {
		log.Printf("Failed to send nearby drivers to %s: %v", userID, err)
	}
}

func handleDriversWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ, locations *tracking.LocationStore) {
	userID, packageSlug, err := resolveDriverIdentity(r)
	if err != nil {
		log.Printf("Driver WS auth failed: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if packageSlug == "" {
		log.Println("No package slug provided")
		http.Error(w, "packageSlug required", http.StatusBadRequest)
		return
	}

	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	connManager.Add(userID, conn)

	ctx := r.Context()

	driverService, err := grpc_clients.NewDriverServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		connManager.Remove(userID)

		driverService.Client.UnregisterDriver(ctx, &driver.RegisterDriverRequest{
			DriverID:    userID,
			PackageSlug: packageSlug,
		})

		if locations != nil {
			_ = locations.RemoveDriver(context.Background(), userID)
		}

		driverService.Close()
		log.Println("Driver offline: ", userID)
	}()

	driverData, err := driverService.Client.RegisterDriver(ctx, &driver.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	})
	if err != nil {
		log.Printf("Error registering driver: %v", err)
		return
	}

	if err := connManager.SendMessage(userID, contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		Data: driverData.Driver,
	}); err != nil {
		log.Printf("Error sending message: %v", err)
		return
	}

	queues := []string{
		messaging.DriverCmdTripRequestQueue,
		messaging.NotifyDriverControlQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)
		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		type driverMessage struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		var driverMsg driverMessage
		if err := json.Unmarshal(message, &driverMsg); err != nil {
			log.Printf("Error unmarshaling driver message: %v", err)
			continue
		}

		switch driverMsg.Type {
		case contracts.DriverCmdLocation:
			if locations == nil {
				continue
			}

			var loc driverLocationMessage
			if err := json.Unmarshal(driverMsg.Data, &loc); err != nil {
				log.Printf("Error unmarshaling driver location: %v", err)
				continue
			}

			d := driverData.Driver
			meta := tracking.DriverLocation{
				ID:        userID,
				Latitude:  loc.Location.Latitude,
				Longitude: loc.Location.Longitude,
			}
			if d != nil {
				meta.Name = d.Name
				meta.ProfilePicture = d.ProfilePicture
				meta.CarPlate = d.CarPlate
				meta.PackageSlug = d.PackageSlug
			}
			if err := locations.UpsertDriver(ctx, meta); err != nil {
				log.Printf("Error updating driver location in Redis: %v", err)
			}
		case contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline:
			var tripMeta struct {
				TripID string `json:"tripID"`
			}
			_ = json.Unmarshal(driverMsg.Data, &tripMeta)
			notifyDriverOfferResponse(userID, tripMeta.TripID, driverMsg.Type == contracts.DriverCmdTripAccept)

			if err := rb.PublishMessage(ctx, driverMsg.Type, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    driverMsg.Data,
			}); err != nil {
				log.Printf("Error publishing message to RabbitMQ: %v", err)
			}
		case contracts.TripCmdVerifyOTP:
			var otpMsg struct {
				TripID  string `json:"tripID"`
				OTP     string `json:"otp"`
				RiderID string `json:"riderID"`
			}
			if err := json.Unmarshal(driverMsg.Data, &otpMsg); err != nil {
				log.Printf("Error unmarshaling OTP verify: %v", err)
				continue
			}
			payload, err := json.Marshal(messaging.TripVerifyOTPData{
				TripID:   otpMsg.TripID,
				OTP:      otpMsg.OTP,
				RiderID:  otpMsg.RiderID,
				DriverID: userID,
			})
			if err != nil {
				continue
			}
			if err := rb.PublishMessage(ctx, contracts.TripCmdVerifyOTP, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    payload,
			}); err != nil {
				log.Printf("Error publishing OTP verify: %v", err)
			}
		case contracts.TripCmdStatus:
			var statusMsg struct {
				Event   string          `json:"event"`
				RiderID string          `json:"riderID"`
				Trip    json.RawMessage `json:"trip"`
			}
			if err := json.Unmarshal(driverMsg.Data, &statusMsg); err != nil {
				log.Printf("Error unmarshaling trip status: %v", err)
				continue
			}
			event := statusMsg.Event
			if event == "" {
				continue
			}
			// Trip start is gated by OTP verification — never accept a forged started event.
			if event == contracts.TripEventStarted {
				log.Printf("Ignoring client-originated trip.event.started from driver %s", userID)
				continue
			}
			allowed := event == contracts.TripEventDriverEnRoute ||
				event == contracts.TripEventDriverArrived ||
				event == contracts.TripEventCompleted
			if !allowed {
				log.Printf("Ignoring unsupported trip status event %s from driver %s", event, userID)
				continue
			}
			if event == contracts.TripEventCompleted {
				notifyDriverOfferResponse(userID, "", false)
			}
			wrapped, err := json.Marshal(struct {
				Trip json.RawMessage `json:"trip"`
			}{Trip: statusMsg.Trip})
			if err != nil {
				log.Printf("Error wrapping trip status payload: %v", err)
				continue
			}
			if err := rb.PublishMessage(ctx, event, contracts.AmqpMessage{
				OwnerID: statusMsg.RiderID,
				Data:    wrapped,
			}); err != nil {
				log.Printf("Error publishing trip status: %v", err)
			}
		default:
			log.Printf("Unknown message type: %s", driverMsg.Type)
		}
	}
}
