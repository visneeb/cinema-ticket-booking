package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Booking is stored in the cinema.bookings collection.
type Booking struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     string        `bson:"user_id"       json:"user_id"`
	ShowtimeID string        `bson:"showtime_id"   json:"showtime_id"`
	SeatID     string        `bson:"seat_id"       json:"seat_id"`
	Status     string        `bson:"status"        json:"status"`
	CreatedAt  time.Time     `bson:"created_at"    json:"created_at"`
}

// AuditLog is stored in the cinema.audit_logs collection.
type AuditLog struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	EventType  string        `bson:"event_type"    json:"event_type"`
	UserID     string        `bson:"user_id"       json:"user_id"`
	ShowtimeID string        `bson:"showtime_id"   json:"showtime_id"`
	SeatID     string        `bson:"seat_id"       json:"seat_id"`
	Message    string        `bson:"message"       json:"message"`
	CreatedAt  time.Time     `bson:"created_at"    json:"created_at"`
}

// Audit event type constants.
const (
	AuditSeatLocked     = "SEAT_LOCKED"
	AuditBookingSuccess = "BOOKING_SUCCESS"
	AuditBookingTimeout = "BOOKING_TIMEOUT"
	AuditSystemError    = "SYSTEM_ERROR"
)

// Seat status values used in both Redis (statusKey) and MongoDB (seats.status).
const (
	SeatAvailable = "AVAILABLE"
	SeatLocked    = "LOCKED"
	SeatBooked    = "BOOKED"
)

// Sentinel errors — handlers map these to HTTP status codes.
var (
	ErrSeatBooked   = ErrConst("seat already booked")
	ErrSeatLocked   = ErrConst("seat already locked by another user")
	ErrLockNotOwned = ErrConst("seat not locked by you or lock expired")
	ErrInternal     = ErrConst("internal error")
)

// ErrConst is a tiny helper so the sentinel errors above can be declared
// as simple string-backed errors without importing "errors" here.
type ErrConst string

func (e ErrConst) Error() string { return string(e) }