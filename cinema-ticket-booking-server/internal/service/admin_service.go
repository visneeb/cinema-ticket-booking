package service

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"cinema-ticket-booking/internal/model"
	"cinema-ticket-booking/internal/repository"
)

// AdminService defines the admin-only operations.
type AdminService interface {
	ListBookings(ctx context.Context, req model.ListBookingRequest) (*model.AdminBookingResponse, error)
	ListMovies(ctx context.Context) ([]model.Movie, error)
	ListUsers(ctx context.Context) ([]model.User, error)
}

type adminService struct {
	bookingRepo *repository.BookingRepository
	repo        repository.AdminRepository
}

// NewAdminService constructs an AdminService.
func NewAdminService(bookingRepo *repository.BookingRepository, repo repository.AdminRepository) AdminService {
	return &adminService{bookingRepo: bookingRepo, repo: repo}
}

func (s *adminService) ListBookings(ctx context.Context, req model.ListBookingRequest) (*model.AdminBookingResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	filter := repository.BookingFilter{
		MovieID:  req.MovieID,
		UserID:   req.UserID,
		DateFrom: req.DateFrom,
		DateTo:   req.DateTo,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	bookings, total, err := s.bookingRepo.ListBookings(ctx, filter)
	if err != nil {
		return nil, err
	}

	// collect unique seat and showtime IDs
	seatSet := make(map[string]struct{})
	showtimeSet := make(map[string]struct{})
	for _, b := range bookings {
		if b.SeatID != "" {
			seatSet[b.SeatID] = struct{}{}
		}
		if b.ShowtimeID != "" {
			showtimeSet[b.ShowtimeID] = struct{}{}
		}
	}
	seatIDs := make([]string, 0, len(seatSet))
	showtimeIDs := make([]string, 0, len(showtimeSet))
	for id := range seatSet {
		seatIDs = append(seatIDs, id)
	}
	for id := range showtimeSet {
		showtimeIDs = append(showtimeIDs, id)
	}

	seatsMap, _ := s.bookingRepo.GetSeatsByIDs(ctx, seatIDs)
	showtimeMap, _ := s.bookingRepo.GetShowtimesMovieIDs(ctx, showtimeIDs)

	items := make([]model.AdminBookingItem, 0, len(bookings))
	for _, b := range bookings {
		id := ""
		if b.ID != (bson.ObjectID{}) {
			id = b.ID.Hex()
		}
		seatLabel := ""
		seatStatus := ""
		if sd, ok := seatsMap[b.SeatID]; ok {
			seatLabel = sd.Label
			seatStatus = sd.Status
		}
		movieID := showtimeMap[b.ShowtimeID]
		items = append(items, model.AdminBookingItem{
			ID:         id,
			UserID:     b.UserID,
			ShowtimeID: b.ShowtimeID,
			MovieID:    movieID,
			SeatID:     b.SeatID,
			SeatLabel:  seatLabel,
			SeatStatus: seatStatus,
			Status:     b.Status,
			CreatedAt:  b.CreatedAt,
		})
	}

	return &model.AdminBookingResponse{
		Items:    items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *adminService) ListMovies(ctx context.Context) ([]model.Movie, error) {
	return s.repo.ListMovies(ctx)
}

func (s *adminService) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.repo.ListUsers(ctx)
}