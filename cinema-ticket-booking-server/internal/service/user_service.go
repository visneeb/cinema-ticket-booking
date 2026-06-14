package service

import (
	"context"
	"log"

	"cinema-ticket-booking/internal/model"
	"cinema-ticket-booking/internal/repository"
)

// UserService handles user profile operations.
type UserService interface {
	UpsertUser(ctx context.Context, uid, email, displayName, photoURL string) (model.User, error)
	FindByUID(ctx context.Context, uid string) (model.User, error)
}

type userService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) UpsertUser(ctx context.Context, uid, email, displayName, photoURL string) (model.User, error) {
	user, err := s.repo.UpsertUser(ctx, uid, email, displayName, photoURL)
	if err != nil {
		log.Printf("[UserService.UpsertUser] uid=%s: %v", uid, err)
		return model.User{}, err
	}
	log.Printf("[UserService.UpsertUser] upserted uid=%s email=%s role=%s", uid, email, user.Role)
	return user, nil
}

func (s *userService) FindByUID(ctx context.Context, uid string) (model.User, error) {
	return s.repo.FindByUID(ctx, uid)
}
