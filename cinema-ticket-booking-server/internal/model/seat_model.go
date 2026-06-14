package model

// Seat is the API shape returned by GET /api/seats and used across services.
type Seat struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	ShowtimeID string `json:"showtime_id"`
}

// SeatLock represents an active lock held by a user with time remaining in seconds.
type SeatLock struct {
	SeatID      string `json:"seat_id"`
	SecondsLeft int64  `json:"seconds_left"`
}
