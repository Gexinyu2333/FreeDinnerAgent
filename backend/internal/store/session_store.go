package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionStore struct {
	db *pgxpool.Pool
}

func NewSessionStore(db *pgxpool.Pool) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(ctx context.Context, id, userID, refreshTokenHash, userAgent string, expiresAt time.Time) error {
	query := `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := s.db.Exec(ctx, query, id, userID, refreshTokenHash, userAgent, expiresAt)
	return err
}
