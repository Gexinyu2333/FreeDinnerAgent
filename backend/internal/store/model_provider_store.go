package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ModelProvider struct {
	ID                       string    `json:"id"`
	UserID                   string    `json:"user_id"`
	Provider                 string    `json:"provider"`
	DisplayName              string    `json:"display_name"`
	ChatBaseURL              *string   `json:"chat_base_url"`
	EncryptedChatAPIKey      string    `json:"-"`
	EmbeddingBaseURL         *string   `json:"embedding_base_url"`
	EncryptedEmbeddingAPIKey *string   `json:"-"`
	DefaultChatModel         string    `json:"default_chat_model"`
	DefaultEmbeddingModel    *string   `json:"default_embedding_model"`
	IsDefault                bool      `json:"is_default"`
	Status                   string    `json:"status"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type PublicModelProvider struct {
	ID                    string  `json:"id"`
	Provider              string  `json:"provider"`
	DisplayName           string  `json:"display_name"`
	ChatBaseURL           *string `json:"chat_base_url"`
	EmbeddingBaseURL      *string `json:"embedding_base_url"`
	DefaultChatModel      string  `json:"default_chat_model"`
	DefaultEmbeddingModel *string `json:"default_embedding_model"`
	IsDefault             bool    `json:"is_default"`
	HasChatAPIKey         bool    `json:"has_chat_api_key"`
	HasEmbeddingAPIKey    bool    `json:"has_embedding_api_key"`
	Status                string  `json:"status"`
}

type ModelProviderCreate struct {
	Provider                 string
	DisplayName              string
	ChatBaseURL              *string
	EncryptedChatAPIKey      string
	EmbeddingBaseURL         *string
	EncryptedEmbeddingAPIKey *string
	DefaultChatModel         string
	DefaultEmbeddingModel    *string
	IsDefault                bool
}

type ModelProviderUpdate struct {
	DisplayName              *string
	ChatBaseURL              **string
	EncryptedChatAPIKey      *string
	EmbeddingBaseURL         **string
	EncryptedEmbeddingAPIKey **string
	DefaultChatModel         *string
	DefaultEmbeddingModel    **string
	IsDefault                *bool
	Status                   *string
}

type ModelProviderStore struct {
	db *pgxpool.Pool
}

func NewModelProviderStore(db *pgxpool.Pool) *ModelProviderStore {
	return &ModelProviderStore{db: db}
}

func (s *ModelProviderStore) List(ctx context.Context, userID string) ([]ModelProvider, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, provider, display_name, chat_base_url, encrypted_chat_api_key,
		       embedding_base_url, encrypted_embedding_api_key, default_chat_model,
		       default_embedding_model, is_default, status, created_at, updated_at
		FROM user_model_providers
		WHERE user_id = $1 AND status <> 'deleted'
		ORDER BY is_default DESC, created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providers := make([]ModelProvider, 0)
	for rows.Next() {
		provider, err := scanModelProvider(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

func (s *ModelProviderStore) Create(ctx context.Context, userID string, input ModelProviderCreate) (ModelProvider, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return ModelProvider{}, err
	}
	defer tx.Rollback(ctx)

	if input.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE user_model_providers SET is_default = FALSE, updated_at = NOW() WHERE user_id = $1`, userID); err != nil {
			return ModelProvider{}, err
		}
	}

	query := `
		INSERT INTO user_model_providers (
			id, user_id, provider, display_name, chat_base_url, encrypted_chat_api_key,
			embedding_base_url, encrypted_embedding_api_key, default_chat_model,
			default_embedding_model, is_default
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, user_id, provider, display_name, chat_base_url, encrypted_chat_api_key,
		          embedding_base_url, encrypted_embedding_api_key, default_chat_model,
		          default_embedding_model, is_default, status, created_at, updated_at
	`
	provider, err := scanModelProvider(tx.QueryRow(ctx, query, uuid.NewString(), userID, input.Provider, input.DisplayName,
		input.ChatBaseURL, input.EncryptedChatAPIKey, input.EmbeddingBaseURL, input.EncryptedEmbeddingAPIKey,
		input.DefaultChatModel, input.DefaultEmbeddingModel, input.IsDefault))
	if err != nil {
		return ModelProvider{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ModelProvider{}, err
	}
	return provider, nil
}

func (s *ModelProviderStore) Update(ctx context.Context, userID, providerID string, update ModelProviderUpdate) (ModelProvider, error) {
	current, err := s.FindByID(ctx, userID, providerID)
	if err != nil {
		return ModelProvider{}, err
	}

	displayName := current.DisplayName
	if update.DisplayName != nil {
		displayName = *update.DisplayName
	}
	chatBaseURL := current.ChatBaseURL
	if update.ChatBaseURL != nil {
		chatBaseURL = *update.ChatBaseURL
	}
	encryptedChatAPIKey := current.EncryptedChatAPIKey
	if update.EncryptedChatAPIKey != nil {
		encryptedChatAPIKey = *update.EncryptedChatAPIKey
	}
	embeddingBaseURL := current.EmbeddingBaseURL
	if update.EmbeddingBaseURL != nil {
		embeddingBaseURL = *update.EmbeddingBaseURL
	}
	encryptedEmbeddingAPIKey := current.EncryptedEmbeddingAPIKey
	if update.EncryptedEmbeddingAPIKey != nil {
		encryptedEmbeddingAPIKey = *update.EncryptedEmbeddingAPIKey
	}
	defaultChatModel := current.DefaultChatModel
	if update.DefaultChatModel != nil {
		defaultChatModel = *update.DefaultChatModel
	}
	defaultEmbeddingModel := current.DefaultEmbeddingModel
	if update.DefaultEmbeddingModel != nil {
		defaultEmbeddingModel = *update.DefaultEmbeddingModel
	}
	isDefault := current.IsDefault
	if update.IsDefault != nil {
		isDefault = *update.IsDefault
	}
	status := current.Status
	if update.Status != nil {
		status = *update.Status
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return ModelProvider{}, err
	}
	defer tx.Rollback(ctx)

	if isDefault {
		if _, err := tx.Exec(ctx, `UPDATE user_model_providers SET is_default = FALSE, updated_at = NOW() WHERE user_id = $1 AND id <> $2`, userID, providerID); err != nil {
			return ModelProvider{}, err
		}
	}

	query := `
		UPDATE user_model_providers
		SET display_name = $3,
		    chat_base_url = $4,
		    encrypted_chat_api_key = $5,
		    embedding_base_url = $6,
		    encrypted_embedding_api_key = $7,
		    default_chat_model = $8,
		    default_embedding_model = $9,
		    is_default = $10,
		    status = $11,
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, provider, display_name, chat_base_url, encrypted_chat_api_key,
		          embedding_base_url, encrypted_embedding_api_key, default_chat_model,
		          default_embedding_model, is_default, status, created_at, updated_at
	`
	provider, err := scanModelProvider(tx.QueryRow(ctx, query, providerID, userID, displayName, chatBaseURL,
		encryptedChatAPIKey, embeddingBaseURL, encryptedEmbeddingAPIKey, defaultChatModel, defaultEmbeddingModel,
		isDefault, status))
	if err != nil {
		return ModelProvider{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ModelProvider{}, err
	}
	return provider, nil
}

func (s *ModelProviderStore) Delete(ctx context.Context, userID, providerID string) error {
	result, err := s.db.Exec(ctx, `
		UPDATE user_model_providers
		SET status = 'deleted', is_default = FALSE, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, providerID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ModelProviderStore) FindByID(ctx context.Context, userID, providerID string) (ModelProvider, error) {
	query := `
		SELECT id, user_id, provider, display_name, chat_base_url, encrypted_chat_api_key,
		       embedding_base_url, encrypted_embedding_api_key, default_chat_model,
		       default_embedding_model, is_default, status, created_at, updated_at
		FROM user_model_providers
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
	`
	return scanModelProvider(s.db.QueryRow(ctx, query, providerID, userID))
}

func (s *ModelProviderStore) FindDefault(ctx context.Context, userID string) (ModelProvider, error) {
	query := `
		SELECT id, user_id, provider, display_name, chat_base_url, encrypted_chat_api_key,
		       embedding_base_url, encrypted_embedding_api_key, default_chat_model,
		       default_embedding_model, is_default, status, created_at, updated_at
		FROM user_model_providers
		WHERE user_id = $1 AND is_default = TRUE AND status = 'active'
		LIMIT 1
	`
	return scanModelProvider(s.db.QueryRow(ctx, query, userID))
}

func ToPublicModelProvider(provider ModelProvider) PublicModelProvider {
	return PublicModelProvider{
		ID:                    provider.ID,
		Provider:              provider.Provider,
		DisplayName:           provider.DisplayName,
		ChatBaseURL:           provider.ChatBaseURL,
		EmbeddingBaseURL:      provider.EmbeddingBaseURL,
		DefaultChatModel:      provider.DefaultChatModel,
		DefaultEmbeddingModel: provider.DefaultEmbeddingModel,
		IsDefault:             provider.IsDefault,
		HasChatAPIKey:         provider.EncryptedChatAPIKey != "",
		HasEmbeddingAPIKey:    provider.EncryptedEmbeddingAPIKey != nil && *provider.EncryptedEmbeddingAPIKey != "",
		Status:                provider.Status,
	}
}

func scanModelProvider(row pgx.Row) (ModelProvider, error) {
	var provider ModelProvider
	if err := row.Scan(
		&provider.ID,
		&provider.UserID,
		&provider.Provider,
		&provider.DisplayName,
		&provider.ChatBaseURL,
		&provider.EncryptedChatAPIKey,
		&provider.EmbeddingBaseURL,
		&provider.EncryptedEmbeddingAPIKey,
		&provider.DefaultChatModel,
		&provider.DefaultEmbeddingModel,
		&provider.IsDefault,
		&provider.Status,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ModelProvider{}, ErrNotFound
		}
		return ModelProvider{}, err
	}
	return provider, nil
}
