package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	wshub "cinema-ticket-booking/internal/websocket"
	"cinema-ticket-booking/pkg/rabbitmq"
	redispkg "cinema-ticket-booking/pkg/redis"
)

// Sentinel errors — handlers map these to HTTP status codes.
var (
	ErrSeatBooked   = errors.New("seat already booked")
	ErrSeatLocked   = errors.New("seat already locked by another user")
	ErrLockNotOwned = errors.New("seat not locked by you or lock expired")
	ErrInternal     = errors.New("internal error")
)

// --- MongoDB models ---

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

const (
	AuditSeatLocked     = "SEAT_LOCKED"
	AuditBookingSuccess = "BOOKING_SUCCESS"
	AuditBookingTimeout = "BOOKING_TIMEOUT"
	AuditSystemError    = "SYSTEM_ERROR"
)

// BookingService coordinates Redis locks, MongoDB, RabbitMQ, and WebSocket.
type BookingService struct {
	DB        *mongo.Database
	RDB       *redis.Client
	Publisher *rabbitmq.Publisher
	Hub       *wshub.Hub
}

func NewBookingService(db *mongo.Database, rdb *redis.Client, pub *rabbitmq.Publisher, hub *wshub.Hub) *BookingService {
	return &BookingService{DB: db, RDB: rdb, Publisher: pub, Hub: hub}
}

// writeAuditLog persists an audit event synchronously. Errors are logged, not propagated.
func (s *BookingService) writeAuditLog(ctx context.Context, eventType, userID, showtimeID, seatID, msg string) {
	doc := AuditLog{
		EventType:  eventType,
		UserID:     userID,
		ShowtimeID: showtimeID,
		SeatID:     seatID,
		Message:    msg,
		CreatedAt:  time.Now().UTC(),
	}
	if _, err := s.DB.Collection("audit_logs").InsertOne(ctx, doc); err != nil {
		log.Printf("[AuditLog] write error: %v", err)
	}
}

// statusKey tracks seat status (LOCKED with TTL | BOOKED permanent) in Redis.
func statusKey(showtimeID, seatID string) string {
	return "seat:status:" + showtimeID + ":" + seatID
}

// LockSeat acquires a 5-minute Redis lock and broadcasts LOCKED to WebSocket clients.
// Returns the number of seconds remaining on the lock (300 for a new lock, or the
// current TTL when the same user re-locks after a page refresh).
//
// Lazy timeout detection: if the status key says LOCKED but the lock key is gone
// (expired), we publish a booking.timeout event before allowing the new lock.
func (s *BookingService) LockSeat(ctx context.Context, showtimeID, seatID, userID string) (int64, error) {
	statusVal := s.RDB.Get(ctx, statusKey(showtimeID, seatID)).Val()

	switch statusVal {
	case "BOOKED":
		return 0, ErrSeatBooked

	case "LOCKED":
		owner, _ := redispkg.GetLockOwner(ctx, s.RDB, showtimeID, seatID)

		if owner == userID {
			// Same user re-locking after a page refresh — return remaining TTL.
			ttl, _ := redispkg.GetLockTTL(ctx, s.RDB, showtimeID, seatID)
			secs := int64(ttl.Seconds())
			if secs < 0 {
				secs = 0
			}
			log.Printf("[BookingService.LockSeat] re-lock uid=%s seat=%s ttl=%ds", userID, seatID, secs)
			return secs, nil
		}

		if owner != "" {
			return 0, ErrSeatLocked // actively locked by a different user
		}

		// Lock key expired — the previous user timed out.
		if s.Publisher != nil {
			_ = s.Publisher.Publish(ctx, rabbitmq.EventBookingTimeout, map[string]string{
				"showtime_id": showtimeID, "seat_id": seatID, "user_id": "",
			})
		}
		s.RDB.Del(ctx, statusKey(showtimeID, seatID))
	}

	ok, err := redispkg.LockSeat(ctx, s.RDB, showtimeID, seatID, userID)
	if err != nil {
		log.Printf("[BookingService.LockSeat] redis error: %v", err)
		s.writeAuditLog(ctx, AuditSystemError, userID, showtimeID, seatID, "redis error: "+err.Error())
		return 0, ErrInternal
	}
	if !ok {
		return 0, ErrSeatLocked
	}

	s.RDB.Set(ctx, statusKey(showtimeID, seatID), "LOCKED", redispkg.LockTTL)
	_ = redispkg.SetUserLock(ctx, s.RDB, showtimeID, userID, seatID) // reverse index for /my-lock

	// Write-through: keep MongoDB in sync so GetSeats is accurate even if Redis restarts.
	if seatOID, oErr := bson.ObjectIDFromHex(seatID); oErr == nil {
		_, _ = s.DB.Collection("seats").UpdateOne(ctx,
			bson.D{{Key: "_id", Value: seatOID}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: "LOCKED"}}}},
		)
	}

	s.Hub.BroadcastSeatEvent(wshub.SeatEvent{SeatID: seatID, ShowtimeID: showtimeID, Status: "LOCKED"})
	s.writeAuditLog(ctx, AuditSeatLocked, userID, showtimeID, seatID, "seat locked")
	log.Printf("[BookingService.LockSeat] lock acquired uid=%s seat=%s", userID, seatID)
	return int64(redispkg.LockTTL.Seconds()), nil
}

// ConfirmBooking verifies the lock, inserts a booking document, and broadcasts BOOKED.
func (s *BookingService) ConfirmBooking(ctx context.Context, showtimeID, seatID, userID string) (*Booking, error) {
	owner, err := redispkg.GetLockOwner(ctx, s.RDB, showtimeID, seatID)
	if err != nil {
		log.Printf("[BookingService.ConfirmBooking] redis error: %v", err)
		s.writeAuditLog(ctx, AuditSystemError, userID, showtimeID, seatID, "redis error: "+err.Error())
		return nil, ErrInternal
	}
	if owner != userID {
		return nil, ErrLockNotOwned
	}

	booking := &Booking{
		UserID:     userID,
		ShowtimeID: showtimeID,
		SeatID:     seatID,
		Status:     "BOOKED",
		CreatedAt:  time.Now().UTC(),
	}
	result, err := s.DB.Collection("bookings").InsertOne(ctx, booking)
	if err != nil {
		log.Printf("[BookingService.ConfirmBooking] mongo error: %v", err)
		s.writeAuditLog(ctx, AuditSystemError, userID, showtimeID, seatID, "mongo insert error: "+err.Error())
		return nil, ErrInternal
	}
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		booking.ID = oid
	}

	// Mark seat permanently booked in Redis (no TTL), release the lock.
	s.RDB.Set(ctx, statusKey(showtimeID, seatID), "BOOKED", 0)
	_ = redispkg.UnlockSeat(ctx, s.RDB, showtimeID, seatID, userID)
	redispkg.DelUserLock(ctx, s.RDB, showtimeID, userID, seatID) // remove seat from reverse index

	// Persist BOOKED status to MongoDB seats so GetSeats stays correct after a Redis restart.
	// seatID is an ObjectID hex string — parse it back before filtering.
	if seatOID, err := bson.ObjectIDFromHex(seatID); err == nil {
		_, _ = s.DB.Collection("seats").UpdateOne(ctx,
			bson.D{{Key: "_id", Value: seatOID}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: "BOOKED"}}}},
		)
	}

	s.Hub.BroadcastSeatEvent(wshub.SeatEvent{SeatID: seatID, ShowtimeID: showtimeID, Status: "BOOKED"})

	if s.Publisher != nil {
		_ = s.Publisher.Publish(ctx, rabbitmq.EventBookingConfirmed, booking)
	}

	s.writeAuditLog(ctx, AuditBookingSuccess, userID, showtimeID, seatID, "booking confirmed")
	log.Printf("[BookingService.ConfirmBooking] booked id=%v uid=%s", booking.ID, userID)
	return booking, nil
}

// ReleaseLock lets the lock owner cancel their reservation early.
// It removes all Redis keys and resets MongoDB status to AVAILABLE,
// then broadcasts AVAILABLE so every connected client updates immediately.
func (s *BookingService) ReleaseLock(ctx context.Context, showtimeID, seatID, userID string) error {
	owner, err := redispkg.GetLockOwner(ctx, s.RDB, showtimeID, seatID)
	if err != nil {
		return ErrInternal
	}
	if owner != userID {
		return ErrLockNotOwned // seat is locked by someone else or already expired
	}

	// Delete distributed lock key + status key + reverse index
	_ = redispkg.UnlockSeat(ctx, s.RDB, showtimeID, seatID, userID)
	s.RDB.Del(ctx, statusKey(showtimeID, seatID))
	redispkg.DelUserLock(ctx, s.RDB, showtimeID, userID, seatID)

	// Reset MongoDB so GetSeats never returns a stale LOCKED.
	if seatOID, oErr := bson.ObjectIDFromHex(seatID); oErr == nil {
		_, _ = s.DB.Collection("seats").UpdateOne(ctx,
			bson.D{{Key: "_id", Value: seatOID}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: "AVAILABLE"}}}},
		)
	}

	s.Hub.BroadcastSeatEvent(wshub.SeatEvent{SeatID: seatID, ShowtimeID: showtimeID, Status: "AVAILABLE"})
	s.writeAuditLog(ctx, AuditBookingTimeout, userID, showtimeID, seatID, "user cancelled reservation")
	log.Printf("[BookingService.ReleaseLock] released uid=%s seat=%s", userID, seatID)
	return nil
}

// StartAuditConsumer listens for booking.timeout events from RabbitMQ,
// writes BOOKING_TIMEOUT audit logs, and broadcasts AVAILABLE via WebSocket.
func (s *BookingService) StartAuditConsumer(conn *amqp.Connection) error {
	consumer, err := rabbitmq.NewConsumer(conn)
	if err != nil {
		return err
	}
	return consumer.Consume("audit_timeout", rabbitmq.EventBookingTimeout, func(body []byte) {
		var ev struct {
			UserID     string `json:"user_id"`
			ShowtimeID string `json:"showtime_id"`
			SeatID     string `json:"seat_id"`
		}
		if err := json.Unmarshal(body, &ev); err != nil {
			log.Printf("[AuditConsumer] unmarshal error: %v", err)
			return
		}
		ctx := context.Background()
		s.RDB.Del(ctx, statusKey(ev.ShowtimeID, ev.SeatID))

		// Write-through: reset MongoDB status so GetSeats never shows a stale LOCKED.
		if seatOID, oErr := bson.ObjectIDFromHex(ev.SeatID); oErr == nil {
			_, _ = s.DB.Collection("seats").UpdateOne(ctx,
				bson.D{{Key: "_id", Value: seatOID}},
				bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: "AVAILABLE"}}}},
			)
		}

		s.Hub.BroadcastSeatEvent(wshub.SeatEvent{SeatID: ev.SeatID, ShowtimeID: ev.ShowtimeID, Status: "AVAILABLE"})
		s.writeAuditLog(ctx, AuditBookingTimeout, ev.UserID, ev.ShowtimeID, ev.SeatID, "lock expired without booking")
		log.Printf("[AuditConsumer] timeout seat=%s showtime=%s", ev.SeatID, ev.ShowtimeID)
	})
}