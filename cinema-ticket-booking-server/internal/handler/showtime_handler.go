package handler

import (
	"log"
	"net/http"

	"cinema-ticket-booking/internal/model"
	"github.com/gin-gonic/gin"
)

// ShowtimeResponse is returned by GET /api/showtimes.
type ShowtimeResponse = model.Showtime

// GetShowtimes returns all showtimes joined with their movie title.
// @Summary      List all showtimes
// @Tags         showtimes
// @Produce      json
// @Success      200  {array}   ShowtimeResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/showtimes [get]
func (h *Handler) GetShowtimes(c *gin.Context) {
	ctx := c.Request.Context()

	showtimes, err := h.Svcs.Booking.GetShowtimes(ctx)
	if err != nil {
		log.Printf("[GetShowtimes] service error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "failed to fetch showtimes"})
		return
	}

	c.JSON(http.StatusOK, showtimes)
}
