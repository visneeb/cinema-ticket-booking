package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	redispkg "cinema-ticket-booking/pkg/redis"
)

// SeatLockRepository wraps all Redis access related to seat locks and seat status.

type SeatLockRepository struct {
	rdb *redis.Client
}

func NewSeatLockRepository(rdb *redis.Client) *SeatLockRepository {
	return &SeatLockRepository{rdb: rdb}
}

// statusKey tracks seat status (LOCKED with TTL | BOOKED permanent) in Redis.
func statusKey(showtimeID, seatID string) string {
	return "seat:status:" + showtimeID + ":" + seatID
}

// GetStatus returns the current seat status string ("", "LOCKED", "BOOKED").
func (r *SeatLockRepository) GetStatus(ctx context.Context, showtimeID, seatID string) string {
	return r.rdb.Get(ctx, statusKey(showtimeID, seatID)).Val()
}

func (r *SeatLockRepository) GetStatuses(ctx context.Context, showtimeID string, seatIDs []string) ([]string, error) {
	if len(seatIDs) == 0 {
		return nil, nil
	}

	keys := make([]string, len(seatIDs))
	for i, seatID := range seatIDs {
		keys[i] = statusKey(showtimeID, seatID)
	}

	vals, err := r.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	statuses := make([]string, len(vals))
	for i, v := range vals {
		if s, ok := v.(string); ok {
			statuses[i] = s
		}
	}
	return statuses, nil
}

// SetStatus sets the seat status key. ttl=0 means no expiry (used for BOOKED).
func (r *SeatLockRepository) SetStatus(ctx context.Context, showtimeID, seatID, status string, ttl time.Duration) {
	r.rdb.Set(ctx, statusKey(showtimeID, seatID), status, ttl)
}

// DeleteStatus removes the seat status key entirely.
func (r *SeatLockRepository) DeleteStatus(ctx context.Context, showtimeID, seatID string) {
	r.rdb.Del(ctx, statusKey(showtimeID, seatID))
}

// AcquireLock attempts to acquire the distributed seat lock (SET NX EX).
func (r *SeatLockRepository) AcquireLock(ctx context.Context, showtimeID, seatID, userID string) (bool, error) {
	return redispkg.LockSeat(ctx, r.rdb, showtimeID, seatID, userID)
}

// ReleaseLock deletes the distributed lock key, only if owned by userID.
func (r *SeatLockRepository) ReleaseLock(ctx context.Context, showtimeID, seatID, userID string) error {
	return redispkg.UnlockSeat(ctx, r.rdb, showtimeID, seatID, userID)
}

// LockOwner returns the user_id currently holding the lock, or "" if none.
func (r *SeatLockRepository) LockOwner(ctx context.Context, showtimeID, seatID string) (string, error) {
	return redispkg.GetLockOwner(ctx, r.rdb, showtimeID, seatID)
}

// LockTTL returns remaining TTL on the distributed lock key.
func (r *SeatLockRepository) LockTTL(ctx context.Context, showtimeID, seatID string) (time.Duration, error) {
	return redispkg.GetLockTTL(ctx, r.rdb, showtimeID, seatID)
}

// SetUserLock writes the reverse index (user → seat) used by /my-lock.
func (r *SeatLockRepository) SetUserLock(ctx context.Context, showtimeID, userID, seatID string) error {
	return redispkg.SetUserLock(ctx, r.rdb, showtimeID, userID, seatID)
}

func (r *SeatLockRepository) GetUserLocks(ctx context.Context, showtimeID, userID string) ([]string, error) {
	return redispkg.GetUserLocks(ctx, r.rdb, showtimeID, userID)
}

// DelUserLock removes the reverse index entry for a user/seat pair.
func (r *SeatLockRepository) DelUserLock(ctx context.Context, showtimeID, userID, seatID string) {
	redispkg.DelUserLock(ctx, r.rdb, showtimeID, userID, seatID)
}

// LockTTLSeconds is the standard seat-lock duration, exposed so the service
const LockTTLSeconds = int64(redispkg.LockTTL / time.Second)

// LockTTL is the standard seat-lock duration as a time.Duration.
var LockTTL = redispkg.LockTTL
