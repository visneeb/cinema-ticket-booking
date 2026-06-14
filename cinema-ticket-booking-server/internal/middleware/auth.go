package middleware

import (
	"log"
	"strings"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

func Auth(authCl *firebaseAuth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "authorization header missing"})
			return
		}
		decoded, err := authCl.VerifyIDToken(c.Request.Context(), token)
		if err != nil {
			log.Printf("[Auth] VerifyIDToken failed: %v", err)
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		log.Printf("[Auth] token verified uid=%s", decoded.UID)
		c.Set("uid", decoded.UID)
		c.Next()
	}
}