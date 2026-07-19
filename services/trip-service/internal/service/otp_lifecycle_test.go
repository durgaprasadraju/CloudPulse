package service_test

import (
	"context"
	"testing"
	"time"

	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/otp"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func setup() (domain.TripService, *domain.TripModel) {
	repo := repository.NewInmemRepository()
	svc := service.NewService(repo)
	fare := &domain.RideFareModel{
		ID:                primitive.NewObjectID(),
		UserID:            "rider-1",
		PackageSlug:       "sedan",
		TotalPriceInCents: 15000,
	}
	_ = repo.SaveRideFare(context.Background(), fare)
	trip, err := svc.CreateTrip(context.Background(), fare)
	if err != nil {
		panic(err)
	}
	return svc, trip
}

func TestOTPIssueAndVerify(t *testing.T) {
	svc, trip := setup()
	ctx := context.Background()

	if _, err := svc.TransitionTrip(ctx, trip.ID.Hex(), "accepted", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TransitionTrip(ctx, trip.ID.Hex(), "en_route", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TransitionTrip(ctx, trip.ID.Hex(), "arrived", nil); err != nil {
		t.Fatal(err)
	}

	code, err := svc.IssueOTP(ctx, trip.ID.Hex())
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Fatalf("otp length %d", len(code))
	}

	if _, err := svc.VerifyOTPAndStart(ctx, trip.ID.Hex(), "000000", "drv"); err == nil {
		t.Fatal("wrong OTP should fail")
	}

	started, err := svc.VerifyOTPAndStart(ctx, trip.ID.Hex(), code, "drv")
	if err != nil {
		t.Fatal(err)
	}
	if started.Status != "in_progress" {
		t.Fatalf("status=%s", started.Status)
	}
}

func TestCancelBlocksStart(t *testing.T) {
	svc, trip := setup()
	ctx := context.Background()

	if _, err := svc.TransitionTrip(ctx, trip.ID.Hex(), "accepted", nil); err != nil {
		t.Fatal(err)
	}
	code, err := svc.IssueOTP(ctx, trip.ID.Hex())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelTrip(ctx, trip.ID.Hex(), "rider-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.VerifyOTPAndStart(ctx, trip.ID.Hex(), code, "drv"); err == nil {
		t.Fatal("verify after cancel must fail")
	}
}

func TestCancelAfterStartRejected(t *testing.T) {
	svc, trip := setup()
	ctx := context.Background()

	_, _ = svc.TransitionTrip(ctx, trip.ID.Hex(), "accepted", nil)
	_, _ = svc.TransitionTrip(ctx, trip.ID.Hex(), "en_route", nil)
	_, _ = svc.TransitionTrip(ctx, trip.ID.Hex(), "arrived", nil)
	code, _ := svc.IssueOTP(ctx, trip.ID.Hex())
	if _, err := svc.VerifyOTPAndStart(ctx, trip.ID.Hex(), code, "drv"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelTrip(ctx, trip.ID.Hex(), "rider-1"); err == nil {
		t.Fatal("cancel after start must fail")
	}
}

func TestCompleteRequiresInProgress(t *testing.T) {
	svc, trip := setup()
	ctx := context.Background()
	if _, err := svc.CompleteTrip(ctx, trip.ID.Hex()); err == nil {
		t.Fatal("complete from pending must fail")
	}
}

func TestOTPExpiry(t *testing.T) {
	repo := repository.NewInmemRepository()
	svc := service.NewService(repo)
	ctx := context.Background()
	fare := &domain.RideFareModel{
		ID:                primitive.NewObjectID(),
		UserID:            "rider-1",
		PackageSlug:       "sedan",
		TotalPriceInCents: 1000,
	}
	_ = repo.SaveRideFare(ctx, fare)
	trip, _ := svc.CreateTrip(ctx, fare)
	_, _ = svc.TransitionTrip(ctx, trip.ID.Hex(), "accepted", nil)
	_, _ = svc.TransitionTrip(ctx, trip.ID.Hex(), "en_route", nil)
	_, _ = svc.TransitionTrip(ctx, trip.ID.Hex(), "arrived", nil)

	code := "123456"
	_ = repo.SetOTP(ctx, trip.ID.Hex(), otp.Hash(code), time.Now().UTC().Add(-time.Minute))
	if _, err := svc.VerifyOTPAndStart(ctx, trip.ID.Hex(), code, "drv"); err == nil {
		t.Fatal("expired OTP must fail")
	}
}

func TestOwnerMismatchCancel(t *testing.T) {
	svc, trip := setup()
	ctx := context.Background()
	_, _ = svc.TransitionTrip(ctx, trip.ID.Hex(), "accepted", nil)
	if _, err := svc.CancelTrip(ctx, trip.ID.Hex(), "someone-else"); err == nil {
		t.Fatal("non-owner cancel must fail")
	}
}
