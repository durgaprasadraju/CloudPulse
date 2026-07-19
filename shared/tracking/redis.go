/*
Package tracking stores real-time driver locations in Redis.

Used by:
  - driver-service: writes location on registration
  - api-gateway: updates location from driver.cmd.location WebSocket messages

Locations live in a Redis GEO set (drivers:locations) plus a per-driver hash
with the latest coordinates, so nearby-driver queries can be added later.
*/
package tracking

import (
	"context"
	"fmt"
	"time"

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

// UpdateDriverLocation records the driver's latest position.
func (s *LocationStore) UpdateDriverLocation(ctx context.Context, driverID string, latitude, longitude float64) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	pipe := s.client.Pipeline()

	pipe.GeoAdd(ctx, geoKey, &redis.GeoLocation{
		Name:      driverID,
		Latitude:  latitude,
		Longitude: longitude,
	})

	key := driverKeyPrefix + driverID
	pipe.HSet(ctx, key, map[string]interface{}{
		"latitude":   latitude,
		"longitude":  longitude,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
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

// NearbyDrivers returns driver IDs within radiusKm of the given point.
func (s *LocationStore) NearbyDrivers(ctx context.Context, latitude, longitude, radiusKm float64) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	res, err := s.client.GeoSearch(ctx, geoKey, &redis.GeoSearchQuery{
		Latitude:   latitude,
		Longitude:  longitude,
		Radius:     radiusKm,
		RadiusUnit: "km",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to search nearby drivers: %w", err)
	}
	return res, nil
}

func (s *LocationStore) Close() error {
	return s.client.Close()
}
