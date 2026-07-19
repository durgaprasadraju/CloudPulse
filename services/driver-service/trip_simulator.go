package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"time"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/tracking"
)

// tripSimulator advances a seeded auto-accepted trip through Uber-like stages
// and moves the driver location in Redis so the rider map animates.
type tripSimulator struct {
	rabbitmq  *messaging.RabbitMQ
	locations *tracking.LocationStore
	service   *Service
}

func newTripSimulator(rb *messaging.RabbitMQ, svc *Service, locations *tracking.LocationStore) *tripSimulator {
	return &tripSimulator{rabbitmq: rb, service: svc, locations: locations}
}

func (s *tripSimulator) Start(trip *pb.Trip, driver *pbd.Driver) {
	go s.run(trip, driver)
}

func (s *tripSimulator) run(trip *pb.Trip, driver *pbd.Driver) {
	ctx := context.Background()
	riderID := trip.UserID

	if trip.Route == nil || len(trip.Route.Geometry) == 0 || len(trip.Route.Geometry[0].Coordinates) < 2 {
		log.Printf("simulate: trip %s missing route geometry, skipping animation", trip.Id)
		_ = s.publishStatus(ctx, contracts.TripEventDriverEnRoute, riderID, trip, driver)
		time.Sleep(2 * time.Second)
		_ = s.publishStatus(ctx, contracts.TripEventDriverArrived, riderID, trip, driver)
		time.Sleep(1 * time.Second)
		_ = s.publishStatus(ctx, contracts.TripEventStarted, riderID, trip, driver)
		time.Sleep(3 * time.Second)
		_ = s.publishStatus(ctx, contracts.TripEventCompleted, riderID, trip, driver)
		return
	}

	pickup := trip.Route.Geometry[0].Coordinates[0]
	destCoords := trip.Route.Geometry[0].Coordinates
	dropoff := destCoords[len(destCoords)-1]

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

	time.Sleep(1500 * time.Millisecond)
	if err := s.publishStatus(ctx, contracts.TripEventDriverEnRoute, riderID, trip, driver); err != nil {
		log.Printf("simulate en_route: %v", err)
	}

	s.animate(ctx, driver, startLat, startLon, pickup.Latitude, pickup.Longitude, 8)

	if err := s.publishStatus(ctx, contracts.TripEventDriverArrived, riderID, trip, driver); err != nil {
		log.Printf("simulate arrived: %v", err)
	}
	time.Sleep(2 * time.Second)

	if err := s.publishStatus(ctx, contracts.TripEventStarted, riderID, trip, driver); err != nil {
		log.Printf("simulate started: %v", err)
	}

	s.animateAlongRoute(ctx, driver, destCoords, 12)

	if err := s.publishStatus(ctx, contracts.TripEventCompleted, riderID, trip, driver); err != nil {
		log.Printf("simulate completed: %v", err)
	}

	_ = dropoff
	log.Printf("Trip simulation complete for trip %s driver %s", trip.Id, driver.Id)
}

func (s *tripSimulator) publishStatus(ctx context.Context, event, riderID string, trip *pb.Trip, driver *pbd.Driver) error {
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

func (s *tripSimulator) animate(ctx context.Context, driver *pbd.Driver, fromLat, fromLon, toLat, toLon float64, steps int) {
	for i := 1; i <= steps; i++ {
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
		time.Sleep(700 * time.Millisecond)
	}
}

func (s *tripSimulator) animateAlongRoute(ctx context.Context, driver *pbd.Driver, coords []*pb.Coordinate, steps int) {
	if len(coords) < 2 {
		return
	}
	// Sample evenly across the route polyline
	for i := 1; i <= steps; i++ {
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
		time.Sleep(700 * time.Millisecond)
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
			drivers := append([]*driverInMap(nil), s.drivers...)
			s.mu.RUnlock()

			for i, d := range drivers {
				if !IsLocalSeededDriver(d.Driver.Id) || d.Driver.Location == nil {
					continue
				}
				// Small circular wander so markers move without leaving SF
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
