package redispkg

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const LockTTL = 5 * time.Minute

// LockSeat attempts a distributed lock using SET NX EX (atomic).
// Returns true if lock was acquired, false if already locked by someone else.
func LockSeat(ctx context.Context, rdb *redis.Client, showtimeID, seatID, userID string) (bool, error) {
	key := lockKey(showtimeID, seatID)
	return rdb.SetNX(ctx, key, userID, LockTTL).Result()
}

// UnlockSeat releases the lock ONLY if the caller owns it (Lua script = atomic compare-and-delete).
// This prevents a user from accidentally releasing another user's lock.
func UnlockSeat(ctx context.Context, rdb *redis.Client, showtimeID, seatID, userID string) error {
	key := lockKey(showtimeID, seatID)
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)
	return script.Run(ctx, rdb, []string{key}, userID).Err()
}

// GetLockOwner returns the user_id holding the lock, or "" if no lock exists.
func GetLockOwner(ctx context.Context, rdb *redis.Client, showtimeID, seatID string) (string, error) {
	val, err := rdb.Get(ctx, lockKey(showtimeID, seatID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// GetLockTTL returns remaining TTL of the lock (useful to show countdown to user).
func GetLockTTL(ctx context.Context, rdb *redis.Client, showtimeID, seatID string) (time.Duration, error) {
	return rdb.TTL(ctx, lockKey(showtimeID, seatID)).Result()
}

func lockKey(showtimeID, seatID string) string {
	return fmt.Sprintf("lock:seat:%s:%s", showtimeID, seatID)
}

// userLockKey returns the Redis key that maps a user to their currently locked seat.
func userLockKey(showtimeID, userID string) string {
	return fmt.Sprintf("user:lock:%s:%s", showtimeID, userID)
}

// SetUserLock adds a seat to the user's lock set and refreshes the set TTL.
// Using a Redis Set allows multiple concurrent seat locks per user.
func SetUserLock(ctx context.Context, rdb *redis.Client, showtimeID, userID, seatID string) error {
	key := userLockKey(showtimeID, userID)
	pipe := rdb.Pipeline()
	pipe.SAdd(ctx, key, seatID)
	pipe.Expire(ctx, key, LockTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// GetUserLocks returns all seat IDs currently locked by the user in this showtime.
func GetUserLocks(ctx context.Context, rdb *redis.Client, showtimeID, userID string) ([]string, error) {
	val, err := rdb.SMembers(ctx, userLockKey(showtimeID, userID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

// DelUserLock removes a specific seat from the user's lock set (called on confirm / release).
func DelUserLock(ctx context.Context, rdb *redis.Client, showtimeID, userID, seatID string) {
	rdb.SRem(ctx, userLockKey(showtimeID, userID), seatID)
}

// ClearUserLocks deletes the entire user lock set (e.g. logout or full cancellation).
func ClearUserLocks(ctx context.Context, rdb *redis.Client, showtimeID, userID string) {
	rdb.Del(ctx, userLockKey(showtimeID, userID))
}