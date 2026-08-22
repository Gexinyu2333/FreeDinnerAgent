package llm

import (
	"context"
	"encoding/json"

	"freedinner/backend/internal/store"
)

func (s *Service) createFailedTurn(ctx context.Context, userID, conversationID string, userMessageID *string, agentConfigID *string, providerID *string, code, message string) (store.AgentTurn, error) {
	turn, err := s.harness.CreateTurn(ctx, store.AgentTurnCreate{
		UserID:         userID,
		ConversationID: conversationID,
		UserMessageID:  userMessageID,
		AgentConfigID:  agentConfigID,
		ProviderID:     providerID,
	})
	if err != nil {
		return store.AgentTurn{}, err
	}
	_, _ = s.harness.AddEvent(ctx, event(turn, "turn_started", map[string]any{
		"mode": "minimal_llm",
	}))
	_, _ = s.harness.StartTurn(ctx, turn.ID, userID, conversationID)
	_, _ = s.harness.AddEvent(ctx, event(turn, "turn_failed", map[string]any{
		"code":  code,
		"error": message,
	}))
	return s.harness.FinishTurn(ctx, turn.ID, userID, conversationID, "failed", nil, &message)
}

func event(turn store.AgentTurn, eventType string, payload map[string]any) store.AgentEventCreate {
	rawPayload, _ := json.Marshal(payload)
	return store.AgentEventCreate{
		TurnID:         turn.ID,
		UserID:         turn.UserID,
		ConversationID: turn.ConversationID,
		EventType:      eventType,
		Payload:        rawPayload,
	}
}
