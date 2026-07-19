package events

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

// otpConsumer handles OTP verification and rider cancellation commands.
type otpConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
}

func NewOTPConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService) *otpConsumer {
	return &otpConsumer{rabbitmq: rabbitmq, service: service}
}

func (c *otpConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.TripOTPVerifyQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("otp: unmarshal amqp: %v", err)
			return err
		}

		switch msg.RoutingKey {
		case contracts.TripCmdVerifyOTP:
			return c.handleVerify(ctx, message)
		case contracts.TripCmdCancel:
			return c.handleCancel(ctx, message)
		}
		return nil
	})
}

func (c *otpConsumer) handleVerify(ctx context.Context, message contracts.AmqpMessage) error {
	var payload messaging.TripVerifyOTPData
	if err := json.Unmarshal(message.Data, &payload); err != nil {
		log.Printf("otp: unmarshal verify: %v", err)
		return err
	}
	if payload.TripID == "" || payload.OTP == "" {
		return c.publishOTPResult(ctx, payload.DriverID, payload.TripID, false, "tripID and otp required")
	}

	driverID := payload.DriverID
	if driverID == "" {
		driverID = message.OwnerID
	}

	trip, err := c.service.VerifyOTPAndStart(ctx, payload.TripID, payload.OTP, driverID)
	if err != nil {
		log.Printf("otp: verify failed trip=%s: %v", payload.TripID, err)
		return c.publishOTPResult(ctx, driverID, payload.TripID, false, err.Error())
	}

	if err := c.publishOTPResult(ctx, driverID, payload.TripID, true, "verified"); err != nil {
		return err
	}

	marshalled, err := json.Marshal(messaging.TripEventData{Trip: trip.ToProto()})
	if err != nil {
		return err
	}
	return c.rabbitmq.PublishMessage(ctx, contracts.TripEventStarted, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    marshalled,
	})
}

func (c *otpConsumer) handleCancel(ctx context.Context, message contracts.AmqpMessage) error {
	var payload messaging.TripCancelData
	if err := json.Unmarshal(message.Data, &payload); err != nil {
		log.Printf("otp: unmarshal cancel: %v", err)
		return err
	}
	userID := payload.UserID
	if userID == "" {
		userID = message.OwnerID
	}
	if payload.TripID == "" {
		return nil
	}

	trip, err := c.service.CancelTrip(ctx, payload.TripID, userID)
	if err != nil {
		log.Printf("otp: cancel failed trip=%s: %v", payload.TripID, err)
		return nil
	}

	marshalled, err := json.Marshal(messaging.TripEventData{Trip: trip.ToProto()})
	if err != nil {
		return err
	}

	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventCancelled, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    marshalled,
	}); err != nil {
		return err
	}

	driverID := ""
	if trip.Driver != nil {
		driverID = trip.Driver.Id
	}
	if driverID != "" {
		_ = c.rabbitmq.PublishMessage(ctx, contracts.TripEventCancelled, contracts.AmqpMessage{
			OwnerID: driverID,
			Data:    marshalled,
		})
		releaseDriver(driverID, trip.ID.Hex())
	}
	return nil
}

func releaseDriver(driverID, tripID string) {
	base := os.Getenv("DRIVER_SERVICE_HTTP_URL")
	if base == "" {
		base = "http://driver-service:8085"
	}
	base = strings.TrimRight(base, "/")
	payload, _ := json.Marshal(map[string]any{
		"driverId": driverID,
		"tripId":   tripID,
		"accepted": false,
	})
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, base+"/internal/offer-response", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("otp: release driver %s: %v", driverID, err)
		return
	}
	_ = resp.Body.Close()
}

func (c *otpConsumer) publishOTPResult(ctx context.Context, driverID, tripID string, success bool, msg string) error {
	if driverID == "" {
		return nil
	}
	payload, err := json.Marshal(messaging.TripOTPResultData{
		TripID:  tripID,
		Success: success,
		Message: msg,
	})
	if err != nil {
		return err
	}
	event := contracts.TripEventOTPFailed
	if success {
		event = contracts.TripEventOTPVerified
	}
	return c.rabbitmq.PublishMessage(ctx, event, contracts.AmqpMessage{
		OwnerID: driverID,
		Data:    payload,
	})
}
