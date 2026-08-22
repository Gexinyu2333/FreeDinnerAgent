package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *ChannelStore) CreateEndpoint(ctx context.Context, input ChannelConnectionEndpointCreate) (ChannelConnectionEndpoint, error) {
	if len(input.EncryptedConfig) == 0 {
		input.EncryptedConfig = json.RawMessage(`{}`)
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	return scanChannelConnectionEndpoint(s.db.QueryRow(ctx, `
		INSERT INTO channel_connection_endpoints (
			id, user_id, channel_connection_id, endpoint_type, display_name, direction,
			transport, url, encrypted_config, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (channel_connection_id, endpoint_type) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			direction = EXCLUDED.direction,
			transport = EXCLUDED.transport,
			url = EXCLUDED.url,
			encrypted_config = EXCLUDED.encrypted_config,
			metadata = EXCLUDED.metadata,
			status = 'active',
			updated_at = NOW()
		RETURNING id, user_id, channel_connection_id, endpoint_type, display_name, direction,
			transport, url, encrypted_config, status, metadata, created_at, updated_at
	`, uuid.NewString(), input.UserID, input.ChannelConnectionID, input.EndpointType, input.DisplayName,
		input.Direction, input.Transport, input.URL, input.EncryptedConfig, input.Metadata))
}

func (s *ChannelStore) ListEndpoints(ctx context.Context, userID, connectionID string) ([]ChannelConnectionEndpoint, error) {
	if _, err := s.FindUserConnectionByID(ctx, userID, connectionID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, channel_connection_id, endpoint_type, display_name, direction,
			transport, url, encrypted_config, status, metadata, created_at, updated_at
		FROM channel_connection_endpoints
		WHERE user_id = $1 AND channel_connection_id = $2 AND status <> 'deleted'
		ORDER BY endpoint_type ASC
	`, userID, connectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	endpoints := make([]ChannelConnectionEndpoint, 0)
	for rows.Next() {
		endpoint, err := scanChannelConnectionEndpoint(rows)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}
	return endpoints, rows.Err()
}

func (s *ChannelStore) FindEndpointByType(ctx context.Context, userID, connectionID, endpointType string) (ChannelConnectionEndpoint, error) {
	return scanChannelConnectionEndpoint(s.db.QueryRow(ctx, `
		SELECT id, user_id, channel_connection_id, endpoint_type, display_name, direction,
			transport, url, encrypted_config, status, metadata, created_at, updated_at
		FROM channel_connection_endpoints
		WHERE user_id = $1 AND channel_connection_id = $2 AND endpoint_type = $3 AND status = 'active'
	`, userID, connectionID, endpointType))
}
