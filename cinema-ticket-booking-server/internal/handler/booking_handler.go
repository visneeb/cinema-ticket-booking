package handler

import (
	"errors"
	"log"
	"net/http"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"cinema-ticket-booking/internal/model"
	"cinema-ticket-booking/internal/service"
	wshub "cinema-ticket-booking/internal/websocket"
)

// Services groups all service-layer dependencies injected into the Handler.
type Services struct {
	Booking *service.BookingService
	User    service.UserService
	Admin   service.AdminService
}

// Handler holds HTTP-layer dependencies: auth client, WebSocket hub, and services.
type Handler struct {
	AuthCl *firebaseAuth.Client
	Hub    *wshub.Hub
	Svcs   Services
}

// New creates a Handler.
func New(authCl *firebaseAuth.Client, hub *wshub.Hub, svcs Services) *Handler {
	return &Handler{AuthCl: authCl, Hub: hub, Svcs: svcs}
}

// --- Response/Request types ---

type SeatResponse = model.Seat

type MessageResponse struct {
	Message string `json:"message"`
}

type LockResponse struct {
	Message     string `json:"message"`
	SecondsLeft int64  `json:"seconds_left"`
}

type SeatLock = model.SeatLock

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

	showtimeOID, err := bson.ObjectIDFromHex(showtimeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, MessageResponse{Message: "invalid showtime_id"})
		return
	}

	seats, err := h.Svcs.Booking.GetSeats(ctx, showtimeOID)
	if err != nil {
		log.Printf("[GetSeats] service error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "failed to fetch seats"})
		return
	}

	c.JSON(http.StatusOK, seats)
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

	secsLeft, err := h.Svcs.Booking.LockSeat(c.Request.Context(), showtimeID, seatID, uid)
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
// @Success      200  {object}  model.Booking
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

	booking, err := h.Svcs.Booking.ConfirmBooking(c.Request.Context(), showtimeID, seatID, uid)
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

	if err := h.Svcs.Booking.ReleaseLock(c.Request.Context(), showtimeID, seatID, uid); err != nil {
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

	locks, err := h.Svcs.Booking.GetMyLocks(ctx, showtimeID, uid)
	if err != nil {
		log.Printf("[GetMyLock] service error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "failed to fetch locks"})
		return
	}

	c.JSON(http.StatusOK, MyLocksResponse{Locks: locks})
}
