package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *ChannelStore) CreateOutboxMessage(ctx context.Context, message ChannelOutboxMessage) (ChannelOutboxMessage, error) {
	if len(message.Payload) == 0 {
		message.Payload = json.RawMessage(`{}`)
	}
	return scanChannelOutboxMessage(s.db.QueryRow(ctx, `
		INSERT INTO channel_outbox_messages (
			id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
	`, uuid.NewString(), message.UserID, message.ChannelConnectionID, message.ExternalConversationID,
		message.ConversationID, message.AgentTurnID, message.ReplyToInboxEventID, message.MessageType,
		message.Content, message.Payload, message.RequiresApproval, message.Status))
}

func (s *ChannelStore) ListOutboxMessages(ctx context.Context, userID, connectionID string, status *string, limit int) ([]ChannelOutboxMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
		FROM channel_outbox_messages
		WHERE user_id = $1 AND channel_connection_id = $2 AND ($3::text IS NULL OR status = $3)
		ORDER BY created_at DESC
		LIMIT $4
	`, userID, connectionID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []ChannelOutboxMessage
	for rows.Next() {
		message, err := scanChannelOutboxMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *ChannelStore) ListApprovedOutboxMessages(ctx context.Context, limit int) ([]ChannelOutboxMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
		FROM channel_outbox_messages
		WHERE status = 'approved'
		ORDER BY COALESCE(approved_at, created_at) ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []ChannelOutboxMessage
	for rows.Next() {
		message, err := scanChannelOutboxMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *ChannelStore) FindOutboxMessage(ctx context.Context, userID, outboxID string) (ChannelOutboxMessage, error) {
	return scanChannelOutboxMessage(s.db.QueryRow(ctx, `
		SELECT id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
		FROM channel_outbox_messages
		WHERE id = $1 AND user_id = $2
	`, outboxID, userID))
}

func (s *ChannelStore) ResolveOutboxMessage(ctx context.Context, userID, outboxID, status string) (ChannelOutboxMessage, error) {
	return scanChannelOutboxMessage(s.db.QueryRow(ctx, `
		UPDATE channel_outbox_messages
		SET status = $3,
			approved_at = CASE WHEN $3 = 'approved' THEN NOW() ELSE approved_at END,
			error_message = CASE WHEN $3 = 'cancelled' THEN 'cancelled by user' ELSE error_message END
		WHERE id = $1 AND user_id = $2 AND status = 'pending'
		RETURNING id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
	`, outboxID, userID, status))
}

func (s *ChannelStore) MarkOutboxSending(ctx context.Context, userID, outboxID string) (ChannelOutboxMessage, error) {
	return scanChannelOutboxMessage(s.db.QueryRow(ctx, `
		UPDATE channel_outbox_messages
		SET status = 'sending'
		WHERE id = $1 AND user_id = $2 AND status = 'approved'
		RETURNING id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
	`, outboxID, userID))
}

func (s *ChannelStore) MarkOutboxSent(ctx context.Context, userID, outboxID string, externalMessageID *string) (ChannelOutboxMessage, error) {
	return scanChannelOutboxMessage(s.db.QueryRow(ctx, `
		UPDATE channel_outbox_messages
		SET status = 'sent', external_message_id = $3, sent_at = NOW(), error_message = NULL
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
	`, outboxID, userID, externalMessageID))
}

func (s *ChannelStore) MarkOutboxFailed(ctx context.Context, userID, outboxID string, errorMessage string) (ChannelOutboxMessage, error) {
	return scanChannelOutboxMessage(s.db.QueryRow(ctx, `
		UPDATE channel_outbox_messages
		SET status = 'failed', error_message = $3
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
	`, outboxID, userID, errorMessage))
}
