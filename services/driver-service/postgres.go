package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	pb "ride-sharing/shared/proto/driver"

	_ "github.com/lib/pq"
)

// DriverStore persists registered drivers in PostgreSQL so registrations
// survive driver-service restarts. Matches the Terraform-provisioned RDS
// instance (locally: the postgres container from docker-compose).
type DriverStore struct {
	db *sql.DB
}

func NewDriverStore(databaseURL string) (*DriverStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	store := &DriverStore{db: db}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *DriverStore) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS drivers (
			id              TEXT PRIMARY KEY,
			name            TEXT NOT NULL,
			package_slug    TEXT NOT NULL,
			car_plate       TEXT NOT NULL,
			profile_picture TEXT NOT NULL DEFAULT '',
			geohash         TEXT NOT NULL DEFAULT '',
			latitude        DOUBLE PRECISION NOT NULL DEFAULT 0,
			longitude       DOUBLE PRECISION NOT NULL DEFAULT 0,
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return fmt.Errorf("failed to create drivers table: %w", err)
	}
	return nil
}

func (s *DriverStore) SaveDriver(ctx context.Context, d *pb.Driver) error {
	var lat, lon float64
	if d.Location != nil {
		lat, lon = d.Location.Latitude, d.Location.Longitude
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO drivers (id, name, package_slug, car_plate, profile_picture, geohash, latitude, longitude, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (id) DO UPDATE SET
			package_slug = EXCLUDED.package_slug,
			geohash      = EXCLUDED.geohash,
			latitude     = EXCLUDED.latitude,
			longitude    = EXCLUDED.longitude,
			updated_at   = now()`,
		d.Id, d.Name, d.PackageSlug, d.CarPlate, d.ProfilePicture, d.Geohash, lat, lon)
	if err != nil {
		return fmt.Errorf("failed to save driver %s: %w", d.Id, err)
	}
	return nil
}

func (s *DriverStore) DeleteDriver(ctx context.Context, driverID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM drivers WHERE id = $1`, driverID)
	if err != nil {
		return fmt.Errorf("failed to delete driver %s: %w", driverID, err)
	}
	return nil
}

func (s *DriverStore) LoadDrivers(ctx context.Context) ([]*pb.Driver, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, package_slug, car_plate, profile_picture, geohash, latitude, longitude
		FROM drivers`)
	if err != nil {
		return nil, fmt.Errorf("failed to load drivers: %w", err)
	}
	defer rows.Close()

	var drivers []*pb.Driver
	for rows.Next() {
		var d pb.Driver
		var lat, lon float64
		if err := rows.Scan(&d.Id, &d.Name, &d.PackageSlug, &d.CarPlate, &d.ProfilePicture, &d.Geohash, &lat, &lon); err != nil {
			return nil, fmt.Errorf("failed to scan driver row: %w", err)
		}
		d.Location = &pb.Location{Latitude: lat, Longitude: lon}
		drivers = append(drivers, &d)
	}
	return drivers, rows.Err()
}

func (s *DriverStore) Close() {
	if err := s.db.Close(); err != nil {
		log.Printf("Failed to close postgres connection: %v", err)
	}
}
