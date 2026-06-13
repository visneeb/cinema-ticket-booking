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