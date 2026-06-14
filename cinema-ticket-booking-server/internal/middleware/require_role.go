package middleware

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"cinema-ticket-booking/internal/model"
)

// UserFinder is the minimal interface RequireRole needs to look up a user's role.
type UserFinder func(ctx context.Context, uid string) (model.User, error)

// RequireRole aborts with 403 unless the authenticated user's role matches.
// Must be placed after the Auth middleware (which sets the "uid" context key).
func RequireRole(findUser UserFinder, role model.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString("uid")
		if uid == "" {
			c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
			return
		}

		user, err := findUser(c.Request.Context(), uid)
		if err != nil {
			log.Printf("[RequireRole] user lookup failed uid=%s: %v", uid, err)
			c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
			return
		}

		if user.Role != role {
			log.Printf("[RequireRole] access denied uid=%s role=%s required=%s", uid, user.Role, role)
			c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
			return
		}

		c.Next()
	}
}
