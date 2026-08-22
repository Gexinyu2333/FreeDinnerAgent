package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *HarnessStore) CreateTurn(ctx context.Context, input AgentTurnCreate) (AgentTurn, error) {
	query := `
		INSERT INTO agent_turns (id, user_id, conversation_id, user_message_id, agent_config_id, provider_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, conversation_id, user_message_id, assistant_message_id, agent_config_id,
		          provider_id, status, cancel_requested, context_build_id, error_message,
		          created_at, started_at, finished_at
	`
	return scanAgentTurn(s.db.QueryRow(ctx, query, uuid.NewString(), input.UserID, input.ConversationID,
		input.UserMessageID, input.AgentConfigID, input.ProviderID))
}

func (s *HarnessStore) StartTurn(ctx context.Context, turnID, userID, conversationID string) (AgentTurn, error) {
	query := `
		UPDATE agent_turns
		SET status = 'running', started_at = COALESCE(started_at, NOW())
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
		RETURNING id, user_id, conversation_id, user_message_id, assistant_message_id, agent_config_id,
		          provider_id, status, cancel_requested, context_build_id, error_message,
		          created_at, started_at, finished_at
	`
	return scanAgentTurn(s.db.QueryRow(ctx, query, turnID, userID, conversationID))
}

func (s *HarnessStore) FinishTurn(ctx context.Context, turnID, userID, conversationID, status string, assistantMessageID *string, errorMessage *string) (AgentTurn, error) {
	query := `
		UPDATE agent_turns
		SET status = $4,
		    assistant_message_id = COALESCE($5, assistant_message_id),
		    error_message = $6,
		    finished_at = NOW()
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
		RETURNING id, user_id, conversation_id, user_message_id, assistant_message_id, agent_config_id,
		          provider_id, status, cancel_requested, context_build_id, error_message,
		          created_at, started_at, finished_at
	`
	return scanAgentTurn(s.db.QueryRow(ctx, query, turnID, userID, conversationID, status, assistantMessageID, errorMessage))
}

func (s *HarnessStore) SetTurnStatus(ctx context.Context, turnID, userID, conversationID, status string, errorMessage *string) (AgentTurn, error) {
	query := `
		UPDATE agent_turns
		SET status = $4,
		    error_message = $5
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
		RETURNING id, user_id, conversation_id, user_message_id, assistant_message_id, agent_config_id,
		          provider_id, status, cancel_requested, context_build_id, error_message,
		          created_at, started_at, finished_at
	`
	return scanAgentTurn(s.db.QueryRow(ctx, query, turnID, userID, conversationID, status, errorMessage))
}

func (s *HarnessStore) GetTurn(ctx context.Context, userID, conversationID, turnID string) (AgentTurn, error) {
	query := `
		SELECT id, user_id, conversation_id, user_message_id, assistant_message_id, agent_config_id,
		       provider_id, status, cancel_requested, context_build_id, error_message,
		       created_at, started_at, finished_at
		FROM agent_turns
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
	`
	return scanAgentTurn(s.db.QueryRow(ctx, query, turnID, userID, conversationID))
}
