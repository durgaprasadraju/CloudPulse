package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	rabbitmq   *messaging.RabbitMQ
	service    *Service
	autoAccept bool
	simulate   bool
	simulator  *tripSimulator
}

func NewTripConsumer(rabbitmq *messaging.RabbitMQ, service *Service, autoAccept, simulate bool, simulator *tripSimulator) *tripConsumer {
	return &tripConsumer{
		rabbitmq:   rabbitmq,
		service:    service,
		autoAccept: autoAccept,
		simulate:   simulate,
		simulator:  simulator,
	}
}

func (c *tripConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp091.Delivery) error {
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

		log.Printf("driver received message: %+v", payload)

		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			return c.handleFindAndNotifyDrivers(ctx, payload)
		}

		log.Printf("unknown trip event: %+v", payload)

		return nil
	})
}

func (c *tripConsumer) handleFindAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {
	suitableIDs := c.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug)

	log.Printf("Found suitable drivers %v", len(suitableIDs))

	if len(suitableIDs) == 0 {
		// Notify the driver that no drivers are available
		if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			log.Printf("Failed to publish message to exchange: %v", err)
			return err
		}

		return nil
	}

	// Prefer a live (browser-connected) driver over a seeded local one when both exist.
	suitableDriverID := suitableIDs[rand.Intn(len(suitableIDs))]
	for _, id := range suitableIDs {
		if !IsLocalSeededDriver(id) {
			suitableDriverID = id
			break
		}
	}

	driver := c.service.GetDriver(suitableDriverID)
	if driver == nil {
		log.Printf("Driver %s not found in memory", suitableDriverID)
		return nil
	}

	// Local prototype: auto-accept seeded drivers so a rider-only tab can reach payment.
	if c.autoAccept && IsLocalSeededDriver(suitableDriverID) {
		log.Printf("Auto-accepting trip %s for local driver %s", payload.Trip.Id, suitableDriverID)
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
			log.Printf("Failed to publish auto-accept: %v", err)
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

	// Notify the driver about a potential trip
	if err := c.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverID,
		Data:    marshalledEvent,
	}); err != nil {
		log.Printf("Failed to publish message to exchange: %v", err)
		return err
	}

	return nil
}
