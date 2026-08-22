package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"freedinner/backend/internal/agent"
	"freedinner/backend/internal/store"
)

func (s *Service) finishWithAnswer(ctx context.Context, in loopInput, stepID string, answer string, response GenerateResponse) (store.SendMessageResult, error) {
	answer = strings.TrimSpace(answer)
	metadata, _ := json.Marshal(map[string]any{
		"source":   "agent_loop",
		"model":    in.provider.DefaultChatModel,
		"provider": in.provider.Provider,
	})
	assistantMessage, err := s.conversations.CreateAssistantMessage(ctx, in.userID, in.conversationID, answer, metadata)
	if err != nil {
		return store.SendMessageResult{}, err
	}
	observation := "生成最终回复。"
	_, _ = s.harness.FinishLoopStep(ctx, stepID, in.userID, in.conversationID, "success", &observation, nil)
	_, _ = s.harness.AddEvent(ctx, event(in.turn, "message_completed", map[string]any{
		"assistant_message_id": assistantMessage.ID,
		"provider":             in.provider.Provider,
		"model":                in.provider.DefaultChatModel,
	}))
	_, _ = s.harness.FinishTurn(ctx, in.turn.ID, in.userID, in.conversationID, "success", &assistantMessage.ID, nil)
	latency := response.LatencyMS
	_ = s.usage.Create(ctx, store.LLMUsageCreate{
		UserID:         in.userID,
		ConversationID: &in.conversationID,
		MessageID:      &assistantMessage.ID,
		ProviderID:     &in.provider.ID,
		Provider:       in.provider.Provider,
		Model:          in.provider.DefaultChatModel,
		InputTokens:    response.InputTokens,
		OutputTokens:   response.OutputTokens,
		TotalTokens:    response.TotalTokens,
		LatencyMS:      &latency,
		Status:         "success",
	})
	_ = s.curateEpisode(ctx, in, assistantMessage, "success")
	return store.SendMessageResult{TurnID: in.turn.ID, UserMessage: in.userMessage, AssistantMessage: assistantMessage}, nil
}

func (s *Service) waitingApproval(ctx context.Context, in loopInput, stepID, message string) (store.SendMessageResult, error) {
	assistantMessage, err := s.conversations.CreateAssistantMessage(ctx, in.userID, in.conversationID, message, json.RawMessage(`{"source":"agent_loop","status":"waiting_approval"}`))
	if err != nil {
		return store.SendMessageResult{}, err
	}
	_, _ = s.harness.FinishLoopStep(ctx, stepID, in.userID, in.conversationID, "success", &message, nil)
	_, _ = s.harness.SetTurnStatus(ctx, in.turn.ID, in.userID, in.conversationID, "waiting_approval", nil)
	return store.SendMessageResult{TurnID: in.turn.ID, UserMessage: in.userMessage, AssistantMessage: assistantMessage}, nil
}

func (s *Service) safeFinal(ctx context.Context, in loopInput, stepID, answer, reason string) (store.SendMessageResult, error) {
	_ = s.recordFallback(ctx, in, optionalStepID(stepID), "safe_final_answer", reason, "created conservative final answer")
	if stepID != "" {
		_, _ = s.harness.FinishLoopStep(ctx, stepID, in.userID, in.conversationID, "failed", &answer, &reason)
	}
	return s.finishWithAnswer(ctx, in, ensureFinalStep(ctx, s, in, stepID), answer, GenerateResponse{})
}

func (s *Service) failLLMTurn(ctx context.Context, in loopInput, stepID string, err error) (store.SendMessageResult, error) {
	errorMessage := err.Error()
	_ = s.usage.Create(ctx, store.LLMUsageCreate{
		UserID:         in.userID,
		ConversationID: &in.conversationID,
		MessageID:      &in.userMessage.ID,
		ProviderID:     &in.provider.ID,
		Provider:       in.provider.Provider,
		Model:          in.provider.DefaultChatModel,
		Status:         "failed",
		ErrorMessage:   &errorMessage,
	})
	_, _ = s.harness.FinishLoopStep(ctx, stepID, in.userID, in.conversationID, "failed", nil, &errorMessage)
	_, _ = s.harness.AddEvent(ctx, event(in.turn, "turn_failed", map[string]any{"code": "LLM_CALL_FAILED", "error": errorMessage}))
	_, _ = s.harness.FinishTurn(ctx, in.turn.ID, in.userID, in.conversationID, "failed", nil, &errorMessage)
	return store.SendMessageResult{TurnID: in.turn.ID, UserMessage: in.userMessage}, fmt.Errorf("%w: %v", ErrLLMCallFailed, err)
}

func ensureFinalStep(ctx context.Context, s *Service, in loopInput, existingStepID string) string {
	if existingStepID != "" {
		return existingStepID
	}
	thought := "达到最大步数或兜底条件，生成保守最终回复。"
	actionType := agent.ActionFinalAnswer
	step, err := s.harness.CreateLoopStep(ctx, store.AgentLoopStepCreate{
		TurnID:         in.turn.ID,
		UserID:         in.userID,
		ConversationID: in.conversationID,
		StepNo:         agent.NormalizeMaxLoopSteps(in.cfg.MaxLoopSteps) + 1,
		StepType:       "finalize",
		ThoughtSummary: &thought,
		ActionType:     &actionType,
		Status:         "running",
	})
	if err != nil {
		return existingStepID
	}
	return step.ID
}
