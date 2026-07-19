package domain

import (
	"context"
	"time"

	"ride-sharing/shared/types"

	tripTypes "ride-sharing/services/trip-service/pkg/types"
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Allowed status transitions for CAS updates.
var AllowedFrom = map[string][]string{
	"accepted":   {"pending"},
	"en_route":   {"accepted"},
	"arrived":    {"en_route", "accepted"},
	"in_progress": {"arrived"},
	"completed":  {"in_progress"},
	"payed":      {"completed"},
	"cancelled":  {"pending", "accepted", "en_route", "arrived"},
}

type TripModel struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        string             `bson:"userID"`
	Status        string             `bson:"status"`
	RideFare      *RideFareModel     `bson:"rideFare"`
	Driver        *pb.TripDriver     `bson:"driver"`
	OTPHash       string             `bson:"otpHash,omitempty"`
	OTPExpiresAt  time.Time          `bson:"otpExpiresAt,omitempty"`
	OTPAttempts   int                `bson:"otpAttempts,omitempty"`
	OTPVerifiedAt *time.Time         `bson:"otpVerifiedAt,omitempty"`
	PaymentDone   bool               `bson:"paymentDone,omitempty"`
	CompletedAt   *time.Time         `bson:"completedAt,omitempty"`
}

func (t *TripModel) ToProto() *pb.Trip {
	var selectedFare *pb.RideFare
	var route *pb.Route
	if t.RideFare != nil {
		selectedFare = t.RideFare.ToProto()
		if t.RideFare.Route != nil {
			route = t.RideFare.Route.ToProto()
		}
	}
	return &pb.Trip{
		Id:           t.ID.Hex(),
		UserID:       t.UserID,
		SelectedFare: selectedFare,
		Status:       t.Status,
		Driver:       t.Driver,
		Route:        route,
	}
}

type TripRepository interface {
	CreateTrip(ctx context.Context, trip *TripModel) (*TripModel, error)
	SaveRideFare(ctx context.Context, f *RideFareModel) error
	GetRideFareByID(ctx context.Context, id string) (*RideFareModel, error)
	GetTripByID(ctx context.Context, id string) (*TripModel, error)
	UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error
	TransitionTrip(ctx context.Context, tripID, toStatus string, fromStatuses []string, driver *pbd.Driver) (*TripModel, error)
	SetOTP(ctx context.Context, tripID, otpHash string, expiresAt time.Time) error
	IncrementOTPAttempts(ctx context.Context, tripID string) (int, error)
	MarkOTPVerified(ctx context.Context, tripID string) (*TripModel, error)
	MarkPaymentDone(ctx context.Context, tripID string) (bool, error)
	CreateReview(ctx context.Context, review *ReviewModel) (*ReviewModel, error)
	GetReviewByTripID(ctx context.Context, tripID string) (*ReviewModel, error)
	ListReviewsByDriver(ctx context.Context, driverID string, limit int64) ([]*ReviewModel, error)
	ListTripsByDriver(ctx context.Context, driverID string, statuses []string, limit int64) ([]*TripModel, error)
}

type TripService interface {
	CreateTrip(ctx context.Context, fare *RideFareModel) (*TripModel, error)
	GetRoute(ctx context.Context, pickup, destination *types.Coordinate, useOsrmApi bool) (*tripTypes.OsrmApiResponse, error)
	EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*RideFareModel
	GenerateTripFares(
		ctx context.Context,
		fares []*RideFareModel,
		userID string,
		Route *tripTypes.OsrmApiResponse,
	) ([]*RideFareModel, error)
	GetAndValidateFare(ctx context.Context, fareID, userID string) (*RideFareModel, error)
	GetTripByID(ctx context.Context, id string) (*TripModel, error)
	UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error
	TransitionTrip(ctx context.Context, tripID, toStatus string, driver *pbd.Driver) (*TripModel, error)
	IssueOTP(ctx context.Context, tripID string) (string, error)
	VerifyOTPAndStart(ctx context.Context, tripID, code, driverID string) (*TripModel, error)
	CancelTrip(ctx context.Context, tripID, userID string) (*TripModel, error)
	CompleteTrip(ctx context.Context, tripID string) (*TripModel, error)
	MarkPaid(ctx context.Context, tripID string) error
	SubmitReview(ctx context.Context, tripID, userID string, rating int, comment string) (*ReviewModel, error)
	DriverDashboard(ctx context.Context, driverID string) (*DriverDashboard, error)
	DriverTrips(ctx context.Context, driverID string) ([]*TripModel, error)
}
