/*
Package tracking stores real-time driver locations in Redis.

Used by:
  - driver-service: writes location + profile on registration
  - api-gateway: updates location from driver.cmd.location WebSocket messages
    and queries nearby drivers for rider map streaming
*/
package tracking

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/mmcloughlin/geohash"
	"github.com/redis/go-redis/v9"
)

const (
	geoKey           = "drivers:locations"
	driverKeyPrefix  = "driver:location:"
	locationTTL      = 30 * time.Minute
	operationTimeout = 5 * time.Second
)

type LocationStore struct {
	client *redis.Client
}

// DriverLocation is the nearby-driver payload pushed to riders over WebSocket.
type DriverLocation struct {
	ID             string  `json:"id"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	Geohash        string  `json:"geohash"`
	Name           string  `json:"name"`
	ProfilePicture string  `json:"profilePicture"`
	CarPlate       string  `json:"carPlate"`
	PackageSlug    string  `json:"packageSlug"`
}

// NewLocationStore connects to Redis using a URL such as redis://redis:6379/0.
func NewLocationStore(redisURL string) (*LocationStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &LocationStore{client: client}, nil
}

// UpdateDriverLocation records the driver's latest position (keeps existing profile fields).
func (s *LocationStore) UpdateDriverLocation(ctx context.Context, driverID string, latitude, longitude float64) error {
	return s.UpsertDriver(ctx, DriverLocation{
		ID:        driverID,
		Latitude:  latitude,
		Longitude: longitude,
	})
}

// UpsertDriver writes location + optional profile metadata into Redis.
func (s *LocationStore) UpsertDriver(ctx context.Context, d DriverLocation) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	pipe := s.client.Pipeline()

	pipe.GeoAdd(ctx, geoKey, &redis.GeoLocation{
		Name:      d.ID,
		Latitude:  d.Latitude,
		Longitude: d.Longitude,
	})

	key := driverKeyPrefix + d.ID
	fields := map[string]interface{}{
		"latitude":   d.Latitude,
		"longitude":  d.Longitude,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
		"geohash":    geohash.Encode(d.Latitude, d.Longitude),
	}
	if d.Name != "" {
		fields["name"] = d.Name
	}
	if d.ProfilePicture != "" {
		fields["profilePicture"] = d.ProfilePicture
	}
	if d.CarPlate != "" {
		fields["carPlate"] = d.CarPlate
	}
	if d.PackageSlug != "" {
		fields["packageSlug"] = d.PackageSlug
	}

	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, locationTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}
	return nil
}

// RemoveDriver clears a driver's tracked location (e.g. on disconnect).
func (s *LocationStore) RemoveDriver(ctx context.Context, driverID string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	pipe := s.client.Pipeline()
	pipe.ZRem(ctx, geoKey, driverID)
	pipe.Del(ctx, driverKeyPrefix+driverID)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to remove driver location: %w", err)
	}
	return nil
}

// NearbyDrivers returns full driver records within radiusKm of the given point.
func (s *LocationStore) NearbyDrivers(ctx context.Context, latitude, longitude, radiusKm float64) ([]DriverLocation, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	ids, err := s.client.GeoSearch(ctx, geoKey, &redis.GeoSearchQuery{
		Latitude:   latitude,
		Longitude:  longitude,
		Radius:     radiusKm,
		RadiusUnit: "km",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to search nearby drivers: %w", err)
	}

	out := make([]DriverLocation, 0, len(ids))
	for _, id := range ids {
		vals, err := s.client.HGetAll(ctx, driverKeyPrefix+id).Result()
		if err != nil || len(vals) == 0 {
			continue
		}
		lat, _ := strconv.ParseFloat(vals["latitude"], 64)
		lon, _ := strconv.ParseFloat(vals["longitude"], 64)
		gh := vals["geohash"]
		if gh == "" {
			gh = geohash.Encode(lat, lon)
		}
		out = append(out, DriverLocation{
			ID:             id,
			Latitude:       lat,
			Longitude:      lon,
			Geohash:        gh,
			Name:           vals["name"],
			ProfilePicture: vals["profilePicture"],
			CarPlate:       vals["carPlate"],
			PackageSlug:    vals["packageSlug"],
		})
	}
	return out, nil
}

func (s *LocationStore) Close() error {
	return s.client.Close()
}
