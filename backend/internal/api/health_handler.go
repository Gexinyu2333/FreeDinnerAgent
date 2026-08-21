package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func healthHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := deps.DB.Ping(ctx); err != nil {
			Error(c, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "database is unavailable")
			return
		}

		OK(c, gin.H{
			"status": "ok",
			"env":    deps.Config.AppEnv,
		})
	}
}
