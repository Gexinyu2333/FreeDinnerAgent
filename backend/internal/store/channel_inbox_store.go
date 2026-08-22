package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func (s *ChannelStore) CreateInboxEvent(ctx context.Context, event ChannelInboxEvent) (ChannelInboxEvent, error) {
	if len(event.RawPayload) == 0 {
		event.RawPayload = json.RawMessage(`{}`)
	}
	return scanChannelInboxEvent(s.db.QueryRow(ctx, `
		INSERT INTO channel_inbox_events (
			id, user_id, channel_connection_id, external_conversation_id, conversation_id, message_id,
			event_type, external_event_id, external_sender_id, external_sender_name, raw_payload,
			normalized_text, should_trigger_agent, trigger_reason, status, processed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (channel_connection_id, external_event_id) DO UPDATE SET
			status = channel_inbox_events.status
		RETURNING id, user_id, channel_connection_id, external_conversation_id, conversation_id, message_id,
			event_type, external_event_id, external_sender_id, external_sender_name, raw_payload,
			normalized_text, should_trigger_agent, trigger_reason, status, received_at, processed_at
	`, uuid.NewString(), event.UserID, event.ChannelConnectionID, event.ExternalConversationID, event.ConversationID,
		event.MessageID, event.EventType, event.ExternalEventID, event.ExternalSenderID, event.ExternalSenderName,
		event.RawPayload, event.NormalizedText, event.ShouldTriggerAgent, event.TriggerReason, event.Status, event.ProcessedAt))
}

func (s *ChannelStore) ListInboxEvents(ctx context.Context, userID, connectionID string, limit int) ([]ChannelInboxEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, channel_connection_id, external_conversation_id, conversation_id, message_id,
			event_type, external_event_id, external_sender_id, external_sender_name, raw_payload,
			normalized_text, should_trigger_agent, trigger_reason, status, received_at, processed_at
		FROM channel_inbox_events
		WHERE user_id = $1 AND channel_connection_id = $2
		ORDER BY received_at DESC
		LIMIT $3
	`, userID, connectionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []ChannelInboxEvent
	for rows.Next() {
		event, err := scanChannelInboxEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *ChannelStore) CountRecentTriggeredInboxEvents(ctx context.Context, userID, connectionID, scopeType string, externalScopeID *string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM channel_inbox_events cie
		LEFT JOIN external_conversations ec ON ec.id = cie.external_conversation_id
		WHERE cie.user_id = $1
		  AND cie.channel_connection_id = $2
		  AND cie.should_trigger_agent = TRUE
		  AND cie.received_at >= $5
		  AND (
		  	$3::text = 'all'
		  	OR ec.external_conversation_type = $3
		  	OR ($4::text IS NOT NULL AND ec.external_conversation_id = $4)
		  )
	`, userID, connectionID, scopeType, externalScopeID, since).Scan(&count)
	return count, err
}

func (s *ChannelStore) CountRecentTriggeredInboxEventsForUser(ctx context.Context, userID string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM channel_inbox_events
		WHERE user_id = $1
		  AND should_trigger_agent = TRUE
		  AND received_at >= $2
	`, userID, since).Scan(&count)
	return count, err
}
