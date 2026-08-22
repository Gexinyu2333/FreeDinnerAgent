package llm

import (
	"context"
	"fmt"

	"freedinner/backend/internal/agent"
	"freedinner/backend/internal/store"
)

type ToolPort interface {
	RouteAgentTools(ctx context.Context, input agent.ToolRouteInput) (agent.RouteResult, error)
	ExecuteAgentTool(ctx context.Context, input agent.ToolExecuteInput) (agent.ToolExecuteResult, error)
}

type loopInput struct {
	userID         string
	conversationID string
	userMessage    store.Message
	turn           store.AgentTurn
	cfg            store.AgentConfig
	provider       store.ModelProvider
	apiKey         string
	input          []ChatMessage
	tools          []agent.ToolDescriptor
}

func (s *Service) runLoop(ctx context.Context, in loopInput) (store.SendMessageResult, error) {
	maxSteps := agent.NormalizeMaxLoopSteps(in.cfg.MaxLoopSteps)
	retryLimit := agent.NormalizeRetryLimit(in.cfg.LLMRetryLimit)
	observations := make([]agent.Observation, 0)
	retryCount := 0
	fallbackProviderUsed := false

	for stepNo := 1; stepNo <= maxSteps; stepNo++ {
		stepType := "reason"
		actionType := "none"
		thoughtSummary := "请求模型输出下一步结构化 action。"
		step, err := s.harness.CreateLoopStep(ctx, store.AgentLoopStepCreate{
			TurnID:         in.turn.ID,
			UserID:         in.userID,
			ConversationID: in.conversationID,
			StepNo:         stepNo,
			StepType:       stepType,
			ThoughtSummary: &thoughtSummary,
			ActionType:     &actionType,
			Status:         "running",
		})
		if err != nil {
			return store.SendMessageResult{}, err
		}
		_, _ = s.harness.AddEvent(ctx, event(in.turn, "loop_step_started", map[string]any{
			"loop_step_id": step.ID,
			"step_no":      step.StepNo,
			"step_type":    step.StepType,
		}))

		modelInput := append([]ChatMessage{}, in.input...)
		if rendered := agent.RenderObservation(observations); rendered != "" {
			modelInput = append(modelInput, ChatMessage{Role: "system", Content: rendered})
		}

		response, err := s.openai.Generate(ctx, GenerateRequest{
			APIKey:               in.apiKey,
			BaseURL:              in.provider.ChatBaseURL,
			Model:                in.provider.DefaultChatModel,
			Input:                modelInput,
			Temperature:          in.cfg.Temperature,
			ThinkingEnabled:      in.cfg.ThinkingEnabled,
			ThinkingEffort:       in.cfg.ThinkingEffort,
			ThinkingBudgetTokens: in.cfg.ThinkingBudgetTokens,
		})
		if err != nil {
			if agent.IsRetryableLLMError(err) && retryCount < retryLimit {
				retryCount++
				_ = s.recordFallback(ctx, in, &step.ID, "retry_llm", err.Error(), fmt.Sprintf("retry %d/%d", retryCount, retryLimit))
				_, _ = s.harness.FinishLoopStep(ctx, step.ID, in.userID, in.conversationID, "skipped", nil, stringPtr(err.Error()))
				continue
			}
			if !fallbackProviderUsed {
				fallbackProvider, fallbackAPIKey, ok := s.findFallbackProvider(ctx, in.userID, in.provider.ID)
				if ok {
					fallbackProviderUsed = true
					_ = s.recordFallback(ctx, in, &step.ID, "provider_fallback", err.Error(), "switch provider to "+fallbackProvider.DisplayName)
					_, _ = s.harness.FinishLoopStep(ctx, step.ID, in.userID, in.conversationID, "skipped", nil, stringPtr(err.Error()))
					in.provider = fallbackProvider
					in.apiKey = fallbackAPIKey
					retryCount = 0
					continue
				}
			}
			return s.failLLMTurn(ctx, in, step.ID, err)
		}

		action, validation := agent.ValidateAction(response.Text, in.tools)
		_ = s.recordValidation(ctx, in, &step.ID, validation, stepNo)
		if !validation.Passed {
			_, _ = s.harness.AddEvent(ctx, event(in.turn, "llm_validation_failed", map[string]any{
				"loop_step_id": step.ID,
				"reason":       validation.Reason,
			}))
			if retryCount < retryLimit {
				retryCount++
				_ = s.recordFallback(ctx, in, &step.ID, "repair_output", validation.Reason, "ask model to emit valid action json")
				_, _ = s.harness.FinishLoopStep(ctx, step.ID, in.userID, in.conversationID, "skipped", nil, &validation.Reason)
				in.input = append(in.input, ChatMessage{Role: "system", Content: "上一条输出格式不合格：" + validation.Reason + "。请只输出符合协议的 JSON action。"})
				continue
			}
			return s.safeFinal(ctx, in, step.ID, agent.SafeFallbackAnswer, validation.Reason)
		}

		if validation.Repaired {
			_ = s.recordFallback(ctx, in, &step.ID, "repair_output", "extracted json action from model output", "used repaired JSON action")
		}

		switch action.Type {
		case agent.ActionFinalAnswer:
			contract := agent.ValidateFinalAnswerContract(action.Answer, observations)
			_ = s.recordValidation(ctx, in, &step.ID, contract, stepNo)
			if !contract.Passed {
				if retryCount < retryLimit {
					retryCount++
					_ = s.recordFallback(ctx, in, &step.ID, "answer_contract", contract.Reason, "ask model to avoid unverified success claim")
					_, _ = s.harness.FinishLoopStep(ctx, step.ID, in.userID, in.conversationID, "skipped", nil, &contract.Reason)
					in.input = append(in.input, ChatMessage{Role: "system", Content: "最终回复校验失败：" + contract.Reason + "。请基于 observation 如实说明已完成、未完成和需要用户处理的部分，只输出符合协议的 JSON action。"})
					continue
				}
				return s.safeFinal(ctx, in, step.ID, agent.SafeFallbackAnswer, contract.Reason)
			}
			return s.finishWithAnswer(ctx, in, step.ID, action.Answer, response)
		case agent.ActionAskUser:
			return s.finishWithAnswer(ctx, in, step.ID, action.Question, response)
		case agent.ActionMemorySearch:
			observation := s.observeMemory(ctx, in, action)
			observations = append(observations, observation)
			_, _ = s.harness.FinishLoopStep(ctx, step.ID, in.userID, in.conversationID, "success", &observation.Text, nil)
			_, _ = s.harness.AddEvent(ctx, event(in.turn, "loop_step_finished", map[string]any{
				"loop_step_id": step.ID,
				"status":       "success",
				"action_type":  agent.ActionMemorySearch,
			}))
		case agent.ActionToolCall:
			observation, waitingApproval := s.observeTool(ctx, in, step.ID, action)
			observations = append(observations, observation)
			if waitingApproval {
				return s.waitingApproval(ctx, in, step.ID, observation.Text)
			}
			status := "success"
			var errMessage *string
			if observation.Failed {
				status = "failed"
				errMessage = &observation.Text
			}
			_, _ = s.harness.FinishLoopStep(ctx, step.ID, in.userID, in.conversationID, status, &observation.Text, errMessage)
			_, _ = s.harness.AddEvent(ctx, event(in.turn, "loop_step_finished", map[string]any{
				"loop_step_id": step.ID,
				"status":       status,
				"action_type":  agent.ActionToolCall,
			}))
		}
	}

	return s.safeFinal(ctx, in, "", agent.MaxStepFallbackAnswer, "max loop steps reached")
}
