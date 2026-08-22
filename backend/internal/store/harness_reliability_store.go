package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *HarnessStore) CreateValidation(ctx context.Context, input LLMOutputValidationCreate) error {
	if input.AttemptNo <= 0 {
		input.AttemptNo = 1
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO llm_output_validations (
			id, turn_id, loop_step_id, user_id, validation_type, status,
			failure_reason, repair_prompt, repaired_output, attempt_no
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, uuid.NewString(), input.TurnID, input.LoopStepID, input.UserID, input.ValidationType, input.Status,
		input.FailureReason, input.RepairPrompt, input.RepairedOutput, input.AttemptNo)
	return err
}

func (s *HarnessStore) CreateFallbackEvent(ctx context.Context, input AgentFallbackEventCreate) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO agent_fallback_events (
			id, turn_id, loop_step_id, user_id, fallback_type, reason, action_taken
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.NewString(), input.TurnID, input.LoopStepID, input.UserID, input.FallbackType, input.Reason, input.ActionTaken)
	return err
}
