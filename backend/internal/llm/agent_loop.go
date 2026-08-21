package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"freedinner/backend/internal/agent"
	"freedinner/backend/internal/contextmgr"
	"freedinner/backend/internal/memory"
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

func (s *Service) observeMemory(ctx context.Context, in loopInput, action agent.Action) agent.Observation {
	if s.memories == nil || !in.cfg.MemoryEnabled {
		return agent.Observation{ActionType: agent.ActionMemorySearch, Text: "memory disabled", Failed: true}
	}
	result, err := s.memories.Retrieve(ctx, memory.RetrieveInput{
		UserID:           in.userID,
		ConversationID:   in.conversationID,
		MessageID:        &in.userMessage.ID,
		Query:            action.Query,
		MaxMemoryTokens:  900,
		LogRetrieval:     true,
		SemanticOnDemand: in.cfg.SemanticMemoryEnabled,
	})
	if err != nil {
		return agent.Observation{ActionType: agent.ActionMemorySearch, Text: err.Error(), Failed: true}
	}
	return agent.Observation{ActionType: agent.ActionMemorySearch, Text: renderMemoryObservation(result)}
}

func (s *Service) observeTool(ctx context.Context, in loopInput, stepID string, action agent.Action) (agent.Observation, bool) {
	if s.tools == nil || !in.cfg.ToolUseEnabled {
		return agent.Observation{ActionType: agent.ActionToolCall, Text: "tool use disabled", Failed: true}, false
	}
	_, _ = s.harness.AddEvent(ctx, event(in.turn, "tool_call_started", map[string]any{
		"loop_step_id": stepID,
		"tool_name":    action.ToolName,
	}))
	idempotencyKey := in.turn.ID + ":" + fmt.Sprint(stepID) + ":" + action.ToolName
	result, err := s.tools.ExecuteAgentTool(ctx, agent.ToolExecuteInput{
		UserID:         in.userID,
		ConversationID: in.conversationID,
		ToolName:       action.ToolName,
		Arguments:      action.Arguments,
		IdempotencyKey: &idempotencyKey,
	})
	if result.ToolCall.ID != "" {
		_ = s.harness.SetLoopStepActionRef(ctx, stepID, in.userID, in.conversationID, &result.ToolCall.ID)
	}
	if err != nil && result.ApprovalRequest != nil {
		_, _ = s.harness.AddEvent(ctx, event(in.turn, "approval_requested", map[string]any{
			"tool_name": action.ToolName,
			"call_id":   result.ToolCall.ID,
		}))
		return agent.Observation{ActionType: agent.ActionToolCall, Text: "工具 " + action.ToolName + " 需要用户审批。", RefID: &result.ToolCall.ID}, true
	}
	if err != nil {
		_, _ = s.harness.AddEvent(ctx, event(in.turn, "tool_call_finished", map[string]any{
			"tool_name": action.ToolName,
			"status":    "failed",
			"error":     err.Error(),
		}))
		return agent.Observation{ActionType: agent.ActionToolCall, Text: "工具 " + action.ToolName + " 执行失败：" + err.Error(), RefID: refOrNil(result.ToolCall.ID), Failed: true}, false
	}
	text := compactObservation(result.Result)
	_, _ = s.harness.AddEvent(ctx, event(in.turn, "tool_call_finished", map[string]any{
		"tool_name": action.ToolName,
		"status":    "success",
		"call_id":   result.ToolCall.ID,
	}))
	return agent.Observation{ActionType: agent.ActionToolCall, Text: "工具 " + action.ToolName + " 返回：" + text, RefID: &result.ToolCall.ID}, false
}

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

func (s *Service) curateEpisode(ctx context.Context, in loopInput, assistantMessage store.Message, status string) error {
	if s.memories == nil {
		return nil
	}
	summary := strings.TrimSpace(in.userMessage.Content)
	if len([]rune(summary)) > 120 {
		summary = string([]rune(summary)[:120]) + "..."
	}
	_, err := s.memories.UpsertWorkingMemory(ctx, memory.WorkingMemoryInput{
		UserID:         in.userID,
		ConversationID: in.conversationID,
		MemoryKey:      "last_episode",
		MemoryValue:    "用户：" + summary + "\n助手：" + trimRunes(assistantMessage.Content, 160),
		Category:       "temporary_context",
	})
	if err != nil {
		return err
	}
	taskType := inferTaskType(in.userMessage.Content)
	episode, err := s.memories.CreateEpisode(ctx, memory.EpisodeInput{
		UserID:             in.userID,
		ConversationID:     in.conversationID,
		UserMessageID:      &in.userMessage.ID,
		AssistantMessageID: &assistantMessage.ID,
		UserInput:          in.userMessage.Content,
		AgentSummary:       "用户请求：" + summary + "；助手已生成回复。",
		FinalResponse:      assistantMessage.Content,
		TaskType:           taskType,
		Status:             status,
		Tags:               episodeTags(in.userMessage.Content),
	})
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{
		"episode_id":      episode.ID,
		"conversation_id": in.conversationID,
		"task_type":       taskType,
	})
	_, _ = s.memories.EnqueueCuratorJob(ctx, in.userID, "episode_summary", payload)
	if in.cfg.DreamingEnabled && shouldTriggerDreaming(in.userMessage.Content) {
		_, _ = s.memories.EnqueueCuratorJob(ctx, in.userID, "dreaming", payload)
	}
	return nil
}

func (s *Service) buildContextMemory(ctx context.Context, userID, conversationID string, messageID *string, cfg store.AgentConfig, query string) []contextmgr.MemoryChunk {
	if s.memories == nil || !cfg.MemoryEnabled {
		return nil
	}
	result, err := s.memories.Retrieve(ctx, memory.RetrieveInput{
		UserID:           userID,
		ConversationID:   conversationID,
		MessageID:        messageID,
		Query:            query,
		MaxMemoryTokens:  1200,
		LogRetrieval:     true,
		SemanticOnDemand: cfg.SemanticMemoryEnabled,
	})
	if err != nil {
		return nil
	}
	chunks := make([]contextmgr.MemoryChunk, 0, len(result.Chunks))
	for _, chunk := range result.Chunks {
		chunks = append(chunks, contextmgr.MemoryChunk{
			Layer:      chunk.Layer,
			RefID:      chunk.RefID,
			Content:    chunk.Content,
			TokenCount: chunk.TokenCount,
			LoadMode:   chunk.LoadMode,
		})
	}
	return chunks
}

func renderMemoryObservation(result memory.RetrieveResult) string {
	if len(result.Chunks) == 0 {
		return "没有检索到相关记忆。"
	}
	parts := make([]string, 0, len(result.Chunks))
	for _, chunk := range result.Chunks {
		parts = append(parts, "["+chunk.Layer+"] "+trimRunes(chunk.Content, 180))
	}
	return strings.Join(parts, "\n")
}

func compactObservation(raw json.RawMessage) string {
	text := string(raw)
	if strings.TrimSpace(text) == "" {
		return "{}"
	}
	return trimRunes(text, 800)
}

func trimRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "..."
}

func inferTaskType(content string) *string {
	lowered := strings.ToLower(content)
	taskType := "chat"
	switch {
	case strings.Contains(lowered, "任务") || strings.Contains(lowered, "待办") || strings.Contains(lowered, "提醒"):
		taskType = "task"
	case strings.Contains(lowered, "总结") || strings.Contains(lowered, "摘要"):
		taskType = "summary"
	case strings.Contains(lowered, "搜索") || strings.Contains(lowered, "检索") || strings.Contains(lowered, "资料"):
		taskType = "search"
	case strings.Contains(lowered, "代码") || strings.Contains(lowered, "运行"):
		taskType = "workspace"
	}
	return &taskType
}

func episodeTags(content string) []string {
	tags := []string{}
	if taskType := inferTaskType(content); taskType != nil {
		tags = append(tags, *taskType)
	}
	if shouldTriggerDreaming(content) {
		tags = append(tags, "dreaming_candidate")
	}
	return tags
}

func shouldTriggerDreaming(content string) bool {
	lowered := strings.ToLower(content)
	keywords := []string{"反复", "每次", "以后", "流程", "习惯", "沉淀", "skill", "技能"}
	for _, keyword := range keywords {
		if strings.Contains(lowered, keyword) {
			return true
		}
	}
	return false
}

func refOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func optionalStepID(value string) *string {
	if value == "" {
		return nil
	}
	return &value
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
