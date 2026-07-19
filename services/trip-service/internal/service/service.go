package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"ride-sharing/services/trip-service/internal/domain"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	"ride-sharing/shared/env"
	"ride-sharing/shared/otp"
	pbd "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type service struct {
	repo   domain.TripRepository
	bonus  BonusPointsUpdater
}

// BonusPointsUpdater credits driver bonus points after a review (e.g. Postgres).
type BonusPointsUpdater interface {
	AddBonusPoints(ctx context.Context, driverID string, points int) error
	GetBonusPoints(ctx context.Context, driverID string) (int, error)
}

func NewService(repo domain.TripRepository) *service {
	return &service{repo: repo}
}

func NewServiceWithBonus(repo domain.TripRepository, bonus BonusPointsUpdater) *service {
	return &service{repo: repo, bonus: bonus}
}

func (s *service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	t := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
		Driver:   &trip.TripDriver{},
	}

	return s.repo.CreateTrip(ctx, t)
}

func (s *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate, useOSRMApi bool) (*tripTypes.OsrmApiResponse, error) {
	if !useOSRMApi {
		// Return a simple mock response in case we don't want to rely on an external API
		return &tripTypes.OsrmApiResponse{
			Routes: []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
				Geometry struct {
					Coordinates [][]float64 `json:"coordinates"`
				} `json:"geometry"`
			}{
				{
					Distance: 5.0, // 5km
					Duration: 600, // 10 minutes
					Geometry: struct {
						Coordinates [][]float64 `json:"coordinates"`
					}{
						Coordinates: [][]float64{
							{pickup.Latitude, pickup.Longitude},
							{destination.Latitude, destination.Longitude},
						},
					},
				},
			},
		}, nil
	}

	// or use our self hosted API (check the course lesson: "Preparing for External API Failures")
	baseURL := env.GetString("OSRM_API", "http://router.project-osrm.org")

	url := fmt.Sprintf(
		"%s/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson",
		baseURL,
		pickup.Longitude, pickup.Latitude,
		destination.Longitude, destination.Latitude,
	)

	log.Printf("Started Fetching from OSRM API: URL: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch route from OSRM API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response: %v", err)
	}

	log.Printf("Got response from OSRM API %s", string(body))

	var routeResp tripTypes.OsrmApiResponse
	if err := json.Unmarshal(body, &routeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &routeResp, nil
}

func (s *service) EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*domain.RideFareModel {
	baseFares := getBaseFares()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))

	for i, f := range baseFares {
		estimatedFares[i] = estimateFareRoute(f, route)
	}

	return estimatedFares
}

func (s *service) GenerateTripFares(ctx context.Context, rideFares []*domain.RideFareModel, userID string, route *tripTypes.OsrmApiResponse) ([]*domain.RideFareModel, error) {
	fares := make([]*domain.RideFareModel, len(rideFares))

	for i, f := range rideFares {
		id := primitive.NewObjectID()

		fare := &domain.RideFareModel{
			UserID:            userID,
			ID:                id,
			TotalPriceInCents: f.TotalPriceInCents,
			PackageSlug:       f.PackageSlug,
			Route:             route,
		}

		if err := s.repo.SaveRideFare(ctx, fare); err != nil {
			return nil, fmt.Errorf("failed to save trip fare: %w", err)
		}

		fares[i] = fare
	}

	return fares, nil
}

func (s *service) GetAndValidateFare(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {
	fare, err := s.repo.GetRideFareByID(ctx, fareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip fare: %w", err)
	}

	if fare == nil {
		return nil, fmt.Errorf("fare does not exist")
	}

	// User fare validation (user is owner of this fare?)
	if userID != fare.UserID {
		return nil, fmt.Errorf("fare does not belong to the user")
	}

	return fare, nil
}

func estimateFareRoute(f *domain.RideFareModel, route *tripTypes.OsrmApiResponse) *domain.RideFareModel {
	pricingCfg := tripTypes.DefaultPricingConfig()
	carPackagePrice := f.TotalPriceInCents

	distanceKm := route.Routes[0].Distance
	durationInMinutes := route.Routes[0].Duration

	distanceFare := distanceKm * pricingCfg.PricePerUnitOfDistance
	timeFare := durationInMinutes * pricingCfg.PricingPerMinute
	totalPrice := carPackagePrice + distanceFare + timeFare

	return &domain.RideFareModel{
		TotalPriceInCents: totalPrice,
		PackageSlug:       f.PackageSlug,
	}
}

func getBaseFares() []*domain.RideFareModel {
	// Values are paise (1/100 INR), shown as ₹ on the web UI.
	return []*domain.RideFareModel{
		{
			PackageSlug:       "suv",
			TotalPriceInCents: 8000,
		},
		{
			PackageSlug:       "sedan",
			TotalPriceInCents: 10000,
		},
		{
			PackageSlug:       "van",
			TotalPriceInCents: 12000,
		},
		{
			PackageSlug:       "luxury",
			TotalPriceInCents: 28000,
		},
	}
}

func (s *service) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	return s.repo.GetTripByID(ctx, id)
}

func (s *service) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	return s.repo.UpdateTrip(ctx, tripID, status, driver)
}

func (s *service) TransitionTrip(ctx context.Context, tripID, toStatus string, driver *pbd.Driver) (*domain.TripModel, error) {
	from := domain.AllowedFrom[toStatus]
	return s.repo.TransitionTrip(ctx, tripID, toStatus, from, driver)
}

func (s *service) IssueOTP(ctx context.Context, tripID string) (string, error) {
	code, err := otp.Generate(otp.DefaultLength)
	if err != nil {
		return "", err
	}
	expires := time.Now().UTC().Add(otp.DefaultTTL)
	if err := s.repo.SetOTP(ctx, tripID, otp.Hash(code), expires); err != nil {
		return "", err
	}
	return code, nil
}

func (s *service) VerifyOTPAndStart(ctx context.Context, tripID, code, driverID string) (*domain.TripModel, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil || trip == nil {
		return nil, fmt.Errorf("trip not found")
	}
	if trip.Status != "arrived" {
		return nil, fmt.Errorf("trip must be arrived before OTP start (status=%s)", trip.Status)
	}
	if driverID != "" && trip.Driver != nil && trip.Driver.Id != "" && trip.Driver.Id != driverID {
		return nil, fmt.Errorf("driver mismatch")
	}
	if trip.OTPVerifiedAt != nil {
		return nil, fmt.Errorf("OTP already verified")
	}
	if time.Now().UTC().After(trip.OTPExpiresAt) {
		return nil, fmt.Errorf("OTP expired")
	}
	if trip.OTPAttempts >= otp.DefaultMaxAttempts {
		return nil, fmt.Errorf("too many OTP attempts")
	}
	if !otp.Verify(code, trip.OTPHash) {
		attempts, _ := s.repo.IncrementOTPAttempts(ctx, tripID)
		remaining := otp.DefaultMaxAttempts - attempts
		if remaining < 0 {
			remaining = 0
		}
		return nil, fmt.Errorf("invalid OTP (%d attempts remaining)", remaining)
	}
	return s.repo.MarkOTPVerified(ctx, tripID)
}

func (s *service) CancelTrip(ctx context.Context, tripID, userID string) (*domain.TripModel, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil || trip == nil {
		return nil, fmt.Errorf("trip not found")
	}
	if userID != "" && trip.UserID != userID {
		return nil, fmt.Errorf("not trip owner")
	}
	return s.repo.TransitionTrip(ctx, tripID, "cancelled", domain.AllowedFrom["cancelled"], nil)
}

func (s *service) CompleteTrip(ctx context.Context, tripID string) (*domain.TripModel, error) {
	return s.repo.TransitionTrip(ctx, tripID, "completed", domain.AllowedFrom["completed"], nil)
}

func (s *service) MarkPaid(ctx context.Context, tripID string) error {
	_, err := s.repo.MarkPaymentDone(ctx, tripID)
	return err
}

func (s *service) SubmitReview(ctx context.Context, tripID, userID string, rating int, comment string) (*domain.ReviewModel, error) {
	bonus, err := domain.BonusPointsForRating(rating)
	if err != nil {
		return nil, err
	}
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil || trip == nil {
		return nil, fmt.Errorf("trip not found")
	}
	if trip.Status != "payed" {
		return nil, domain.ErrTripNotPaid
	}
	if userID != "" && trip.UserID != userID {
		return nil, domain.ErrNotTripOwner
	}
	if trip.Driver == nil || trip.Driver.Id == "" {
		return nil, domain.ErrNoDriver
	}
	existing, err := s.repo.GetReviewByTripID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrReviewExists
	}

	review := &domain.ReviewModel{
		TripID:      tripID,
		UserID:      trip.UserID,
		DriverID:    trip.Driver.Id,
		Rating:      rating,
		Comment:     comment,
		BonusPoints: bonus,
	}
	created, err := s.repo.CreateReview(ctx, review)
	if err != nil {
		return nil, err
	}
	if s.bonus != nil {
		if err := s.bonus.AddBonusPoints(ctx, trip.Driver.Id, bonus); err != nil {
			log.Printf("failed to credit bonus points for driver %s: %v", trip.Driver.Id, err)
		}
	}
	return created, nil
}

func (s *service) DriverTrips(ctx context.Context, driverID string) ([]*domain.TripModel, error) {
	return s.repo.ListTripsByDriver(ctx, driverID, []string{"completed", "payed"}, 50)
}

func (s *service) DriverDashboard(ctx context.Context, driverID string) (*domain.DriverDashboard, error) {
	trips, err := s.repo.ListTripsByDriver(ctx, driverID, []string{"completed", "payed"}, 20)
	if err != nil {
		return nil, err
	}
	reviews, err := s.repo.ListReviewsByDriver(ctx, driverID, 20)
	if err != nil {
		return nil, err
	}

	var sum float64
	for _, r := range reviews {
		sum += float64(r.Rating)
	}
	avg := 0.0
	if len(reviews) > 0 {
		avg = sum / float64(len(reviews))
	}

	bonusPoints := 0
	if s.bonus != nil {
		if pts, err := s.bonus.GetBonusPoints(ctx, driverID); err == nil {
			bonusPoints = pts
		}
	} else {
		for _, r := range reviews {
			bonusPoints += r.BonusPoints
		}
	}

	return &domain.DriverDashboard{
		TripCount:     len(trips),
		BonusPoints:   bonusPoints,
		AverageRating: avg,
		RecentTrips:   trips,
		RecentReviews: reviews,
	}, nil
}
