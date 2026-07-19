package main

import (
	"context"
	"log"
	"strings"
	"sync"

	math "math/rand/v2"

	"ride-sharing/shared/auth"
	pb "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/tracking"
	"ride-sharing/shared/util"

	"github.com/mmcloughlin/geohash"
)

type driverPresence struct {
	Driver *pb.Driver
	Online bool
	Busy   bool
}

type Service struct {
	drivers       map[string]*driverPresence
	pendingOffers map[string]string // driverID -> tripID
	mu            sync.RWMutex

	accounts  *auth.AccountStore
	locations *tracking.LocationStore
}

func NewService() *Service {
	return &Service{
		drivers:       make(map[string]*driverPresence),
		pendingOffers: make(map[string]string),
	}
}

func (s *Service) AttachAccounts(accounts *auth.AccountStore) {
	s.accounts = accounts
}

func (s *Service) AttachLocationStore(locations *tracking.LocationStore) {
	s.locations = locations
}

// FindAvailableDrivers returns online, non-busy drivers matching the package.
func (s *Service) FindAvailableDrivers(packageType string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matching []string
	for id, d := range s.drivers {
		if !d.Online || d.Busy || d.Driver == nil {
			continue
		}
		if _, offered := s.pendingOffers[id]; offered {
			continue
		}
		if d.Driver.PackageSlug == packageType {
			matching = append(matching, id)
		}
	}
	return matching
}

func (s *Service) SetPendingOffer(driverID, tripID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingOffers[driverID] = tripID
	if s.accounts != nil {
		_ = s.accounts.SetAvailability(context.Background(), driverID, "offered")
	}
}

func (s *Service) ClearPendingOffer(driverID, tripID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.pendingOffers[driverID]
	if !ok {
		return false
	}
	if tripID != "" && current != tripID {
		return false
	}
	delete(s.pendingOffers, driverID)
	return true
}

func (s *Service) MarkBusy(driverID string, busy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.drivers[driverID]; ok {
		d.Busy = busy
	}
	if busy {
		delete(s.pendingOffers, driverID)
	}
	if s.accounts != nil {
		status := "available"
		if busy {
			status = "busy"
		}
		_ = s.accounts.SetAvailability(context.Background(), driverID, status)
	}
}

func (s *Service) RegisterDriver(driverId string, packageSlug string) (*pb.Driver, error) {
	driver := s.goOnline(driverId, packageSlug)

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

	if s.accounts != nil {
		_ = s.accounts.SetAvailability(context.Background(), driverId, "available")
	}

	return driver, nil
}

func (s *Service) goOnline(driverId string, packageSlug string) *pb.Driver {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.drivers[driverId]; ok && existing.Driver != nil {
		if packageSlug != "" {
			existing.Driver.PackageSlug = packageSlug
		}
		existing.Online = true
		existing.Busy = false
		if IsLocalSeededDriver(driverId) && len(PredefinedRoutes) > 0 {
			route := PredefinedRoutes[math.IntN(len(PredefinedRoutes))]
			existing.Driver.Location = &pb.Location{Latitude: route[0][0], Longitude: route[0][1]}
			existing.Driver.Geohash = geohash.Encode(route[0][0], route[0][1])
		}
		return existing.Driver
	}

	driver := s.buildDriverProfile(driverId, packageSlug)
	s.drivers[driverId] = &driverPresence{Driver: driver, Online: true, Busy: false}
	return driver
}

func (s *Service) buildDriverProfile(driverId, packageSlug string) *pb.Driver {
	// Prefer durable account profile when available.
	if s.accounts != nil && !IsLocalSeededDriver(driverId) {
		if acc, err := s.accounts.GetByID(context.Background(), driverId); err == nil && acc != nil {
			lat, lon := acc.Latitude, acc.Longitude
			if lat == 0 && lon == 0 && len(PredefinedRoutes) > 0 {
				route := PredefinedRoutes[0]
				lat, lon = route[0][0], route[0][1]
			}
			slug := acc.PackageSlug
			if packageSlug != "" {
				slug = packageSlug
			}
			return &pb.Driver{
				Id:             acc.ID,
				Name:           acc.Name,
				ProfilePicture: acc.ProfilePicture,
				CarPlate:       acc.CarPlate,
				PackageSlug:    slug,
				Geohash:        geohash.Encode(lat, lon),
				Location:       &pb.Location{Latitude: lat, Longitude: lon},
			}
		}
	}

	randomIndex := math.IntN(len(PredefinedRoutes))
	randomRoute := PredefinedRoutes[randomIndex]
	return &pb.Driver{
		Id:             driverId,
		Geohash:        geohash.Encode(randomRoute[0][0], randomRoute[0][1]),
		Location:       &pb.Location{Latitude: randomRoute[0][0], Longitude: randomRoute[0][1]},
		Name:           "CloudPulse Driver",
		PackageSlug:    packageSlug,
		ProfilePicture: util.GetRandomAvatar(randomIndex),
		CarPlate:       GenerateRandomPlate(),
	}
}

func (s *Service) SeedLocalDrivers(packageSlugs []string) {
	for _, slug := range packageSlugs {
		driverID := "local-driver-" + slug
		_, _ = s.RegisterDriver(driverID, slug)
	}
}

func (s *Service) GetDriver(driverID string) *pb.Driver {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if d, ok := s.drivers[driverID]; ok {
		return d.Driver
	}
	return nil
}

func IsLocalSeededDriver(driverID string) bool {
	return strings.HasPrefix(driverID, "local-driver-")
}

// UnregisterDriver marks the driver offline and clears Redis presence.
// Durable account rows are kept.
func (s *Service) UnregisterDriver(driverId string) {
	s.mu.Lock()
	if d, ok := s.drivers[driverId]; ok {
		d.Online = false
		d.Busy = false
		if IsLocalSeededDriver(driverId) {
			delete(s.drivers, driverId)
		}
	}
	s.mu.Unlock()

	if s.accounts != nil && !IsLocalSeededDriver(driverId) {
		_ = s.accounts.SetAvailability(context.Background(), driverId, "offline")
	}

	if s.locations != nil {
		if err := s.locations.RemoveDriver(context.Background(), driverId); err != nil {
			log.Printf("Failed to remove driver %s from tracking: %v", driverId, err)
		}
	}
}
