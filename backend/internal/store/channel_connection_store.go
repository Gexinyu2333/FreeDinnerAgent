package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *ChannelStore) CreateConnection(ctx context.Context, input ChannelConnectionCreate) (ChannelConnection, error) {
	if len(input.EncryptedConfig) == 0 {
		input.EncryptedConfig = json.RawMessage(`{}`)
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	query := `
		INSERT INTO channel_connections (
			id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, status, last_health_status, last_event_at, last_checked_at, metadata,
			created_at, updated_at
	`
	return scanChannelConnection(s.db.QueryRow(ctx, query, uuid.NewString(), input.UserID, input.ProviderID,
		input.DisplayName, input.ExternalAccountID, input.ExternalAccountName, input.EncryptedConfig, input.Metadata))
}

func (s *ChannelStore) ListConnections(ctx context.Context, userID string) ([]ChannelConnection, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, status, last_health_status, last_event_at, last_checked_at, metadata,
			created_at, updated_at
		FROM channel_connections
		WHERE user_id = $1 AND status <> 'deleted'
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	connections := make([]ChannelConnection, 0)
	for rows.Next() {
		connection, err := scanChannelConnection(rows)
		if err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}
	return connections, rows.Err()
}

func (s *ChannelStore) UpdateConnection(ctx context.Context, input ChannelConnectionUpdate) (ChannelConnection, error) {
	if len(input.EncryptedConfig) == 0 {
		input.EncryptedConfig = json.RawMessage(`{}`)
	}
	return scanChannelConnection(s.db.QueryRow(ctx, `
		UPDATE channel_connections
		SET display_name = $3,
			external_account_id = $4,
			external_account_name = $5,
			encrypted_config = $6,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
		RETURNING id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, status, last_health_status, last_event_at, last_checked_at, metadata,
			created_at, updated_at
	`, input.ConnectionID, input.UserID, input.DisplayName, input.ExternalAccountID,
		input.ExternalAccountName, input.EncryptedConfig))
}

func (s *ChannelStore) DeleteConnection(ctx context.Context, userID, connectionID string) error {
	commandTag, err := s.db.Exec(ctx, `
		UPDATE channel_connections
		SET status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
	`, connectionID, userID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ChannelStore) FindConnectionByID(ctx context.Context, connectionID string) (ChannelConnection, error) {
	return scanChannelConnection(s.db.QueryRow(ctx, `
		SELECT id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, status, last_health_status, last_event_at, last_checked_at, metadata,
			created_at, updated_at
		FROM channel_connections
		WHERE id = $1 AND status <> 'deleted'
	`, connectionID))
}

func (s *ChannelStore) FindUserConnectionByID(ctx context.Context, userID, connectionID string) (ChannelConnection, error) {
	return scanChannelConnection(s.db.QueryRow(ctx, `
		SELECT id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, status, last_health_status, last_event_at, last_checked_at, metadata,
			created_at, updated_at
		FROM channel_connections
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
	`, connectionID, userID))
}
