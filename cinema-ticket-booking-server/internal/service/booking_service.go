package service

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"cinema-ticket-booking/internal/model"
	"cinema-ticket-booking/internal/repository"
	wshub "cinema-ticket-booking/internal/websocket"
	"cinema-ticket-booking/pkg/rabbitmq"
)

// Re-export sentinel errors so existing callers (handlers) that reference
// service.ErrSeatBooked etc. keep working without changes.
var (
	ErrSeatBooked   = model.ErrSeatBooked
	ErrSeatLocked   = model.ErrSeatLocked
	ErrLockNotOwned = model.ErrLockNotOwned
	ErrInternal     = model.ErrInternal
)

type Booking = model.Booking

type Showtime = model.Showtime

type Seat = model.Seat

type SeatLock = model.SeatLock

// BookingService coordinates Redis locks, MongoDB, RabbitMQ, and WebSocket.
// It contains business rules only — all data access goes through the
// SeatLockRepository (Redis) and BookingRepository (MongoDB).
type BookingService struct {
	locks     *repository.SeatLockRepository
	bookings  *repository.BookingRepository
	Publisher *rabbitmq.Publisher
	Hub       *wshub.Hub
}

func NewBookingService(
	locks *repository.SeatLockRepository,
	bookings *repository.BookingRepository,
	pub *rabbitmq.Publisher,
	hub *wshub.Hub,
) *BookingService {
	return &BookingService{locks: locks, bookings: bookings, Publisher: pub, Hub: hub}
}

// LockSeat acquires a 5-minute Redis lock and broadcasts LOCKED to WebSocket clients.
// Returns the number of seconds remaining on the lock (300 for a new lock, or the
// current TTL when the same user re-locks after a page refresh).
//
// Lazy timeout detection: if the status key says LOCKED but the lock key is gone
// (expired), we publish a booking.timeout event before allowing the new lock.
func (s *BookingService) LockSeat(ctx context.Context, showtimeID, seatID, userID string) (int64, error) {
	statusVal := s.locks.GetStatus(ctx, showtimeID, seatID)

	switch statusVal {
	case model.SeatBooked:
		return 0, ErrSeatBooked

	case model.SeatLocked:
		owner, _ := s.locks.LockOwner(ctx, showtimeID, seatID)

		if owner == userID {
			// Same user re-locking after a page refresh — return remaining TTL.
			ttl, _ := s.locks.LockTTL(ctx, showtimeID, seatID)
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
		s.locks.DeleteStatus(ctx, showtimeID, seatID)
	}

	ok, err := s.locks.AcquireLock(ctx, showtimeID, seatID, userID)
	if err != nil {
		log.Printf("[BookingService.LockSeat] redis error: %v", err)
		s.bookings.InsertAuditLog(ctx, model.AuditSystemError, userID, showtimeID, seatID, "redis error: "+err.Error())
		return 0, ErrInternal
	}
	if !ok {
		return 0, ErrSeatLocked
	}

	s.locks.SetStatus(ctx, showtimeID, seatID, model.SeatLocked, repository.LockTTL)
	_ = s.locks.SetUserLock(ctx, showtimeID, userID, seatID) // reverse index for /my-lock

	// Write-through: keep MongoDB in sync so GetSeats is accurate even if Redis restarts.
	s.bookings.SetSeatStatus(ctx, seatID, model.SeatLocked)

	s.Hub.BroadcastSeatEvent(wshub.SeatEvent{SeatID: seatID, ShowtimeID: showtimeID, Status: model.SeatLocked})
	s.bookings.InsertAuditLog(ctx, model.AuditSeatLocked, userID, showtimeID, seatID, "seat locked")
	log.Printf("[BookingService.LockSeat] lock acquired uid=%s seat=%s", userID, seatID)
	return repository.LockTTLSeconds, nil
}

// ConfirmBooking verifies the lock, inserts a booking document, and broadcasts BOOKED.
func (s *BookingService) ConfirmBooking(ctx context.Context, showtimeID, seatID, userID string) (*Booking, error) {
	owner, err := s.locks.LockOwner(ctx, showtimeID, seatID)
	if err != nil {
		log.Printf("[BookingService.ConfirmBooking] redis error: %v", err)
		s.bookings.InsertAuditLog(ctx, model.AuditSystemError, userID, showtimeID, seatID, "redis error: "+err.Error())
		return nil, ErrInternal
	}
	if owner != userID {
		return nil, ErrLockNotOwned
	}

	booking := &Booking{
		UserID:     userID,
		ShowtimeID: showtimeID,
		SeatID:     seatID,
		Status:     model.SeatBooked,
		CreatedAt:  time.Now().UTC(),
	}
	oid, err := s.bookings.InsertBooking(ctx, booking)
	if err != nil {
		log.Printf("[BookingService.ConfirmBooking] mongo error: %v", err)
		s.bookings.InsertAuditLog(ctx, model.AuditSystemError, userID, showtimeID, seatID, "mongo insert error: "+err.Error())
		return nil, ErrInternal
	}
	booking.ID = oid

	// Mark seat permanently booked in Redis (no TTL), release the lock.
	s.locks.SetStatus(ctx, showtimeID, seatID, model.SeatBooked, 0)
	_ = s.locks.ReleaseLock(ctx, showtimeID, seatID, userID)
	s.locks.DelUserLock(ctx, showtimeID, userID, seatID) // remove seat from reverse index

	// Persist BOOKED status to MongoDB seats so GetSeats stays correct after a Redis restart.
	s.bookings.SetSeatStatus(ctx, seatID, model.SeatBooked)

	s.Hub.BroadcastSeatEvent(wshub.SeatEvent{SeatID: seatID, ShowtimeID: showtimeID, Status: model.SeatBooked})

	if s.Publisher != nil {
		_ = s.Publisher.Publish(ctx, rabbitmq.EventBookingConfirmed, booking)
	}

	s.bookings.InsertAuditLog(ctx, model.AuditBookingSuccess, userID, showtimeID, seatID, "booking confirmed")
	log.Printf("[BookingService.ConfirmBooking] booked id=%v uid=%s", booking.ID, userID)
	return booking, nil
}

// ReleaseLock lets the lock owner cancel their reservation early.
// It removes all Redis keys and resets MongoDB status to AVAILABLE,
// then broadcasts AVAILABLE so every connected client updates immediately.
func (s *BookingService) ReleaseLock(ctx context.Context, showtimeID, seatID, userID string) error {
	owner, err := s.locks.LockOwner(ctx, showtimeID, seatID)
	if err != nil {
		return ErrInternal
	}
	if owner != userID {
		return ErrLockNotOwned // seat is locked by someone else or already expired
	}

	// Delete distributed lock key + status key + reverse index
	_ = s.locks.ReleaseLock(ctx, showtimeID, seatID, userID)
	s.locks.DeleteStatus(ctx, showtimeID, seatID)
	s.locks.DelUserLock(ctx, showtimeID, userID, seatID)

	// Reset MongoDB so GetSeats never returns a stale LOCKED.
	s.bookings.SetSeatStatus(ctx, seatID, model.SeatAvailable)

	s.Hub.BroadcastSeatEvent(wshub.SeatEvent{SeatID: seatID, ShowtimeID: showtimeID, Status: model.SeatAvailable})
	s.bookings.InsertAuditLog(ctx, model.AuditBookingTimeout, userID, showtimeID, seatID, "user cancelled reservation")
	log.Printf("[BookingService.ReleaseLock] released uid=%s seat=%s", userID, seatID)
	return nil
}

func (s *BookingService) GetShowtimes(ctx context.Context) ([]Showtime, error) {
	return s.bookings.ListShowtimes(ctx)
}

func (s *BookingService) GetSeats(ctx context.Context, showtimeID bson.ObjectID) ([]Seat, error) {
	docs, err := s.bookings.FindSeatsByShowtime(ctx, showtimeID)
	if err != nil {
		return nil, err
	}

	seatIDs := make([]string, len(docs))
	for i, doc := range docs {
		seatIDs[i] = doc.ID.Hex()
	}

	redisStatuses, err := s.locks.GetStatuses(ctx, showtimeID.Hex(), seatIDs)
	if err != nil {
		log.Printf("[BookingService.GetSeats] redis status error: %v", err)
		return nil, err
	}

	result := make([]Seat, len(docs))
	for i, doc := range docs {
		status := doc.Status
		if status == "" {
			status = model.SeatAvailable
		}

		if i < len(redisStatuses) && redisStatuses[i] != "" {
			redisStatus := redisStatuses[i]
			if redisStatus == model.SeatBooked && doc.Status != model.SeatBooked {
				go s.locks.DeleteStatus(ctx, showtimeID.Hex(), doc.ID.Hex())
			} else if redisStatus == model.SeatLocked {
				owner, ownerErr := s.locks.LockOwner(ctx, showtimeID.Hex(), doc.ID.Hex())
				if ownerErr == nil && owner == "" {
					status = model.SeatAvailable
					go s.locks.DeleteStatus(ctx, showtimeID.Hex(), doc.ID.Hex())
					if doc.Status == model.SeatLocked {
						go s.bookings.SetSeatStatus(context.Background(), doc.ID.Hex(), model.SeatAvailable)
					}
				} else {
					status = redisStatus
				}
			} else {
				status = redisStatus
			}
		} else if status == model.SeatLocked {
			status = model.SeatAvailable
		}

		result[i] = Seat{
			ID:         doc.ID.Hex(),
			Label:      doc.Label,
			Status:     status,
			ShowtimeID: doc.ShowtimeID.Hex(),
		}
	}

	return result, nil
}

func (s *BookingService) GetMyLocks(ctx context.Context, showtimeID, userID string) ([]SeatLock, error) {
	seatIDs, err := s.locks.GetUserLocks(ctx, showtimeID, userID)
	if err != nil {
		log.Printf("[BookingService.GetMyLocks] redis error: %v", err)
		return nil, ErrInternal
	}

	locks := make([]SeatLock, 0, len(seatIDs))
	for _, seatID := range seatIDs {
		owner, err := s.locks.LockOwner(ctx, showtimeID, seatID)
		if err != nil {
			log.Printf("[BookingService.GetMyLocks] LockOwner error: %v", err)
			continue
		}
		if owner != userID {
			s.locks.DelUserLock(ctx, showtimeID, userID, seatID)
			continue
		}

		ttl, err := s.locks.LockTTL(ctx, showtimeID, seatID)
		if err != nil {
			log.Printf("[BookingService.GetMyLocks] GetLockTTL error: %v", err)
			continue
		}

		secs := int64(ttl.Seconds())
		if secs < 0 {
			secs = 0
		}
		locks = append(locks, SeatLock{SeatID: seatID, SecondsLeft: secs})
	}

	return locks, nil
}

