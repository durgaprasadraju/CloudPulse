package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var (
	ErrEmailTaken      = errors.New("email already registered")
	ErrInvalidLogin    = errors.New("invalid email or password")
	ErrDriverNotFound  = errors.New("driver not found")
	ErrValidation      = errors.New("validation failed")
)

type Account struct {
	ID             string
	Email          string
	PasswordHash   string
	Name           string
	Phone          string
	PackageSlug    string
	CarPlate       string
	ProfilePicture string
	Geohash        string
	Latitude       float64
	Longitude      float64
	Availability   string // offline | available | offered | busy
	BonusPoints    int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AccountStore struct {
	db *sql.DB
}

func NewAccountStore(databaseURL string) (*AccountStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	store := &AccountStore{db: db}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *AccountStore) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS drivers (
			id               TEXT PRIMARY KEY,
			name             TEXT NOT NULL,
			package_slug     TEXT NOT NULL,
			car_plate        TEXT NOT NULL,
			profile_picture  TEXT NOT NULL DEFAULT '',
			geohash          TEXT NOT NULL DEFAULT '',
			latitude         DOUBLE PRECISION NOT NULL DEFAULT 0,
			longitude        DOUBLE PRECISION NOT NULL DEFAULT 0,
			updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		ALTER TABLE drivers ADD COLUMN IF NOT EXISTS email TEXT;
		ALTER TABLE drivers ADD COLUMN IF NOT EXISTS password_hash TEXT;
		ALTER TABLE drivers ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '';
		ALTER TABLE drivers ADD COLUMN IF NOT EXISTS availability TEXT NOT NULL DEFAULT 'offline';
		ALTER TABLE drivers ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
		ALTER TABLE drivers ADD COLUMN IF NOT EXISTS bonus_points INTEGER NOT NULL DEFAULT 0;
		CREATE UNIQUE INDEX IF NOT EXISTS drivers_email_unique
			ON drivers (lower(email))
			WHERE email IS NOT NULL AND email <> '';
	`)
	return err
}

type RegisterInput struct {
	Email       string
	Password    string
	Name        string
	Phone       string
	PackageSlug string
	CarPlate    string
}

func (s *AccountStore) Register(ctx context.Context, in RegisterInput) (*Account, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	in.Phone = strings.TrimSpace(in.Phone)
	in.PackageSlug = strings.TrimSpace(strings.ToLower(in.PackageSlug))
	in.CarPlate = strings.ToUpper(strings.TrimSpace(in.CarPlate))

	if in.Email == "" || in.Password == "" || in.Name == "" || in.PackageSlug == "" || in.CarPlate == "" {
		return nil, fmt.Errorf("%w: email, password, name, packageSlug and carPlate are required", ErrValidation)
	}
	if len(in.Password) < 6 {
		return nil, fmt.Errorf("%w: password must be at least 6 characters", ErrValidation)
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	id := NewDriverID()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO drivers (
			id, email, password_hash, name, phone, package_slug, car_plate,
			profile_picture, availability, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,'', 'offline', now(), now())`,
		id, in.Email, hash, in.Name, in.Phone, in.PackageSlug, in.CarPlate,
	)
	if err != nil {
		if strings.Contains(err.Error(), "drivers_email_unique") || strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *AccountStore) Authenticate(ctx context.Context, email, password string) (*Account, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	acc, err := s.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrDriverNotFound) {
			return nil, ErrInvalidLogin
		}
		return nil, err
	}
	if acc.PasswordHash == "" || !CheckPassword(acc.PasswordHash, password) {
		return nil, ErrInvalidLogin
	}
	return acc, nil
}

func (s *AccountStore) GetByID(ctx context.Context, id string) (*Account, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(email,''), COALESCE(password_hash,''), name, COALESCE(phone,''),
		       package_slug, car_plate, profile_picture, geohash, latitude, longitude,
		       COALESCE(availability,'offline'), COALESCE(bonus_points,0),
		       COALESCE(created_at, now()), updated_at
		FROM drivers WHERE id = $1`, id)
	return scanAccount(row)
}

func (s *AccountStore) GetByEmail(ctx context.Context, email string) (*Account, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(email,''), COALESCE(password_hash,''), name, COALESCE(phone,''),
		       package_slug, car_plate, profile_picture, geohash, latitude, longitude,
		       COALESCE(availability,'offline'), COALESCE(bonus_points,0),
		       COALESCE(created_at, now()), updated_at
		FROM drivers WHERE lower(email) = lower($1)`, email)
	return scanAccount(row)
}

func (s *AccountStore) SetAvailability(ctx context.Context, id, availability string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE drivers SET availability = $2, updated_at = now() WHERE id = $1`, id, availability)
	return err
}

func (s *AccountStore) AddBonusPoints(ctx context.Context, id string, points int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE drivers SET bonus_points = COALESCE(bonus_points,0) + $2, updated_at = now() WHERE id = $1`,
		id, points)
	return err
}

func (s *AccountStore) UpdateLocation(ctx context.Context, id string, lat, lon float64, geohash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE drivers SET latitude=$2, longitude=$3, geohash=$4, updated_at=now() WHERE id=$1`,
		id, lat, lon, geohash)
	return err
}

func (s *AccountStore) Close() error {
	return s.db.Close()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanAccount(row scannable) (*Account, error) {
	var a Account
	err := row.Scan(
		&a.ID, &a.Email, &a.PasswordHash, &a.Name, &a.Phone,
		&a.PackageSlug, &a.CarPlate, &a.ProfilePicture, &a.Geohash,
		&a.Latitude, &a.Longitude, &a.Availability, &a.BonusPoints, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDriverNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// PublicProfile omits password hash for API responses.
func (a *Account) PublicProfile() map[string]any {
	return map[string]any{
		"id":             a.ID,
		"email":          a.Email,
		"name":           a.Name,
		"phone":          a.Phone,
		"packageSlug":    a.PackageSlug,
		"carPlate":       a.CarPlate,
		"profilePicture": a.ProfilePicture,
		"availability":   a.Availability,
		"bonusPoints":    a.BonusPoints,
		"location": map[string]float64{
			"latitude":  a.Latitude,
			"longitude": a.Longitude,
		},
	}
}
