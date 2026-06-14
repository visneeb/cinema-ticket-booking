package handler

import (
	"net/http"
	"strconv"

	admindto "cinema-ticket-booking/internal/model"

	"github.com/gin-gonic/gin"
)

// ListBookings godoc
// @Summary      List all bookings (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        movie_id   query  string  false  "Movie ObjectID hex"
// @Param        user_id    query  string  false  "User UID"
// @Param        date_from  query  string  false  "Start date YYYY-MM-DD"
// @Param        date_to    query  string  false  "End date YYYY-MM-DD"
// @Param        status     query  string  false  "Booking status"
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        page_size  query  int     false  "Page size (default 20, max 100)"
// @Success      200  {object}  model.AdminBookingResponse
// @Failure      403  {object}  MessageResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/admin/bookings [get]
func (h *Handler) ListBookings(c *gin.Context) {
	page, _ := strconv.ParseInt(c.Query("page"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.Query("page_size"), 10, 64)

	req := admindto.ListBookingRequest{
		MovieID:  c.Query("movie_id"),
		UserID:   c.Query("user_id"),
		DateFrom: c.Query("date_from"),
		DateTo:   c.Query("date_to"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.Svcs.Admin.ListBookings(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListMovies godoc
// @Summary      List all movies (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  MessageResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/admin/movies [get]
func (h *Handler) ListMovies(c *gin.Context) {
	result, err := h.Svcs.Admin.ListMovies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": result})
}

// ListUsers godoc
// @Summary      List all users (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  MessageResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/admin/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	result, err := h.Svcs.Admin.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": result})
}