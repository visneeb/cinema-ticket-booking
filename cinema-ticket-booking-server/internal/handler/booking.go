package handler

import (
	"context"
	"errors"
	"log"
	"net/http"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"cinema-ticket-booking/internal/service"
	wshub "cinema-ticket-booking/internal/websocket"
	"cinema-ticket-booking/pkg/rabbitmq"
	redispkg "cinema-ticket-booking/pkg/redis"
)

// Handler holds all service dependencies.
type Handler struct {
	DB        *mongo.Database
	RDB       *redis.Client
	MQ        *amqp.Connection
	AuthCl    *firebaseAuth.Client
	Publisher *rabbitmq.Publisher
	Svc       *service.BookingService
	Hub       *wshub.Hub
}

// New creates a Handler with injected dependencies.
func New(
	db *mongo.Database,
	rdb *redis.Client,
	mq *amqp.Connection,
	authCl *firebaseAuth.Client,
	pub *rabbitmq.Publisher,
	svc *service.BookingService,
	hub *wshub.Hub,
) *Handler {
	return &Handler{DB: db, RDB: rdb, MQ: mq, AuthCl: authCl, Publisher: pub, Svc: svc, Hub: hub}
}

// --- Response/Request types ---

type SeatResponse struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	ShowtimeID string `json:"showtime_id"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type LockResponse struct {
	Message     string `json:"message"`
	SecondsLeft int64  `json:"seconds_left"`
}

// SeatLock describes a single active seat lock for the current user.
type SeatLock struct {
	SeatID      string `json:"seat_id"`
	SecondsLeft int64  `json:"seconds_left"`
}

// MyLocksResponse lists all active seat locks the current user holds.
type MyLocksResponse struct {
	Locks []SeatLock `json:"locks"`
}

// GetSeats queries the seats collection in MongoDB and overlays real-time
// LOCKED/BOOKED status from Redis so every client sees the current state.
// Both showtime_id and returned seat IDs are ObjectID hex strings.
// @Summary      List seats for a showtime
// @Tags         seats
// @Produce      json
// @Param        showtime_id  path  string  true  "Showtime ObjectID hex"
// @Success      200  {array}   SeatResponse
// @Failure      400  {object}  MessageResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/seats [get]
func (h *Handler) GetSeats(c *gin.Context) {
	showtimeIDStr := c.Param("showtime_id")
	ctx := c.Request.Context()

	// Parse showtime_id as ObjectID — seed stores it as ObjectID, not string
	showtimeOID, err := bson.ObjectIDFromHex(showtimeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, MessageResponse{Message: "invalid showtime_id"})
		return
	}

	// 1. Fetch seat documents from MongoDB, sorted by _id (insertion order = A1…D10).
	// Without a sort the {showtime_id,status} index makes MongoDB return BOOKED
	// seats last, causing them to visually "jump" to the end of the grid.
	findOpts := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}})
	cursor, err := h.DB.Collection("seats").Find(ctx, bson.D{{Key: "showtime_id", Value: showtimeOID}}, findOpts)
	if err != nil {
		log.Printf("[GetSeats] mongo find error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "failed to fetch seats"})
		return
	}
	defer cursor.Close(ctx)

	var docs []struct {
		ID         bson.ObjectID `bson:"_id"`
		Label      string        `bson:"label"`
		ShowtimeID bson.ObjectID `bson:"showtime_id"`
		Status     string        `bson:"status"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		log.Printf("[GetSeats] cursor decode error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "failed to read seats"})
		return
	}

	if len(docs) == 0 {
		c.JSON(http.StatusOK, []SeatResponse{})
		return
	}

	// 2. Batch-fetch real-time status from Redis in one round-trip.
	// Redis keys use hex strings — cheap to compute, URL-safe.
	keys := make([]string, len(docs))
	for i, doc := range docs {
		keys[i] = "seat:status:" + showtimeIDStr + ":" + doc.ID.Hex()
	}
	redisVals, _ := h.RDB.MGet(ctx, keys...).Result()

	// 3. Merge rules (Redis is the realtime layer, MongoDB is the permanent source of truth):
	//   • Redis = "LOCKED"  → use LOCKED (ephemeral, Redis is authoritative)
	//   • Redis = "BOOKED" AND MongoDB = "BOOKED" → use BOOKED
	//   • Redis = "BOOKED" AND MongoDB ≠ "BOOKED" → stale Redis cache; trust MongoDB + evict key
	//   • Redis empty AND MongoDB = "LOCKED" → lock expired; treat as AVAILABLE (zombie guard)
	//   • Redis empty → use MongoDB (AVAILABLE or BOOKED)
	result := make([]SeatResponse, len(docs))
	for i, doc := range docs {
		status := doc.Status
		if status == "" {
			status = "AVAILABLE"
		}
		if i < len(redisVals) && redisVals[i] != nil {
			if redisStatus, ok := redisVals[i].(string); ok && redisStatus != "" {
				if redisStatus == "BOOKED" && doc.Status != "BOOKED" {
					// Redis says BOOKED but MongoDB was reset — evict the stale key.
					go h.RDB.Del(ctx, keys[i])
					// status stays whatever MongoDB says (AVAILABLE)
				} else if redisStatus == "LOCKED" {
					owner, err := redispkg.GetLockOwner(ctx, h.RDB, showtimeIDStr, doc.ID.Hex())
					if err == nil && owner == "" {
						// Stale LOCKED key: the lock itself expired, so this seat is actually available.
						status = "AVAILABLE"
						go h.RDB.Del(ctx, keys[i])
						if doc.Status == "LOCKED" {
							seatOID, oErr := bson.ObjectIDFromHex(doc.ID.Hex())
							if oErr == nil {
								go h.DB.Collection("seats").UpdateOne(context.Background(),
									bson.D{{Key: "_id", Value: seatOID}},
									bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: "AVAILABLE"}}}},
								)
							}
						}
					} else {
						status = redisStatus
					}
				} else {
					status = redisStatus
				}
			}
		} else if status == "LOCKED" {
			// MongoDB says LOCKED but Redis key is gone — lock already expired.
			status = "AVAILABLE"
		}
		result[i] = SeatResponse{
			ID:         doc.ID.Hex(),
			Label:      doc.Label,
			Status:     status,
			ShowtimeID: doc.ShowtimeID.Hex(),
		}
	}
	c.JSON(http.StatusOK, result)
}

// LockSeat acquires a 5-minute Redis lock and broadcasts LOCKED via WebSocket.
// @Summary      Lock a seat for payment
// @Tags         seats
// @Security     BearerAuth
// @Produce      json
// @Param        showtime_id  path  string  true  "Showtime ID"
// @Param        seat_id      path  string  true  "Seat ID"
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  MessageResponse
// @Failure      409  {object}  MessageResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/seats/{seat_id}/lock [post]
func (h *Handler) LockSeat(c *gin.Context) {
	uid := c.GetString("uid")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, MessageResponse{Message: "unauthorized"})
		return
	}
	showtimeID := c.Param("showtime_id")
	seatID := c.Param("seat_id")
	log.Printf("[LockSeat] uid=%s showtime=%s seat=%s", uid, showtimeID, seatID)

	secsLeft, err := h.Svc.LockSeat(c.Request.Context(), showtimeID, seatID, uid)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSeatLocked), errors.Is(err, service.ErrSeatBooked):
			c.JSON(http.StatusConflict, MessageResponse{Message: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, MessageResponse{Message: "lock failed"})
		}
		return
	}
	c.JSON(http.StatusOK, LockResponse{Message: "seat locked", SecondsLeft: secsLeft})
}

// ConfirmBooking verifies the lock, persists the booking, and broadcasts BOOKED.
// @Summary      Confirm booking after payment
// @Tags         bookings
// @Security     BearerAuth
// @Produce      json
// @Param        showtime_id  path  string  true  "Showtime ID"
// @Param        seat_id      path  string  true  "Seat ID"
// @Success      200  {object}  service.Booking
// @Failure      401  {object}  MessageResponse
// @Failure      403  {object}  MessageResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/seats/{seat_id}/book [post]
func (h *Handler) ConfirmBooking(c *gin.Context) {
	uid := c.GetString("uid")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, MessageResponse{Message: "unauthorized"})
		return
	}
	showtimeID := c.Param("showtime_id")
	seatID := c.Param("seat_id")
	log.Printf("[ConfirmBooking] uid=%s showtime=%s seat=%s", uid, showtimeID, seatID)

	booking, err := h.Svc.ConfirmBooking(c.Request.Context(), showtimeID, seatID, uid)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLockNotOwned):
			c.JSON(http.StatusForbidden, MessageResponse{Message: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, MessageResponse{Message: "booking failed"})
		}
		return
	}
	c.JSON(http.StatusOK, booking)
}

// ReleaseLock lets the lock owner cancel their reservation early.
// @Summary      Release an active seat lock
// @Tags         seats
// @Security     BearerAuth
// @Produce      json
// @Param        showtime_id  path  string  true  "Showtime ObjectID hex"
// @Param        seat_id      path  string  true  "Seat ObjectID hex"
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  MessageResponse
// @Failure      403  {object}  MessageResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/seats/{seat_id}/lock [delete]
func (h *Handler) ReleaseLock(c *gin.Context) {
	uid := c.GetString("uid")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, MessageResponse{Message: "unauthorized"})
		return
	}
	showtimeID := c.Param("showtime_id")
	seatID := c.Param("seat_id")

	if err := h.Svc.ReleaseLock(c.Request.Context(), showtimeID, seatID, uid); err != nil {
		switch {
		case errors.Is(err, service.ErrLockNotOwned):
			c.JSON(http.StatusForbidden, MessageResponse{Message: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, MessageResponse{Message: "release failed"})
		}
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "seat released"})
}

// GetMyLock returns all seats currently locked by the authenticated user for this showtime.
// Stale entries (lock expired in Redis) are cleaned up lazily and excluded from the result.
// @Summary      Get caller's active seat locks
// @Tags         seats
// @Security     BearerAuth
// @Produce      json
// @Param        showtime_id  path  string  true  "Showtime ObjectID hex"
// @Success      200  {object}  MyLocksResponse
// @Failure      401  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/my-lock [get]
func (h *Handler) GetMyLock(c *gin.Context) {
	uid := c.GetString("uid")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, MessageResponse{Message: "unauthorized"})
		return
	}
	showtimeID := c.Param("showtime_id")
	ctx := c.Request.Context()

	seatIDs, err := redispkg.GetUserLocks(ctx, h.RDB, showtimeID, uid)
	if err != nil || len(seatIDs) == 0 {
		c.JSON(http.StatusOK, MyLocksResponse{Locks: []SeatLock{}})
		return
	}

	locks := make([]SeatLock, 0, len(seatIDs))
	for _, seatID := range seatIDs {
		owner, _ := redispkg.GetLockOwner(ctx, h.RDB, showtimeID, seatID)
		if owner != uid {
			// Stale entry — lock expired; remove it lazily.
			redispkg.DelUserLock(ctx, h.RDB, showtimeID, uid, seatID)
			continue
		}
		ttl, _ := redispkg.GetLockTTL(ctx, h.RDB, showtimeID, seatID)
		secs := int64(ttl.Seconds())
		if secs < 0 {
			secs = 0
		}
		locks = append(locks, SeatLock{SeatID: seatID, SecondsLeft: secs})
	}

	c.JSON(http.StatusOK, MyLocksResponse{Locks: locks})
}