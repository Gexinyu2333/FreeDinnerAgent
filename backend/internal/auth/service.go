package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"freedinner/backend/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrUsernameTaken     = errors.New("username already exists")
	ErrInvalidCredential = errors.New("invalid username or password")
	ErrUserDisabled      = errors.New("user is disabled")
)

type Service struct {
	users    *store.UserStore
	sessions *store.SessionStore
	secret   string
}

type RegisterInput struct {
	Username    string
	Password    string
	DisplayName *string
	UserAgent   string
}

type LoginInput struct {
	Username  string
	Password  string
	UserAgent string
}

type AuthResult struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	User         PublicUser `json:"user"`
}

type PublicUser struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name"`
}

func NewService(users *store.UserStore, sessions *store.SessionStore, secret string) *Service {
	return &Service{
		users:    users,
		sessions: sessions,
		secret:   secret,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	username := strings.TrimSpace(input.Username)
	if len(username) < 3 || len(username) > 64 || len(input.Password) < 8 {
		return AuthResult{}, ErrInvalidInput
	}

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return AuthResult{}, err
	}

	user, err := s.users.Create(ctx, store.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: passwordHash,
		DisplayName:  normalizeDisplayName(input.DisplayName),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return AuthResult{}, ErrUsernameTaken
		}
		return AuthResult{}, err
	}

	if err := s.users.CreateDefaultAgentConfig(ctx, uuid.NewString(), user.ID); err != nil {
		return AuthResult{}, err
	}

	return s.issueTokens(ctx, user, input.UserAgent)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	user, err := s.users.FindByUsername(ctx, strings.TrimSpace(input.Username))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return AuthResult{}, ErrInvalidCredential
		}
		return AuthResult{}, err
	}

	if user.Status != "active" {
		return AuthResult{}, ErrUserDisabled
	}
	if !CheckPassword(input.Password, user.PasswordHash) {
		return AuthResult{}, ErrInvalidCredential
	}

	if err := s.users.UpdateLastLogin(ctx, user.ID); err != nil {
		return AuthResult{}, err
	}

	return s.issueTokens(ctx, user, input.UserAgent)
}

func (s *Service) CurrentUser(ctx context.Context, userID string) (PublicUser, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	return toPublicUser(user), nil
}

func (s *Service) issueTokens(ctx context.Context, user store.User, userAgent string) (AuthResult, error) {
	accessToken, err := GenerateAccessToken(s.secret, user.ID, user.Username, 24*time.Hour)
	if err != nil {
		return AuthResult{}, err
	}

	refreshToken, refreshTokenHash, err := GenerateRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}

	if err := s.sessions.Create(ctx, uuid.NewString(), user.ID, refreshTokenHash, userAgent, time.Now().Add(30*24*time.Hour)); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toPublicUser(user),
	}, nil
}

func toPublicUser(user store.User) PublicUser {
	return PublicUser{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	}
}

func normalizeDisplayName(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
