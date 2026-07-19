package repository

import (
	"context"
	"fmt"
	"time"

	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/db"
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoRepository struct {
	db *mongo.Database
}

func NewMongoRepository(db *mongo.Database) *mongoRepository {
	return &mongoRepository{db: db}
}

func (r *mongoRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	result, err := r.db.Collection(db.TripsCollection).InsertOne(ctx, trip)
	if err != nil {
		return nil, err
	}

	trip.ID = result.InsertedID.(primitive.ObjectID)

	return trip, nil
}

func (r *mongoRepository) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	result := r.db.Collection(db.TripsCollection).FindOne(ctx, bson.M{"_id": _id})
	if result.Err() != nil {
		return nil, result.Err()
	}

	var trip domain.TripModel
	err = result.Decode(&trip)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}

func (r *mongoRepository) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	from := domain.AllowedFrom[status]
	_, err := r.TransitionTrip(ctx, tripID, status, from, driver)
	return err
}

func (r *mongoRepository) TransitionTrip(ctx context.Context, tripID, toStatus string, fromStatuses []string, driver *pbd.Driver) (*domain.TripModel, error) {
	_id, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": _id}
	if len(fromStatuses) > 0 {
		filter["status"] = bson.M{"$in": fromStatuses}
	}

	set := bson.M{"status": toStatus}
	if toStatus == "completed" {
		now := time.Now().UTC()
		set["completedAt"] = now
	}
	if driver != nil {
		set["driver"] = &pb.TripDriver{
			Id:             driver.Id,
			Name:           driver.Name,
			CarPlate:       driver.CarPlate,
			ProfilePicture: driver.ProfilePicture,
		}
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var trip domain.TripModel
	err = r.db.Collection(db.TripsCollection).FindOneAndUpdate(
		ctx,
		filter,
		bson.M{"$set": set},
		opts,
	).Decode(&trip)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("illegal transition to %s for trip %s", toStatus, tripID)
		}
		return nil, err
	}
	return &trip, nil
}

func (r *mongoRepository) SetOTP(ctx context.Context, tripID, otpHash string, expiresAt time.Time) error {
	_id, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return err
	}
	result, err := r.db.Collection(db.TripsCollection).UpdateOne(ctx,
		bson.M{"_id": _id},
		bson.M{"$set": bson.M{
			"otpHash":      otpHash,
			"otpExpiresAt": expiresAt,
			"otpAttempts":  0,
		}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("trip not found: %s", tripID)
	}
	return nil
}

func (r *mongoRepository) IncrementOTPAttempts(ctx context.Context, tripID string) (int, error) {
	_id, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return 0, err
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var trip domain.TripModel
	err = r.db.Collection(db.TripsCollection).FindOneAndUpdate(
		ctx,
		bson.M{"_id": _id},
		bson.M{"$inc": bson.M{"otpAttempts": 1}},
		opts,
	).Decode(&trip)
	if err != nil {
		return 0, err
	}
	return trip.OTPAttempts, nil
}

func (r *mongoRepository) MarkOTPVerified(ctx context.Context, tripID string) (*domain.TripModel, error) {
	_id, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var trip domain.TripModel
	err = r.db.Collection(db.TripsCollection).FindOneAndUpdate(
		ctx,
		bson.M{"_id": _id, "status": "arrived"},
		bson.M{"$set": bson.M{
			"status":        "in_progress",
			"otpVerifiedAt": now,
		}},
		opts,
	).Decode(&trip)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("trip not arrived or already started: %s", tripID)
		}
		return nil, err
	}
	return &trip, nil
}

func (r *mongoRepository) MarkPaymentDone(ctx context.Context, tripID string) (bool, error) {
	_id, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return false, err
	}
	result, err := r.db.Collection(db.TripsCollection).UpdateOne(ctx,
		bson.M{"_id": _id, "status": "completed", "paymentDone": bson.M{"$ne": true}},
		bson.M{"$set": bson.M{"paymentDone": true, "status": "payed"}},
	)
	if err != nil {
		return false, err
	}
	return result.ModifiedCount > 0, nil
}

func (r *mongoRepository) SaveRideFare(ctx context.Context, fare *domain.RideFareModel) error {
	result, err := r.db.Collection(db.RideFaresCollection).InsertOne(ctx, fare)
	if err != nil {
		return err
	}

	fare.ID = result.InsertedID.(primitive.ObjectID)

	return nil
}

func (r *mongoRepository) GetRideFareByID(ctx context.Context, id string) (*domain.RideFareModel, error) {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	result := r.db.Collection(db.RideFaresCollection).FindOne(ctx, bson.M{"_id": _id})
	if result.Err() != nil {
		return nil, result.Err()
	}

	var fare domain.RideFareModel
	err = result.Decode(&fare)
	if err != nil {
		return nil, err
	}

	return &fare, nil
}

func (r *mongoRepository) CreateReview(ctx context.Context, review *domain.ReviewModel) (*domain.ReviewModel, error) {
	if review.CreatedAt.IsZero() {
		review.CreatedAt = time.Now().UTC()
	}
	result, err := r.db.Collection(db.ReviewsCollection).InsertOne(ctx, review)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, domain.ErrReviewExists
		}
		return nil, err
	}
	review.ID = result.InsertedID.(primitive.ObjectID)
	return review, nil
}

func (r *mongoRepository) GetReviewByTripID(ctx context.Context, tripID string) (*domain.ReviewModel, error) {
	var review domain.ReviewModel
	err := r.db.Collection(db.ReviewsCollection).FindOne(ctx, bson.M{"tripID": tripID}).Decode(&review)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &review, nil
}

func (r *mongoRepository) ListReviewsByDriver(ctx context.Context, driverID string, limit int64) ([]*domain.ReviewModel, error) {
	if limit <= 0 {
		limit = 20
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit)
	cursor, err := r.db.Collection(db.ReviewsCollection).Find(ctx, bson.M{"driverID": driverID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reviews []*domain.ReviewModel
	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *mongoRepository) ListTripsByDriver(ctx context.Context, driverID string, statuses []string, limit int64) ([]*domain.TripModel, error) {
	if limit <= 0 {
		limit = 20
	}
	filter := bson.M{"driver.id": driverID}
	if len(statuses) > 0 {
		filter["status"] = bson.M{"$in": statuses}
	}
	opts := options.Find().SetSort(bson.D{{Key: "completedAt", Value: -1}}).SetLimit(limit)
	cursor, err := r.db.Collection(db.TripsCollection).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var trips []*domain.TripModel
	if err := cursor.All(ctx, &trips); err != nil {
		return nil, err
	}
	return trips, nil
}

// EnsureReviewIndexes creates a unique index so each trip can only be reviewed once.
func EnsureReviewIndexes(ctx context.Context, database *mongo.Database) error {
	_, err := database.Collection(db.ReviewsCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "tripID", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}
