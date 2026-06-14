package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"cinema-ticket-booking/internal/model"
)

// Repository handles MongoDB reads for the admin dashboard (movies and users).
type AdminRepository interface {
	ListMovies(ctx context.Context) ([]model.Movie, error)
	ListUsers(ctx context.Context) ([]model.User, error)
}

type repository struct {
	db *mongo.Database
}

func NewRepository(db *mongo.Database) AdminRepository {
	return &repository{db: db}
}

func (r *repository) ListMovies(ctx context.Context) ([]model.Movie, error) {
	cur, err := r.db.Collection("movies").Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var movies []model.Movie
	err = cur.All(ctx, &movies)
	return movies, err
}

func (r *repository) ListUsers(ctx context.Context) ([]model.User, error) {
	cur, err := r.db.Collection("users").Find(ctx, bson.D{{Key: "role", Value: model.RoleUser}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var users []model.User
	err = cur.All(ctx, &users)
	return users, err
}