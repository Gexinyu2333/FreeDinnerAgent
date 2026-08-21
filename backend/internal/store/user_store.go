package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("record not found")

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	DisplayName  *string    `json:"display_name"`
	Status       string     `json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	PasswordHash string     `json:"-"`
}

type UserStore struct {
	db *pgxpool.Pool
}

func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(ctx context.Context, user User) (User, error) {
	query := `
		INSERT INTO users (id, username, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, display_name, status, last_login_at, created_at, updated_at, password_hash
	`
	return scanUser(s.db.QueryRow(ctx, query, user.ID, user.Username, user.PasswordHash, user.DisplayName))
}

func (s *UserStore) CreateDefaultAgentConfig(ctx context.Context, id, userID string) error {
	query := `
		INSERT INTO user_agent_configs (id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, name) DO NOTHING
	`
	_, err := s.db.Exec(ctx, query, id, userID)
	return err
}

func (s *UserStore) FindByUsername(ctx context.Context, username string) (User, error) {
	query := `
		SELECT id, username, display_name, status, last_login_at, created_at, updated_at, password_hash
		FROM users
		WHERE username = $1
	`
	return scanUser(s.db.QueryRow(ctx, query, username))
}

func (s *UserStore) FindByID(ctx context.Context, id string) (User, error) {
	query := `
		SELECT id, username, display_name, status, last_login_at, created_at, updated_at, password_hash
		FROM users
		WHERE id = $1
	`
	return scanUser(s.db.QueryRow(ctx, query, id))
}

func (s *UserStore) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func scanUser(row pgx.Row) (User, error) {
	var user User
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.Status,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.PasswordHash,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return user, nil
}
