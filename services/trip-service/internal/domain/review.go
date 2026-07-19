package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ReviewModel is a rider's post-payment rating of a completed trip.
type ReviewModel struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TripID      string             `bson:"tripID" json:"tripID"`
	UserID      string             `bson:"userID" json:"userID"`
	DriverID    string             `bson:"driverID" json:"driverID"`
	Rating      int                `bson:"rating" json:"rating"`
	Comment     string             `bson:"comment,omitempty" json:"comment,omitempty"`
	BonusPoints int                `bson:"bonusPoints" json:"bonusPoints"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
}

// BonusPointsForRating awards 1–5 points equal to the star rating.
func BonusPointsForRating(rating int) (int, error) {
	if rating < 1 || rating > 5 {
		return 0, ErrInvalidRating
	}
	return rating, nil
}

var ErrInvalidRating = errString("rating must be between 1 and 5")
var ErrReviewExists = errString("trip already reviewed")
var ErrTripNotPaid = errString("trip must be paid before review")
var ErrNotTripOwner = errString("not trip owner")
var ErrNoDriver = errString("trip has no assigned driver")

type errString string

func (e errString) Error() string { return string(e) }

// DriverDashboard aggregates trips, reviews, and bonus points for a driver.
type DriverDashboard struct {
	TripCount       int            `json:"tripCount"`
	BonusPoints     int            `json:"bonusPoints"`
	AverageRating   float64        `json:"averageRating"`
	RecentTrips     []*TripModel   `json:"recentTrips"`
	RecentReviews   []*ReviewModel `json:"recentReviews"`
}
