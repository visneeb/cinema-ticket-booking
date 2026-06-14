package model

import "time"

// Showtime is the API shape returned by GET /api/showtimes.
type Showtime struct {
	ID          string    `json:"id"`
	MovieID     string    `json:"movie_id"`
	StartsAt    time.Time `json:"starts_at"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}
