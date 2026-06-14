package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"cinema-ticket-booking/internal/model"
)

// UserRepository handles MongoDB reads and writes for the users collection.
type UserRepository struct {
	db *mongo.Database
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{db: db}
}

// UpsertUser creates or updates the user document on every login.
// role defaults to "user" for new accounts and is never overwritten.
func (r *UserRepository) UpsertUser(ctx context.Context, uid, email, displayName, photoURL string) (model.User, error) {
	now := time.Now().UTC()

	filter := bson.D{{Key: "_id", Value: uid}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "email", Value: email},
			{Key: "display_name", Value: displayName},
			{Key: "photo_url", Value: photoURL},
			{Key: "last_login_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "created_at", Value: now},
			{Key: "role", Value: model.RoleUser},
		}},
	}

	_, err := r.db.Collection("users").UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return model.User{}, err
	}

	return r.FindByUID(ctx, uid)
}

// FindByUID returns the stored user document, or an error if not found.
func (r *UserRepository) FindByUID(ctx context.Context, uid string) (model.User, error) {
	var user model.User
	err := r.db.Collection("users").
		FindOne(ctx, bson.D{{Key: "_id", Value: uid}}).
		Decode(&user)
	return user, err
}
