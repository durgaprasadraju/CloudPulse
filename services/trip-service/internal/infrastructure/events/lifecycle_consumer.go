package events

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pbd "ride-sharing/shared/proto/driver"

	"github.com/rabbitmq/amqp091-go"
)

// lifecycleConsumer persists trip status transitions and creates payment
// after the trip is completed (Uber-like: pay after the ride).
type lifecycleConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
}

func NewLifecycleConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService) *lifecycleConsumer {
	return &lifecycleConsumer{rabbitmq: rabbitmq, service: service}
}

func (c *lifecycleConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.TripLifecycleUpdateQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("lifecycle: unmarshal amqp: %v", err)
			return err
		}

		var payload messaging.TripEventData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			var cancelPayload struct {
				TripID string `json:"tripID"`
				Status string `json:"status"`
			}
			if err2 := json.Unmarshal(message.Data, &cancelPayload); err2 != nil {
				log.Printf("lifecycle: unmarshal payload: %v", err)
				return err
			}
			if cancelPayload.TripID != "" && msg.RoutingKey == contracts.TripEventCancelled {
				_, _ = c.service.CancelTrip(ctx, cancelPayload.TripID, message.OwnerID)
			}
			return nil
		}

		if payload.Trip == nil || payload.Trip.Id == "" {
			return nil
		}

		status := statusFromRoutingKey(msg.RoutingKey)
		if status == "" {
			return nil
		}

		// Started is only allowed via OTP verification — ignore forged client events.
		if msg.RoutingKey == contracts.TripEventStarted {
			log.Printf("lifecycle: ignoring client-originated started for trip %s (OTP required)", payload.Trip.Id)
			return nil
		}

		var driver *pbd.Driver
		if payload.Trip.Driver != nil {
			driver = &pbd.Driver{
				Id:             payload.Trip.Driver.Id,
				Name:           payload.Trip.Driver.Name,
				ProfilePicture: payload.Trip.Driver.ProfilePicture,
				CarPlate:       payload.Trip.Driver.CarPlate,
			}
		}

		if msg.RoutingKey == contracts.TripEventCancelled {
			_, err := c.service.CancelTrip(ctx, payload.Trip.Id, message.OwnerID)
			if err != nil {
				log.Printf("lifecycle: cancel trip %s: %v", payload.Trip.Id, err)
			}
			return nil
		}

		if msg.RoutingKey == contracts.TripEventCompleted {
			trip, err := c.service.CompleteTrip(ctx, payload.Trip.Id)
			if err != nil {
				log.Printf("lifecycle: complete trip %s: %v", payload.Trip.Id, err)
				return nil
			}
			return c.createPaymentSession(ctx, trip.ID.Hex())
		}

		if _, err := c.service.TransitionTrip(ctx, payload.Trip.Id, status, driver); err != nil {
			log.Printf("lifecycle: update trip %s -> %s: %v", payload.Trip.Id, status, err)
			return nil
		}
		return nil
	})
}

func statusFromRoutingKey(key string) string {
	switch key {
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
		return ""
	}
}

func (c *lifecycleConsumer) createPaymentSession(ctx context.Context, tripID string) error {
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil || trip == nil {
		return err
	}
	if trip.Status != "completed" && trip.Status != "payed" {
		log.Printf("lifecycle: skip payment for trip %s status=%s", tripID, trip.Status)
		return nil
	}

	driverID := ""
	if trip.Driver != nil {
		driverID = trip.Driver.Id
	}

	amount := 0.0
	if trip.RideFare != nil {
		amount = trip.RideFare.TotalPriceInCents
	}

	marshalledPayload, err := json.Marshal(messaging.PaymentTripResponseData{
		TripID:   tripID,
		UserID:   trip.UserID,
		DriverID: driverID,
		Amount:   amount,
		Currency: "INR",
	})
	if err != nil {
		return err
	}

	return c.rabbitmq.PublishMessage(ctx, contracts.PaymentCmdCreateSession, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    marshalledPayload,
	})
}
