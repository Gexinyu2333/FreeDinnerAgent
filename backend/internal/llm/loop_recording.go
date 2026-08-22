package llm

import (
	"context"

	"freedinner/backend/internal/agent"
	"freedinner/backend/internal/store"
)

func (s *Service) recordValidation(ctx context.Context, in loopInput, stepID *string, result agent.ValidationResult, attemptNo int) error {
	status := "passed"
	var reason *string
	var repaired *string
	if !result.Passed {
		status = "failed"
		reason = &result.Reason
	}
	if result.Repaired {
		status = "repaired"
		repaired = &result.RepairOutput
	}
	return s.harness.CreateValidation(ctx, store.LLMOutputValidationCreate{
		TurnID:         in.turn.ID,
		LoopStepID:     stepID,
		UserID:         in.userID,
		ValidationType: "json_schema",
		Status:         status,
		FailureReason:  reason,
		RepairedOutput: repaired,
		AttemptNo:      attemptNo,
	})
}

func (s *Service) recordFallback(ctx context.Context, in loopInput, stepID *string, fallbackType, reason, actionTaken string) error {
	_, _ = s.harness.AddEvent(ctx, event(in.turn, "fallback_triggered", map[string]any{
		"type":   fallbackType,
		"reason": reason,
		"action": actionTaken,
	}))
	return s.harness.CreateFallbackEvent(ctx, store.AgentFallbackEventCreate{
		TurnID:       in.turn.ID,
		LoopStepID:   stepID,
		UserID:       in.userID,
		FallbackType: fallbackType,
		Reason:       reason,
		ActionTaken:  actionTaken,
	})
}
