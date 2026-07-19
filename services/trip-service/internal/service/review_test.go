package service_test

import (
	"context"
	"errors"
	"testing"

	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	pbd "ride-sharing/shared/proto/driver"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type memBonus struct {
	points map[string]int
}

func (m *memBonus) AddBonusPoints(ctx context.Context, driverID string, points int) error {
	if m.points == nil {
		m.points = map[string]int{}
	}
	m.points[driverID] += points
	return nil
}

func (m *memBonus) GetBonusPoints(ctx context.Context, driverID string) (int, error) {
	return m.points[driverID], nil
}

func TestBonusPointsForRating(t *testing.T) {
	pts, err := domain.BonusPointsForRating(4)
	if err != nil || pts != 4 {
		t.Fatalf("want 4, got %d %v", pts, err)
	}
	if _, err := domain.BonusPointsForRating(0); !errors.Is(err, domain.ErrInvalidRating) {
		t.Fatalf("expected invalid rating, got %v", err)
	}
	if _, err := domain.BonusPointsForRating(6); !errors.Is(err, domain.ErrInvalidRating) {
		t.Fatalf("expected invalid rating, got %v", err)
	}
}

func payTripWithDriver(t *testing.T, repo domain.TripRepository, tripID, driverID string) {
	t.Helper()
	ctx := context.Background()
	driver := &pbd.Driver{Id: driverID, Name: "D"}
	for _, st := range []string{"accepted", "en_route", "arrived", "in_progress", "completed"} {
		d := (*pbd.Driver)(nil)
		if st == "accepted" {
			d = driver
		}
		if _, err := repo.TransitionTrip(ctx, tripID, st, domain.AllowedFrom[st], d); err != nil {
			t.Fatalf("transition %s: %v", st, err)
		}
	}
	ok, err := repo.MarkPaymentDone(ctx, tripID)
	if err != nil || !ok {
		t.Fatalf("mark paid: ok=%v err=%v", ok, err)
	}
}

func TestSubmitReviewOnceAndBonus(t *testing.T) {
	repo := repository.NewInmemRepository()
	bonus := &memBonus{points: map[string]int{}}
	svc := service.NewServiceWithBonus(repo, bonus)
	ctx := context.Background()

	fare := &domain.RideFareModel{
		ID:                primitive.NewObjectID(),
		UserID:            "rider-1",
		PackageSlug:       "sedan",
		TotalPriceInCents: 20000,
	}
	_ = repo.SaveRideFare(ctx, fare)
	trip, err := svc.CreateTrip(ctx, fare)
	if err != nil {
		t.Fatal(err)
	}
	payTripWithDriver(t, repo, trip.ID.Hex(), "drv-1")

	rev, err := svc.SubmitReview(ctx, trip.ID.Hex(), "rider-1", 5, "Great ride")
	if err != nil {
		t.Fatal(err)
	}
	if rev.BonusPoints != 5 {
		t.Fatalf("bonus want 5 got %d", rev.BonusPoints)
	}
	if bonus.points["drv-1"] != 5 {
		t.Fatalf("driver bonus want 5 got %d", bonus.points["drv-1"])
	}

	_, err = svc.SubmitReview(ctx, trip.ID.Hex(), "rider-1", 4, "again")
	if !errors.Is(err, domain.ErrReviewExists) {
		t.Fatalf("expected review exists, got %v", err)
	}
}

func TestDriverDashboardAggregation(t *testing.T) {
	repo := repository.NewInmemRepository()
	bonus := &memBonus{points: map[string]int{}}
	svc := service.NewServiceWithBonus(repo, bonus)
	ctx := context.Background()

	fare := &domain.RideFareModel{
		ID: primitive.NewObjectID(), UserID: "rider-1", PackageSlug: "sedan", TotalPriceInCents: 10000,
	}
	_ = repo.SaveRideFare(ctx, fare)
	trip, _ := svc.CreateTrip(ctx, fare)
	payTripWithDriver(t, repo, trip.ID.Hex(), "drv-1")

	if _, err := svc.SubmitReview(ctx, trip.ID.Hex(), "rider-1", 4, "ok"); err != nil {
		t.Fatal(err)
	}

	dash, err := svc.DriverDashboard(ctx, "drv-1")
	if err != nil {
		t.Fatal(err)
	}
	if dash.TripCount < 1 {
		t.Fatalf("expected trips, got %d", dash.TripCount)
	}
	if dash.BonusPoints != 4 {
		t.Fatalf("bonus want 4 got %d", dash.BonusPoints)
	}
	if dash.AverageRating != 4 {
		t.Fatalf("avg want 4 got %v", dash.AverageRating)
	}
	if len(dash.RecentReviews) != 1 {
		t.Fatalf("reviews want 1 got %d", len(dash.RecentReviews))
	}
}

func TestSubmitReviewRequiresPaid(t *testing.T) {
	repo := repository.NewInmemRepository()
	svc := service.NewService(repo)
	ctx := context.Background()
	fare := &domain.RideFareModel{
		ID: primitive.NewObjectID(), UserID: "rider-1", PackageSlug: "sedan", TotalPriceInCents: 10000,
	}
	_ = repo.SaveRideFare(ctx, fare)
	trip, _ := svc.CreateTrip(ctx, fare)
	_, err := svc.SubmitReview(ctx, trip.ID.Hex(), "rider-1", 5, "")
	if !errors.Is(err, domain.ErrTripNotPaid) {
		t.Fatalf("expected not paid, got %v", err)
	}
}
