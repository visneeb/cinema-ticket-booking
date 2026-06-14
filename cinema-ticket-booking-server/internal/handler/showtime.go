package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ShowtimeResponse is returned by GET /api/showtimes.
type ShowtimeResponse struct {
	ID          string    `json:"id"`
	MovieID     string    `json:"movie_id"`
	StartsAt    time.Time `json:"starts_at"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

// GetShowtimes returns all showtimes joined with their movie title.
// @Summary      List all showtimes
// @Tags         showtimes
// @Produce      json
// @Success      200  {array}   ShowtimeResponse
// @Failure      500  {object}  MessageResponse
// @Router       /api/showtimes [get]
func (h *Handler) GetShowtimes(c *gin.Context) {
	ctx := c.Request.Context()

	// Fetch all showtime documents
	cursor, err := h.DB.Collection("showtimes").Find(ctx, bson.D{})
	if err != nil {
		log.Printf("[GetShowtimes] mongo find error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "failed to fetch showtimes"})
		return
	}
	defer cursor.Close(ctx)

	var showtimes []struct {
		ID       bson.ObjectID `bson:"_id"`
		MovieID  bson.ObjectID `bson:"movie_id"`
		StartsAt time.Time     `bson:"starts_at"`
	}
	if err := cursor.All(ctx, &showtimes); err != nil {
		log.Printf("[GetShowtimes] cursor decode error: %v", err)
		c.JSON(http.StatusInternalServerError, MessageResponse{Message: "failed to read showtimes"})
		return
	}

	// Build a unique set of movie IDs for a single batch lookup
	movieIDSet := make(map[bson.ObjectID]struct{}, len(showtimes))
	for _, st := range showtimes {
		movieIDSet[st.MovieID] = struct{}{}
	}
	movieIDs := make([]bson.ObjectID, 0, len(movieIDSet))
	for id := range movieIDSet {
		movieIDs = append(movieIDs, id)
	}

	// Fetch all referenced movies in one query
	movieMap := make(map[string]struct {
		Title       string
		Description string
	})
	if len(movieIDs) > 0 {
		mc, err := h.DB.Collection("movies").Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: movieIDs}}}})
		if err == nil {
			defer mc.Close(ctx)
			var movies []struct {
				ID          bson.ObjectID `bson:"_id"`
				Title       string        `bson:"title"`
				Description string        `bson:"description"`
			}
			if err := mc.All(ctx, &movies); err == nil {
				for _, m := range movies {
					movieMap[m.ID.Hex()] = struct {
						Title       string
						Description string
					}{Title: m.Title, Description: m.Description}
				}
			}
		}
	}

	result := make([]ShowtimeResponse, len(showtimes))
	for i, st := range showtimes {
		movie := movieMap[st.MovieID.Hex()]
		result[i] = ShowtimeResponse{
			ID:          st.ID.Hex(),
			MovieID:     st.MovieID.Hex(),
			StartsAt:    st.StartsAt,
			Title:       movie.Title,
			Description: movie.Description,
		}
	}
	c.JSON(http.StatusOK, result)
}
