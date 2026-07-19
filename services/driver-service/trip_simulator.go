package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"sync"
	"time"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/tracking"

	"github.com/rabbitmq/amqp091-go"
)

// tripSimulator advances a seeded auto-accepted trip through Uber-like stages
// and moves the driver location in Redis so the rider map animates.
// It pauses after arrived until OTP verification (trip.event.started) and aborts on cancel.
type tripSimulator struct {
	rabbitmq  *messaging.RabbitMQ
	locations *tracking.LocationStore
	service   *Service

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
	started map[string]chan struct{}
}

func newTripSimulator(rb *messaging.RabbitMQ, svc *Service, locations *tracking.LocationStore) *tripSimulator {
	s := &tripSimulator{
		rabbitmq:  rb,
		service:   svc,
		locations: locations,
		cancels:   make(map[string]context.CancelFunc),
		started:   make(map[string]chan struct{}),
	}
	go s.listenControlEvents()
	return s
}

func (s *tripSimulator) Start(trip *pb.Trip, driver *pbd.Driver) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	if prev, ok := s.cancels[trip.Id]; ok {
		prev()
	}
	s.cancels[trip.Id] = cancel
	s.started[trip.Id] = make(chan struct{})
	s.mu.Unlock()
	go s.run(ctx, trip, driver)
}

func (s *tripSimulator) Cancel(tripID string) {
	s.mu.Lock()
	if cancel, ok := s.cancels[tripID]; ok {
		cancel()
		delete(s.cancels, tripID)
	}
	delete(s.started, tripID)
	s.mu.Unlock()
}

func (s *tripSimulator) MarkStarted(tripID string) {
	s.mu.Lock()
	ch, ok := s.started[tripID]
	s.mu.Unlock()
	if !ok || ch == nil {
		return
	}
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func (s *tripSimulator) listenControlEvents() {
	_ = s.rabbitmq.ConsumeMessages(messaging.DriverSimControlQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &envelope); err != nil {
			return nil
		}
		switch msg.RoutingKey {
		case contracts.TripEventStarted, contracts.TripEventOTPVerified:
			var data messaging.TripEventData
			var result messaging.TripOTPResultData
			if json.Unmarshal(envelope.Data, &data) == nil && data.Trip != nil {
				s.MarkStarted(data.Trip.Id)
			} else if json.Unmarshal(envelope.Data, &result) == nil && result.Success {
				s.MarkStarted(result.TripID)
			}
		case contracts.TripEventCancelled:
			var data messaging.TripEventData
			if json.Unmarshal(envelope.Data, &data) == nil && data.Trip != nil {
				s.Cancel(data.Trip.Id)
				if data.Trip.Driver != nil && data.Trip.Driver.Id != "" {
					s.service.MarkBusy(data.Trip.Driver.Id, false)
				}
			}
			if envelope.OwnerID != "" {
				s.service.MarkBusy(envelope.OwnerID, false)
			}
		}
		return nil
	})
}

func (s *tripSimulator) run(ctx context.Context, trip *pb.Trip, driver *pbd.Driver) {
	defer s.Cancel(trip.Id)

	riderID := trip.UserID

	if trip.Route == nil || len(trip.Route.Geometry) == 0 || len(trip.Route.Geometry[0].Coordinates) < 2 {
		log.Printf("simulate: trip %s missing route geometry, skipping animation", trip.Id)
		_ = s.publishStatus(ctx, contracts.TripEventDriverEnRoute, riderID, trip, driver)
		if !sleepOrDone(ctx, 2*time.Second) {
			return
		}
		_ = s.publishStatus(ctx, contracts.TripEventDriverArrived, riderID, trip, driver)
		if !s.waitForOTPStart(ctx, trip.Id) {
			return
		}
		if !sleepOrDone(ctx, 3*time.Second) {
			return
		}
		_ = s.publishStatus(ctx, contracts.TripEventCompleted, riderID, trip, driver)
		return
	}

	pickup := trip.Route.Geometry[0].Coordinates[0]
	destCoords := trip.Route.Geometry[0].Coordinates

	startLat, startLon := driver.Location.Latitude, driver.Location.Longitude
	if s.locations != nil {
		_ = s.locations.UpsertDriver(ctx, tracking.DriverLocation{
			ID:             driver.Id,
			Latitude:       startLat,
			Longitude:      startLon,
			Name:           driver.Name,
			ProfilePicture: driver.ProfilePicture,
			CarPlate:       driver.CarPlate,
			PackageSlug:    driver.PackageSlug,
		})
	}

	if !sleepOrDone(ctx, 1500*time.Millisecond) {
		return
	}
	if err := s.publishStatus(ctx, contracts.TripEventDriverEnRoute, riderID, trip, driver); err != nil {
		log.Printf("simulate en_route: %v", err)
	}

	if !s.animate(ctx, driver, startLat, startLon, pickup.Latitude, pickup.Longitude, 8) {
		return
	}

	if err := s.publishStatus(ctx, contracts.TripEventDriverArrived, riderID, trip, driver); err != nil {
		log.Printf("simulate arrived: %v", err)
	}

	log.Printf("simulate: waiting for OTP start on trip %s", trip.Id)
	if !s.waitForOTPStart(ctx, trip.Id) {
		log.Printf("simulate: cancelled or timed out waiting for OTP on trip %s", trip.Id)
		return
	}

	if !s.animateAlongRoute(ctx, driver, destCoords, 12) {
		return
	}

	if err := s.publishStatus(ctx, contracts.TripEventCompleted, riderID, trip, driver); err != nil {
		log.Printf("simulate completed: %v", err)
	}
	log.Printf("Trip simulation complete for trip %s driver %s", trip.Id, driver.Id)
}

func (s *tripSimulator) waitForOTPStart(ctx context.Context, tripID string) bool {
	s.mu.Lock()
	ch := s.started[tripID]
	s.mu.Unlock()
	if ch == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case <-ch:
		return true
	case <-time.After(10 * time.Minute):
		return false
	}
}

func (s *tripSimulator) publishStatus(ctx context.Context, event, riderID string, trip *pb.Trip, driver *pbd.Driver) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	trip.Status = statusFromEvent(event)
	if driver != nil {
		trip.Driver = &pb.TripDriver{
			Id:             driver.Id,
			Name:           driver.Name,
			ProfilePicture: driver.ProfilePicture,
			CarPlate:       driver.CarPlate,
		}
	}
	payload, err := json.Marshal(messaging.TripEventData{Trip: trip})
	if err != nil {
		return err
	}
	return s.rabbitmq.PublishMessage(ctx, event, contracts.AmqpMessage{
		OwnerID: riderID,
		Data:    payload,
	})
}

func statusFromEvent(event string) string {
	switch event {
	case contracts.TripEventDriverEnRoute:
		return "en_route"
	case contracts.TripEventDriverArrived:
		return "arrived"
	case contracts.TripEventStarted:
		return "in_progress"
	case contracts.TripEventCompleted:
		return "completed"
	case contracts.TripEventCancelled:
		return "cancelled"
	default:
		return "accepted"
	}
}

func (s *tripSimulator) animate(ctx context.Context, driver *pbd.Driver, fromLat, fromLon, toLat, toLon float64, steps int) bool {
	for i := 1; i <= steps; i++ {
		if ctx.Err() != nil {
			return false
		}
		t := float64(i) / float64(steps)
		lat := fromLat + (toLat-fromLat)*t
		lon := fromLon + (toLon-fromLon)*t
		driver.Location.Latitude = lat
		driver.Location.Longitude = lon
		if s.locations != nil {
			_ = s.locations.UpsertDriver(ctx, tracking.DriverLocation{
				ID:             driver.Id,
				Latitude:       lat,
				Longitude:      lon,
				Name:           driver.Name,
				ProfilePicture: driver.ProfilePicture,
				CarPlate:       driver.CarPlate,
				PackageSlug:    driver.PackageSlug,
			})
		}
		if !sleepOrDone(ctx, 700*time.Millisecond) {
			return false
		}
	}
	return true
}

func (s *tripSimulator) animateAlongRoute(ctx context.Context, driver *pbd.Driver, coords []*pb.Coordinate, steps int) bool {
	if len(coords) < 2 {
		return true
	}
	for i := 1; i <= steps; i++ {
		if ctx.Err() != nil {
			return false
		}
		idx := int(math.Round(float64(len(coords)-1) * float64(i) / float64(steps)))
		if idx >= len(coords) {
			idx = len(coords) - 1
		}
		c := coords[idx]
		driver.Location.Latitude = c.Latitude
		driver.Location.Longitude = c.Longitude
		if s.locations != nil {
			_ = s.locations.UpsertDriver(ctx, tracking.DriverLocation{
				ID:             driver.Id,
				Latitude:       c.Latitude,
				Longitude:      c.Longitude,
				Name:           driver.Name,
				ProfilePicture: driver.ProfilePicture,
				CarPlate:       driver.CarPlate,
				PackageSlug:    driver.PackageSlug,
			})
		}
		if !sleepOrDone(ctx, 700*time.Millisecond) {
			return false
		}
	}
	return true
}

func sleepOrDone(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// wanderSeeds gently moves seeded drivers so the rider map feels alive.
func (s *Service) WanderSeedDrivers(ctx context.Context) {
	if s.locations == nil {
		return
	}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	offsets := 0.0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			offsets += 0.0003
			s.mu.RLock()
			var drivers []*driverPresence
			for _, d := range s.drivers {
				if d.Online && d.Driver != nil && IsLocalSeededDriver(d.Driver.Id) {
					drivers = append(drivers, d)
				}
			}
			s.mu.RUnlock()

			for i, d := range drivers {
				if d.Driver.Location == nil {
					continue
				}
				phase := float64(i) + offsets
				lat := d.Driver.Location.Latitude + 0.0015*math.Sin(phase)
				lon := d.Driver.Location.Longitude + 0.0015*math.Cos(phase)
				_ = s.locations.UpsertDriver(ctx, tracking.DriverLocation{
					ID:             d.Driver.Id,
					Latitude:       lat,
					Longitude:      lon,
					Name:           d.Driver.Name,
					ProfilePicture: d.Driver.ProfilePicture,
					CarPlate:       d.Driver.CarPlate,
					PackageSlug:    d.Driver.PackageSlug,
				})
			}
		}
	}
}
