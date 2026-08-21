package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"freedinner/backend/internal/secret"
	"freedinner/backend/internal/store"
)

var (
	ErrModelProviderRequired = errors.New("model provider required")
	ErrUnsupportedProvider   = errors.New("unsupported model provider")
	ErrLLMCallFailed         = errors.New("llm call failed")
)

type Service struct {
	conversations  *store.ConversationStore
	agentConfigs   *store.AgentConfigStore
	modelProviders *store.ModelProviderStore
	usage          *store.LLMUsageStore
	harness        *store.HarnessStore
	crypto         secret.Crypto
	openai         *OpenAIClient
}

func NewService(
	conversations *store.ConversationStore,
	agentConfigs *store.AgentConfigStore,
	modelProviders *store.ModelProviderStore,
	usage *store.LLMUsageStore,
	harness *store.HarnessStore,
	crypto secret.Crypto,
	openai *OpenAIClient,
) *Service {
	return &Service{
		conversations:  conversations,
		agentConfigs:   agentConfigs,
		modelProviders: modelProviders,
		usage:          usage,
		harness:        harness,
		crypto:         crypto,
		openai:         openai,
	}
}

func (s *Service) SendMessage(ctx context.Context, userID, conversationID, content string) (store.SendMessageResult, error) {
	cfg, err := s.agentConfigs.GetDefault(ctx, userID)
	if err != nil {
		return store.SendMessageResult{}, err
	}

	userMessage, err := s.conversations.CreateUserMessage(ctx, userID, conversationID, content)
	if err != nil {
		return store.SendMessageResult{}, err
	}

	agentConfigID := cfg.ID
	provider, err := s.modelProviders.FindDefault(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			turn, turnErr := s.createFailedTurn(ctx, userID, conversationID, &userMessage.ID, &agentConfigID, nil, "MODEL_PROVIDER_REQUIRED", "model provider required")
			if turnErr == nil {
				return store.SendMessageResult{TurnID: turn.ID, UserMessage: userMessage}, ErrModelProviderRequired
			}
			return store.SendMessageResult{}, ErrModelProviderRequired
		}
		return store.SendMessageResult{}, err
	}
	if provider.Provider != "openai" {
		turn, turnErr := s.createFailedTurn(ctx, userID, conversationID, &userMessage.ID, &agentConfigID, &provider.ID, "UNSUPPORTED_PROVIDER", "unsupported model provider")
		if turnErr == nil {
			return store.SendMessageResult{TurnID: turn.ID, UserMessage: userMessage}, ErrUnsupportedProvider
		}
		return store.SendMessageResult{}, ErrUnsupportedProvider
	}

	apiKey, err := s.crypto.Decrypt(provider.EncryptedChatAPIKey)
	if err != nil {
		return store.SendMessageResult{}, err
	}

	turn, err := s.harness.CreateTurn(ctx, store.AgentTurnCreate{
		UserID:         userID,
		ConversationID: conversationID,
		UserMessageID:  &userMessage.ID,
		AgentConfigID:  &agentConfigID,
		ProviderID:     &provider.ID,
	})
	if err != nil {
		return store.SendMessageResult{}, err
	}
	_, _ = s.harness.AddEvent(ctx, event(turn, "turn_started", map[string]any{
		"mode":     "minimal_llm",
		"provider": provider.Provider,
		"model":    provider.DefaultChatModel,
	}))
	if _, err := s.harness.StartTurn(ctx, turn.ID, userID, conversationID); err != nil {
		return store.SendMessageResult{}, err
	}

	messages, err := s.conversations.ListMessages(ctx, userID, conversationID)
	if err != nil {
		return store.SendMessageResult{}, err
	}

	input := buildOpenAIInput(cfg.SystemPrompt, messages)
	_, _ = s.harness.AddEvent(ctx, event(turn, "context_built", map[string]any{
		"recent_message_count": len(messages),
		"input_message_count":  len(input),
		"max_recent_messages":  20,
	}))
	thoughtSummary := "构建最小 LLM 上下文并请求模型生成最终回复。"
	actionType := "final_answer"
	loopStep, err := s.harness.CreateLoopStep(ctx, store.AgentLoopStepCreate{
		TurnID:         turn.ID,
		UserID:         userID,
		ConversationID: conversationID,
		StepNo:         1,
		StepType:       "reason",
		ThoughtSummary: &thoughtSummary,
		ActionType:     &actionType,
		Status:         "running",
	})
	if err != nil {
		return store.SendMessageResult{}, err
	}
	_, _ = s.harness.AddEvent(ctx, event(turn, "loop_step_started", map[string]any{
		"loop_step_id": loopStep.ID,
		"step_no":      loopStep.StepNo,
		"step_type":    loopStep.StepType,
		"action_type":  actionType,
	}))

	response, err := s.openai.Generate(ctx, GenerateRequest{
		APIKey:  apiKey,
		BaseURL: provider.ChatBaseURL,
		Model:   provider.DefaultChatModel,
		Input:   input,
	})
	if err != nil {
		errorMessage := err.Error()
		_ = s.usage.Create(ctx, store.LLMUsageCreate{
			UserID:         userID,
			ConversationID: &conversationID,
			MessageID:      &userMessage.ID,
			ProviderID:     &provider.ID,
			Provider:       provider.Provider,
			Model:          provider.DefaultChatModel,
			Status:         "failed",
			ErrorMessage:   &errorMessage,
		})
		_, _ = s.harness.FinishLoopStep(ctx, loopStep.ID, userID, conversationID, "failed", nil, &errorMessage)
		_, _ = s.harness.AddEvent(ctx, event(turn, "loop_step_finished", map[string]any{
			"loop_step_id": loopStep.ID,
			"status":       "failed",
			"error":        errorMessage,
		}))
		_, _ = s.harness.AddEvent(ctx, event(turn, "turn_failed", map[string]any{
			"code":  "LLM_CALL_FAILED",
			"error": errorMessage,
		}))
		_, _ = s.harness.FinishTurn(ctx, turn.ID, userID, conversationID, "failed", nil, &errorMessage)
		return store.SendMessageResult{TurnID: turn.ID, UserMessage: userMessage}, fmt.Errorf("%w: %v", ErrLLMCallFailed, err)
	}

	metadata, _ := json.Marshal(map[string]any{
		"source":   "openai",
		"model":    provider.DefaultChatModel,
		"provider": provider.Provider,
	})
	assistantMessage, err := s.conversations.CreateAssistantMessage(ctx, userID, conversationID, response.Text, metadata)
	if err != nil {
		return store.SendMessageResult{}, err
	}
	observation := "OpenAI 返回最终回复。"
	_, _ = s.harness.FinishLoopStep(ctx, loopStep.ID, userID, conversationID, "success", &observation, nil)
	_, _ = s.harness.AddEvent(ctx, event(turn, "loop_step_finished", map[string]any{
		"loop_step_id": loopStep.ID,
		"status":       "success",
	}))
	_, _ = s.harness.AddEvent(ctx, event(turn, "message_completed", map[string]any{
		"assistant_message_id": assistantMessage.ID,
		"provider":             provider.Provider,
		"model":                provider.DefaultChatModel,
	}))
	_, _ = s.harness.FinishTurn(ctx, turn.ID, userID, conversationID, "success", &assistantMessage.ID, nil)

	latency := response.LatencyMS
	_ = s.usage.Create(ctx, store.LLMUsageCreate{
		UserID:         userID,
		ConversationID: &conversationID,
		MessageID:      &assistantMessage.ID,
		ProviderID:     &provider.ID,
		Provider:       provider.Provider,
		Model:          provider.DefaultChatModel,
		InputTokens:    response.InputTokens,
		OutputTokens:   response.OutputTokens,
		TotalTokens:    response.TotalTokens,
		LatencyMS:      &latency,
		Status:         "success",
	})

	return store.SendMessageResult{
		TurnID:           turn.ID,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
	}, nil
}

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

func buildOpenAIInput(systemPrompt string, messages []store.Message) []ChatMessage {
	input := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	start := 0
	if len(messages) > 20 {
		start = len(messages) - 20
	}

	for _, msg := range messages[start:] {
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
			continue
		}
		input = append(input, ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return input
}
