package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *ChannelStore) FindExternalConversation(ctx context.Context, connectionID, externalID string) (ExternalConversation, error) {
	return scanExternalConversation(s.db.QueryRow(ctx, `
		SELECT id, user_id, channel_connection_id, conversation_id, external_conversation_id,
			external_conversation_type, external_title, last_message_at, status, metadata, created_at, updated_at
		FROM external_conversations
		WHERE channel_connection_id = $1 AND external_conversation_id = $2 AND status <> 'deleted'
	`, connectionID, externalID))
}

func (s *ChannelStore) CreateExternalConversation(ctx context.Context, userID, connectionID, conversationID, externalID, externalType string, externalTitle *string, metadata json.RawMessage) (ExternalConversation, error) {
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	return scanExternalConversation(s.db.QueryRow(ctx, `
		INSERT INTO external_conversations (
			id, user_id, channel_connection_id, conversation_id, external_conversation_id,
			external_conversation_type, external_title, last_message_at, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8)
		RETURNING id, user_id, channel_connection_id, conversation_id, external_conversation_id,
			external_conversation_type, external_title, last_message_at, status, metadata, created_at, updated_at
	`, uuid.NewString(), userID, connectionID, conversationID, externalID, externalType, externalTitle, metadata))
}

func (s *ChannelStore) ListExternalConversations(ctx context.Context, userID, connectionID string, limit int) ([]ExternalConversation, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, channel_connection_id, conversation_id, external_conversation_id,
			external_conversation_type, external_title, last_message_at, status, metadata, created_at, updated_at
		FROM external_conversations
		WHERE user_id = $1 AND channel_connection_id = $2 AND status <> 'deleted'
		ORDER BY updated_at DESC
		LIMIT $3
	`, userID, connectionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	conversations := make([]ExternalConversation, 0)
	for rows.Next() {
		conversation, err := scanExternalConversation(rows)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}
