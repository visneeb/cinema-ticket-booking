package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SeatResponse struct {
    ID          string `json:"id"`
    Label       string `json:"label"`
    Status      string `json:"status"`
    ShowtimeID  string `json:"showtime_id"`
}

type BookingRequest struct {
    SeatID string `json:"seat_id" binding:"required"`
}

type MessageResponse struct {
    Message string `json:"message"`
}

// GetSeats returns all seats for a showtime
// @Summary      List seats for a showtime
// @Tags         seats
// @Produce      json
// @Param        showtime_id  path      string  true  "Showtime ID"
// @Success      200  {array}   SeatResponse
// @Failure      400  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/seats [get]
func GetSeats(c *gin.Context) {
    showtimeID := c.Param("showtime_id")
    // TODO: query MongoDB
    c.JSON(http.StatusOK, []SeatResponse{
        {ID: "seat-1", Label: "A1", Status: "AVAILABLE", ShowtimeID: showtimeID},
        {ID: "seat-2", Label: "A2", Status: "AVAILABLE", ShowtimeID: showtimeID},
    })
}

// LockSeat locks a seat for 5 minutes
// @Summary      Lock a seat for payment
// @Tags         seats
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        showtime_id  path      string          true  "Showtime ID"
// @Param        seat_id      path      string          true  "Seat ID"
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  MessageResponse
// @Failure      409  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/seats/{seat_id}/lock [post]
func LockSeat(c *gin.Context) {
    // TODO: call Redis LockSeat
    c.JSON(http.StatusOK, MessageResponse{Message: "seat locked for 5 minutes"})
}

// ConfirmBooking marks seat as BOOKED after payment
// @Summary      Confirm booking after payment
// @Tags         bookings
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        showtime_id  path      string          true  "Showtime ID"
// @Param        seat_id      path      string          true  "Seat ID"
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  MessageResponse
// @Failure      404  {object}  MessageResponse
// @Router       /api/showtimes/{showtime_id}/seats/{seat_id}/book [post]
func ConfirmBooking(c *gin.Context) {
    // TODO: update MongoDB + publish RabbitMQ event
    c.JSON(http.StatusOK, MessageResponse{Message: "booking confirmed"})
}