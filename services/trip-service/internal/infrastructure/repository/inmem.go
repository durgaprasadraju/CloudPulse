package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ride-sharing/services/trip-service/internal/domain"
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type inmemRepository struct {
	mu        sync.Mutex
	trips     map[string]*domain.TripModel
	rideFares map[string]*domain.RideFareModel
	reviews   map[string]*domain.ReviewModel // keyed by tripID
}

func NewInmemRepository() *inmemRepository {
	return &inmemRepository{
		trips:     make(map[string]*domain.TripModel),
		rideFares: make(map[string]*domain.RideFareModel),
		reviews:   make(map[string]*domain.ReviewModel),
	}
}

func (r *inmemRepository) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	trip, ok := r.trips[id]
	if !ok {
		return nil, nil
	}
	cp := *trip
	return &cp, nil
}

func (r *inmemRepository) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	from := domain.AllowedFrom[status]
	_, err := r.TransitionTrip(ctx, tripID, status, from, driver)
	return err
}

func (r *inmemRepository) TransitionTrip(ctx context.Context, tripID, toStatus string, fromStatuses []string, driver *pbd.Driver) (*domain.TripModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	trip, ok := r.trips[tripID]
	if !ok {
		return nil, fmt.Errorf("trip not found with ID: %s", tripID)
	}
	if len(fromStatuses) > 0 {
		allowed := false
		for _, s := range fromStatuses {
			if trip.Status == s {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("illegal transition to %s for trip %s (current=%s)", toStatus, tripID, trip.Status)
		}
	}
	trip.Status = toStatus
	if toStatus == "completed" {
		now := time.Now().UTC()
		trip.CompletedAt = &now
	}
	if driver != nil {
		trip.Driver = &pb.TripDriver{
			Id:             driver.Id,
			Name:           driver.Name,
			CarPlate:       driver.CarPlate,
			ProfilePicture: driver.ProfilePicture,
		}
	}
	cp := *trip
	return &cp, nil
}

func (r *inmemRepository) SetOTP(ctx context.Context, tripID, otpHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	trip, ok := r.trips[tripID]
	if !ok {
		return fmt.Errorf("trip not found: %s", tripID)
	}
	trip.OTPHash = otpHash
	trip.OTPExpiresAt = expiresAt
	trip.OTPAttempts = 0
	return nil
}

func (r *inmemRepository) IncrementOTPAttempts(ctx context.Context, tripID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	trip, ok := r.trips[tripID]
	if !ok {
		return 0, fmt.Errorf("trip not found: %s", tripID)
	}
	trip.OTPAttempts++
	return trip.OTPAttempts, nil
}

func (r *inmemRepository) MarkOTPVerified(ctx context.Context, tripID string) (*domain.TripModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	trip, ok := r.trips[tripID]
	if !ok {
		return nil, fmt.Errorf("trip not found: %s", tripID)
	}
	if trip.Status != "arrived" {
		return nil, fmt.Errorf("trip not arrived or already started: %s", tripID)
	}
	now := time.Now().UTC()
	trip.Status = "in_progress"
	trip.OTPVerifiedAt = &now
	cp := *trip
	return &cp, nil
}

func (r *inmemRepository) MarkPaymentDone(ctx context.Context, tripID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	trip, ok := r.trips[tripID]
	if !ok {
		return false, fmt.Errorf("trip not found: %s", tripID)
	}
	if trip.Status != "completed" || trip.PaymentDone {
		return false, nil
	}
	trip.PaymentDone = true
	trip.Status = "payed"
	return true, nil
}

func (r *inmemRepository) GetRideFareByID(ctx context.Context, id string) (*domain.RideFareModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fare, exist := r.rideFares[id]
	if !exist {
		return nil, fmt.Errorf("fare does not exist with ID: %s", id)
	}

	return fare, nil
}

func (r *inmemRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trips[trip.ID.Hex()] = trip
	return trip, nil
}

func (r *inmemRepository) SaveRideFare(ctx context.Context, f *domain.RideFareModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rideFares[f.ID.Hex()] = f
	return nil
}

func (r *inmemRepository) CreateReview(ctx context.Context, review *domain.ReviewModel) (*domain.ReviewModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.reviews[review.TripID]; exists {
		return nil, domain.ErrReviewExists
	}
	if review.ID.IsZero() {
		review.ID = primitive.NewObjectID()
	}
	if review.CreatedAt.IsZero() {
		review.CreatedAt = time.Now().UTC()
	}
	cp := *review
	r.reviews[review.TripID] = &cp
	return &cp, nil
}

func (r *inmemRepository) GetReviewByTripID(ctx context.Context, tripID string) (*domain.ReviewModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rev, ok := r.reviews[tripID]
	if !ok {
		return nil, nil
	}
	cp := *rev
	return &cp, nil
}

func (r *inmemRepository) ListReviewsByDriver(ctx context.Context, driverID string, limit int64) ([]*domain.ReviewModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.ReviewModel
	for _, rev := range r.reviews {
		if rev.DriverID == driverID {
			cp := *rev
			out = append(out, &cp)
		}
	}
	if limit > 0 && int64(len(out)) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *inmemRepository) ListTripsByDriver(ctx context.Context, driverID string, statuses []string, limit int64) ([]*domain.TripModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	statusSet := map[string]bool{}
	for _, s := range statuses {
		statusSet[s] = true
	}
	var out []*domain.TripModel
	for _, trip := range r.trips {
		if trip.Driver == nil || trip.Driver.Id != driverID {
			continue
		}
		if len(statusSet) > 0 && !statusSet[trip.Status] {
			continue
		}
		cp := *trip
		out = append(out, &cp)
	}
	if limit > 0 && int64(len(out)) > limit {
		out = out[:limit]
	}
	return out, nil
}
