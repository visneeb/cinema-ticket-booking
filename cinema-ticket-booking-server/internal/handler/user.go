package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// UserProfile is the document stored in the cinema.users collection.
type UserProfile struct {
	UID         string    `bson:"_id"          json:"uid"`
	Email       string    `bson:"email"        json:"email"`
	DisplayName string    `bson:"display_name" json:"display_name"`
	PhotoURL    string    `bson:"photo_url"    json:"photo_url"`
	CreatedAt   time.Time `bson:"created_at"   json:"created_at"`
	LastLoginAt time.Time `bson:"last_login_at" json:"last_login_at"`
}

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

	// Fetch full profile from Firebase Admin SDK (email, name, photoURL)
	firebaseUser, err := h.AuthCl.GetUser(c.Request.Context(), uid)
	if err != nil {
		log.Printf("[UpsertUser] GetUser error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "could not fetch user info"})
		return
	}

	now := time.Now().UTC()

	// Upsert: $set updates on every login, $setOnInsert runs only on first creation
	filter := bson.D{{Key: "_id", Value: uid}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "email",        Value: firebaseUser.Email},
			{Key: "display_name", Value: firebaseUser.DisplayName},
			{Key: "photo_url",    Value: firebaseUser.PhotoURL},
			{Key: "last_login_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "created_at", Value: now},
		}},
	}

	_, err = h.DB.Collection("users").UpdateOne(
		c.Request.Context(),
		filter,
		update,
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		log.Printf("[UpsertUser] mongodb upsert error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "could not save user"})
		return
	}

	log.Printf("[UpsertUser] upserted uid=%s email=%s", uid, firebaseUser.Email)
	c.JSON(http.StatusOK, UserProfile{
		UID:         uid,
		Email:       firebaseUser.Email,
		DisplayName: firebaseUser.DisplayName,
		PhotoURL:    firebaseUser.PhotoURL,
	})
}
