package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *HarnessStore) CreateLoopStep(ctx context.Context, input AgentLoopStepCreate) (AgentLoopStep, error) {
	if input.Status == "" {
		input.Status = "running"
	}
	query := `
		INSERT INTO agent_loop_steps (
			id, turn_id, user_id, conversation_id, step_no, step_type,
			thought_summary, action_type, token_count, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, turn_id, user_id, conversation_id, step_no, step_type, thought_summary,
		          action_type, action_ref_id, observation, token_count, status, error_message,
		          created_at, finished_at
	`
	return scanAgentLoopStep(s.db.QueryRow(ctx, query, uuid.NewString(), input.TurnID, input.UserID,
		input.ConversationID, input.StepNo, input.StepType, input.ThoughtSummary, input.ActionType,
		input.TokenCount, input.Status))
}

func (s *HarnessStore) FinishLoopStep(ctx context.Context, stepID, userID, conversationID, status string, observation *string, errorMessage *string) (AgentLoopStep, error) {
	query := `
		UPDATE agent_loop_steps
		SET status = $4, observation = $5, error_message = $6, finished_at = NOW()
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
		RETURNING id, turn_id, user_id, conversation_id, step_no, step_type, thought_summary,
		          action_type, action_ref_id, observation, token_count, status, error_message,
		          created_at, finished_at
	`
	return scanAgentLoopStep(s.db.QueryRow(ctx, query, stepID, userID, conversationID, status, observation, errorMessage))
}

func (s *HarnessStore) SetLoopStepActionRef(ctx context.Context, stepID, userID, conversationID string, actionRefID *string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE agent_loop_steps
		SET action_ref_id = $4
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
	`, stepID, userID, conversationID, actionRefID)
	return err
}

func (s *HarnessStore) ListLoopSteps(ctx context.Context, userID, conversationID, turnID string) ([]AgentLoopStep, error) {
	if _, err := s.GetTurn(ctx, userID, conversationID, turnID); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, turn_id, user_id, conversation_id, step_no, step_type, thought_summary,
		       action_type, action_ref_id, observation, token_count, status, error_message,
		       created_at, finished_at
		FROM agent_loop_steps
		WHERE user_id = $1 AND conversation_id = $2 AND turn_id = $3
		ORDER BY step_no ASC
	`, userID, conversationID, turnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	steps := make([]AgentLoopStep, 0)
	for rows.Next() {
		step, err := scanAgentLoopStep(rows)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}
