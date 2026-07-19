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
			// cancel may send a minimal payload
			var cancelPayload struct {
				TripID string `json:"tripID"`
				Status string `json:"status"`
			}
			if err2 := json.Unmarshal(message.Data, &cancelPayload); err2 != nil {
				log.Printf("lifecycle: unmarshal payload: %v", err)
				return err
			}
			if cancelPayload.TripID != "" {
				return c.service.UpdateTrip(ctx, cancelPayload.TripID, "cancelled", nil)
			}
			return nil
		}

		if payload.Trip == nil || payload.Trip.Id == "" {
			return nil
		}

		status := payload.Trip.Status
		if status == "" {
			status = statusFromRoutingKey(msg.RoutingKey)
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

		if err := c.service.UpdateTrip(ctx, payload.Trip.Id, status, driver); err != nil {
			log.Printf("lifecycle: update trip %s -> %s: %v", payload.Trip.Id, status, err)
			return err
		}

		if msg.RoutingKey == contracts.TripEventCompleted {
			return c.createPaymentSession(ctx, payload.Trip.Id)
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
		return "accepted"
	}
}

func (c *lifecycleConsumer) createPaymentSession(ctx context.Context, tripID string) error {
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil || trip == nil {
		return err
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
		Currency: "USD",
	})
	if err != nil {
		return err
	}

	return c.rabbitmq.PublishMessage(ctx, contracts.PaymentCmdCreateSession, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    marshalledPayload,
	})
}
