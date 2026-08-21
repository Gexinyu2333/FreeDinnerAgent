package api

import (
	"net/http"
	"strings"

	"freedinner/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

const userIDContextKey = "user_id"

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header")
			c.Abort()
			return
		}

		claims, err := auth.ParseAccessToken(secret, parts[1])
		if err != nil {
			Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(userIDContextKey, claims.UserID)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) (string, bool) {
	userID, ok := c.Get(userIDContextKey)
	if !ok {
		return "", false
	}
	value, ok := userID.(string)
	return value, ok && value != ""
}
