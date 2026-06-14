package handler

import (
	"log"
	"net/http"
	"time"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

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
}

// New creates a Handler with injected dependencies.
func New(db *mongo.Database, rdb *redis.Client, mq *amqp.Connection, authCl *firebaseAuth.Client, pub *rabbitmq.Publisher) *Handler {
	return &Handler{DB: db, RDB: rdb, MQ: mq, AuthCl: authCl, Publisher: pub}
}

// --- Models ---

// Booking is the document stored in MongoDB.
type Booking struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     string        `bson:"user_id"       json:"user_id"`
	ShowtimeID string        `bson:"showtime_id"   json:"showtime_id"`
	SeatID     string        `bson:"seat_id"       json:"seat_id"`
	Status     string        `bson:"status"        json:"status"`
	CreatedAt  time.Time     `bson:"created_at"    json:"created_at"`
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

// GetSeats returns all seats for a showtime
// @Summary      List seats for a showtime
// @Tags         seats
// @Produce      json
// @Param        showtime_id  path  string  true  "Showtime ID"
// @Success      200  {array}   SeatResponse
// @Failure      400  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/seats [get]
func (h *Handler) GetSeats(c *gin.Context) {
	showtimeID := c.Param("showtime_id")
	// TODO: query h.DB seats collection
	c.JSON(http.StatusOK, []SeatResponse{
		{ID: "seat-1", Label: "A1", Status: "AVAILABLE", ShowtimeID: showtimeID},
		{ID: "seat-2", Label: "A2", Status: "AVAILABLE", ShowtimeID: showtimeID},
	})
}

// LockSeat locks a seat for 5 minutes using a Redis distributed lock.
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
		log.Println("[LockSeat] uid missing from context — auth middleware may not have run")
		c.JSON(http.StatusUnauthorized, MessageResponse{Message: "unauthorized"})
		return
	}

	showtimeID := c.Param("showtime_id")
	seatID := c.Param("seat_id")

	log.Printf("[LockSeat] uid=%s showtime=%s seat=%s", uid, showtimeID, seatID)

	ok, err := redispkg.LockSeat(c.Request.Context(), h.RDB, showtimeID, seatID, uid)
	if err != nil {
		log.Printf("[LockSeat] redis error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "lock failed: " + err.Error()})
		return
	}
	if !ok {
		log.Printf("[LockSeat] seat already locked — uid=%s", uid)
		c.JSON(http.StatusConflict, MessageResponse{Message: "seat already locked by another user"})
		return
	}

	log.Printf("[LockSeat] lock acquired — uid=%s", uid)
	c.JSON(http.StatusOK, MessageResponse{Message: "seat locked for 5 minutes"})
}

// ConfirmBooking verifies the seat lock, saves the booking to MongoDB,
// publishes a booking.confirmed event, then releases the Redis lock.
// @Summary      Confirm booking after payment
// @Tags         bookings
// @Security     BearerAuth
// @Produce      json
// @Param        showtime_id  path  string  true  "Showtime ID"
// @Param        seat_id      path  string  true  "Seat ID"
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  MessageResponse
// @Failure      403  {object}  MessageResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/seats/{seat_id}/book [post]
func (h *Handler) ConfirmBooking(c *gin.Context) {
	uid := c.GetString("uid")
	if uid == "" {
		log.Println("[ConfirmBooking] uid missing from context — auth middleware may not have run")
		c.JSON(http.StatusUnauthorized, MessageResponse{Message: "unauthorized"})
		return
	}

	showtimeID := c.Param("showtime_id")
	seatID := c.Param("seat_id")

	log.Printf("[ConfirmBooking] uid=%s showtime=%s seat=%s", uid, showtimeID, seatID)

	// Verify this user owns the Redis lock
	owner, err := redispkg.GetLockOwner(c.Request.Context(), h.RDB, showtimeID, seatID)
	if err != nil {
		log.Printf("[ConfirmBooking] redis error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "could not verify lock"})
		return
	}
	log.Printf("[ConfirmBooking] lock owner=%q uid=%q match=%v", owner, uid, owner == uid)
	if owner != uid {
		c.JSON(http.StatusForbidden, MessageResponse{Message: "seat not locked by you or lock expired"})
		return
	}

	// Save booking to MongoDB
	booking := Booking{
		UserID:     uid,
		ShowtimeID: showtimeID,
		SeatID:     seatID,
		Status:     "BOOKED",
		CreatedAt:  time.Now().UTC(),
	}
	result, err := h.DB.Collection("bookings").InsertOne(c.Request.Context(), booking)
	if err != nil {
		log.Printf("[ConfirmBooking] mongodb insert error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "could not save booking"})
		return
	}
	log.Printf("[ConfirmBooking] booking inserted id=%v user_id=%s", result.InsertedID, uid)

	// Publish booking.confirmed event to RabbitMQ
	if h.Publisher != nil {
		_ = h.Publisher.Publish(c.Request.Context(), rabbitmq.EventBookingConfirmed, booking)
	}

	// Release the Redis lock now that the booking is persisted
	_ = redispkg.UnlockSeat(c.Request.Context(), h.RDB, showtimeID, seatID, uid)

	c.JSON(http.StatusOK, MessageResponse{Message: "booking confirmed"})
}