package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	rabbitmq     *messaging.RabbitMQ
	service      *Service
	autoAccept   bool
	simulate     bool
	simulator    *tripSimulator
	offerTimeout time.Duration
}

func NewTripConsumer(rabbitmq *messaging.RabbitMQ, service *Service, autoAccept, simulate bool, simulator *tripSimulator) *tripConsumer {
	timeoutSec := env.GetInt("TRIP_OFFER_TIMEOUT_SEC", 30)
	return &tripConsumer{
		rabbitmq:     rabbitmq,
		service:      service,
		autoAccept:   autoAccept,
		simulate:     simulate,
		simulator:    simulator,
		offerTimeout: time.Duration(timeoutSec) * time.Second,
	}
}

func (c *tripConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, c.onFindDrivers)
}

func (c *tripConsumer) onFindDrivers(ctx context.Context, msg amqp091.Delivery) error {
	var tripEvent contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &tripEvent); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return err
	}

	var payload messaging.TripEventData
	if err := json.Unmarshal(tripEvent.Data, &payload); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return err
	}

	switch msg.RoutingKey {
	case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
		return c.handleFindAndNotifyDrivers(ctx, payload)
	}
	return nil
}

func (c *tripConsumer) handleFindAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {
	suitableIDs := c.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug)
	log.Printf("Found suitable drivers %v", len(suitableIDs))

	if len(suitableIDs) == 0 {
		if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			return err
		}
		return nil
	}

	suitableDriverID := suitableIDs[rand.Intn(len(suitableIDs))]
	for _, id := range suitableIDs {
		if !IsLocalSeededDriver(id) {
			suitableDriverID = id
			break
		}
	}

	driver := c.service.GetDriver(suitableDriverID)
	if driver == nil {
		return nil
	}

	if c.autoAccept && IsLocalSeededDriver(suitableDriverID) {
		log.Printf("Auto-accepting trip %s for local driver %s", payload.Trip.Id, suitableDriverID)
		c.service.MarkBusy(suitableDriverID, true)
		acceptPayload, err := json.Marshal(messaging.DriverTripResponseData{
			TripID:  payload.Trip.Id,
			RiderID: payload.Trip.UserID,
			Driver:  driver,
		})
		if err != nil {
			return err
		}
		if err := c.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripAccept, contracts.AmqpMessage{
			OwnerID: suitableDriverID,
			Data:    acceptPayload,
		}); err != nil {
			return err
		}
		if c.simulate && c.simulator != nil {
			c.simulator.Start(payload.Trip, driver)
		}
		return nil
	}

	marshalledEvent, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	c.service.SetPendingOffer(suitableDriverID, payload.Trip.Id)

	if err := c.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverID,
		Data:    marshalledEvent,
	}); err != nil {
		c.service.ClearPendingOffer(suitableDriverID, payload.Trip.Id)
		return err
	}

	go c.watchOfferTimeout(payload.Trip.Id, payload.Trip.UserID, suitableDriverID, payload)
	return nil
}

func (c *tripConsumer) watchOfferTimeout(tripID, riderID, driverID string, payload messaging.TripEventData) {
	time.Sleep(c.offerTimeout)
	if !c.service.ClearPendingOffer(driverID, tripID) {
		return
	}
	log.Printf("Offer timeout for trip %s driver %s — rematching", tripID, driverID)
	c.service.MarkBusy(driverID, false)

	ctx := context.Background()
	marshalledPayload, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverNotInterested, contracts.AmqpMessage{
		OwnerID: riderID,
		Data:    marshalledPayload,
	})
}
