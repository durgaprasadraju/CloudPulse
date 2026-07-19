package main

import (
	"context"
	"log"
	"strings"

	math "math/rand/v2"
	pb "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/tracking"
	"ride-sharing/shared/util"
	"sync"

	"github.com/mmcloughlin/geohash"
)

type driverInMap struct {
	Driver *pb.Driver
	// Index int
	// TODO: route
}

type Service struct {
	drivers []*driverInMap
	mu      sync.RWMutex

	// Optional Terraform-provisioned backends (nil when not configured)
	store     *DriverStore            // PostgreSQL — persistent driver registry
	locations *tracking.LocationStore // Redis — real-time location tracking
}

func NewService() *Service {
	return &Service{
		drivers: make([]*driverInMap, 0),
	}
}

// AttachStore enables PostgreSQL persistence and loads previously
// registered drivers into memory.
func (s *Service) AttachStore(store *DriverStore) {
	s.store = store

	persisted, err := store.LoadDrivers(context.Background())
	if err != nil {
		log.Printf("Failed to load persisted drivers: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range persisted {
		s.drivers = append(s.drivers, &driverInMap{Driver: d})
	}
	log.Printf("Loaded %d drivers from PostgreSQL", len(persisted))
}

// AttachLocationStore enables Redis location tracking.
func (s *Service) AttachLocationStore(locations *tracking.LocationStore) {
	s.locations = locations
}

func (s *Service) FindAvailableDrivers(packageType string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matchingDrivers []string

	for _, driver := range s.drivers {
		if driver.Driver.PackageSlug == packageType {
			matchingDrivers = append(matchingDrivers, driver.Driver.Id)
		}
	}

	if len(matchingDrivers) == 0 {
		return []string{}
	}

	return matchingDrivers
}

func (s *Service) RegisterDriver(driverId string, packageSlug string) (*pb.Driver, error) {
	driver := s.registerInMemory(driverId, packageSlug)

	if s.store != nil {
		if err := s.store.SaveDriver(context.Background(), driver); err != nil {
			log.Printf("Failed to persist driver %s: %v", driverId, err)
		}
	}

	if s.locations != nil && driver.Location != nil {
		if err := s.locations.UpsertDriver(context.Background(), tracking.DriverLocation{
			ID:             driverId,
			Latitude:       driver.Location.Latitude,
			Longitude:      driver.Location.Longitude,
			Name:           driver.Name,
			ProfilePicture: driver.ProfilePicture,
			CarPlate:       driver.CarPlate,
			PackageSlug:    driver.PackageSlug,
		}); err != nil {
			log.Printf("Failed to track driver %s location: %v", driverId, err)
		}
	}

	return driver, nil
}

func (s *Service) registerInMemory(driverId string, packageSlug string) *pb.Driver {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reuse existing registration if the same driver reconnects
	for _, existing := range s.drivers {
		if existing.Driver.Id == driverId {
			existing.Driver.PackageSlug = packageSlug
			return existing.Driver
		}
	}

	randomIndex := math.IntN(len(PredefinedRoutes))
	randomRoute := PredefinedRoutes[randomIndex]

	randomPlate := GenerateRandomPlate()
	randomAvatar := util.GetRandomAvatar(randomIndex)

	// we can ignore this property for now, but it must be sent to the frontend.
	geohash := geohash.Encode(randomRoute[0][0], randomRoute[0][1])

	driver := &pb.Driver{
		Id:             driverId,
		Geohash:        geohash,
		Location:       &pb.Location{Latitude: randomRoute[0][0], Longitude: randomRoute[0][1]},
		Name:           "Lando Norris",
		PackageSlug:    packageSlug,
		ProfilePicture: randomAvatar,
		CarPlate:       randomPlate,
	}

	s.drivers = append(s.drivers, &driverInMap{
		Driver: driver,
	})

	return driver
}

// SeedLocalDrivers registers one always-available driver per package for local prototypes.
func (s *Service) SeedLocalDrivers(packageSlugs []string) {
	for _, slug := range packageSlugs {
		driverID := "local-driver-" + slug
		_, _ = s.RegisterDriver(driverID, slug)
	}
}

func (s *Service) GetDriver(driverID string) *pb.Driver {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, driver := range s.drivers {
		if driver.Driver.Id == driverID {
			return driver.Driver
		}
	}
	return nil
}

func IsLocalSeededDriver(driverID string) bool {
	return strings.HasPrefix(driverID, "local-driver-")
}

func (s *Service) UnregisterDriver(driverId string) {
	s.mu.Lock()
	for i, driver := range s.drivers {
		if driver.Driver.Id == driverId {
			s.drivers = append(s.drivers[:i], s.drivers[i+1:]...)
			break
		}
	}
	s.mu.Unlock()

	if s.store != nil {
		if err := s.store.DeleteDriver(context.Background(), driverId); err != nil {
			log.Printf("Failed to delete persisted driver %s: %v", driverId, err)
		}
	}

	if s.locations != nil {
		if err := s.locations.RemoveDriver(context.Background(), driverId); err != nil {
			log.Printf("Failed to remove driver %s from tracking: %v", driverId, err)
		}
	}
}
