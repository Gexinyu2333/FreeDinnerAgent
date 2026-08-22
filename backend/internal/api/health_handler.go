package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func healthHandler(appEnv string, db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			Error(c, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "database is unavailable")
			return
		}

		OK(c, gin.H{
			"status": "ok",
			"env":    appEnv,
		})
	}
}
