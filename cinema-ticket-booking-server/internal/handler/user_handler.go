package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"cinema-ticket-booking/internal/model"
)

type UserProfile = model.User

// UpsertUser creates or updates the authenticated user's profile in MongoDB.
// Called by the frontend after every Google login or session restore.
// @Summary      Upsert current user profile
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  UserProfile
// @Failure      401  {object}  MessageResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/users/me [post]
func (h *Handler) UpsertUser(c *gin.Context) {
	uid := c.GetString("uid")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, MessageResponse{Message: "unauthorized"})
		return
	}

	// Fetch full profile from Firebase Admin SDK (email, name, photoURL).
	firebaseUser, err := h.AuthCl.GetUser(c.Request.Context(), uid)
	if err != nil {
		log.Printf("[UpsertUser] Firebase GetUser error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "could not fetch user info"})
		return
	}

	user, err := h.Svcs.User.UpsertUser(
		c.Request.Context(),
		uid,
		firebaseUser.Email,
		firebaseUser.DisplayName,
		firebaseUser.PhotoURL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "could not save user"})
		return
	}

	c.JSON(http.StatusOK, user)
}
