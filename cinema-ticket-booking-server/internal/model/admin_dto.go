package model

import "time"

// AdminBookingItem is the enriched shape returned to the admin UI.
// Kept separate to avoid changing existing model.Booking serialization.
type AdminBookingItem struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	ShowtimeID string    `json:"showtime_id"`
	MovieID    string    `json:"movie_id"`
	SeatID     string    `json:"seat_id"`
	SeatLabel  string    `json:"seat_label"`
	SeatStatus string    `json:"seat_status"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type AdminBookingResponse struct {
	Items    []AdminBookingItem `json:"items"`
	Total    int64              `json:"total"`
	Page     int64              `json:"page"`
	PageSize int64              `json:"page_size"`
}
