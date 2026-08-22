package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *HarnessStore) AddEvent(ctx context.Context, input AgentEventCreate) (AgentEvent, error) {
	if len(input.Payload) == 0 {
		input.Payload = json.RawMessage(`{}`)
	}
	query := `
		WITH next_sequence AS (
			SELECT COALESCE(MAX(sequence_no), 0) + 1 AS sequence_no
			FROM agent_events
			WHERE turn_id = $1
		)
		INSERT INTO agent_events (id, turn_id, user_id, conversation_id, event_type, payload, sequence_no)
		SELECT $2, $1, $3, $4, $5, $6, sequence_no
		FROM next_sequence
		RETURNING id, turn_id, user_id, conversation_id, event_type, payload, sequence_no, created_at
	`
	return scanAgentEvent(s.db.QueryRow(ctx, query, input.TurnID, uuid.NewString(), input.UserID,
		input.ConversationID, input.EventType, input.Payload))
}

func (s *HarnessStore) ListEvents(ctx context.Context, userID, conversationID string, turnID *string) ([]AgentEvent, error) {
	query := `
		SELECT id, turn_id, user_id, conversation_id, event_type, payload, sequence_no, created_at
		FROM agent_events
		WHERE user_id = $1 AND conversation_id = $2 AND ($3::uuid IS NULL OR turn_id = $3)
		ORDER BY created_at ASC, sequence_no ASC
	`
	rows, err := s.db.Query(ctx, query, userID, conversationID, turnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]AgentEvent, 0)
	for rows.Next() {
		event, err := scanAgentEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
