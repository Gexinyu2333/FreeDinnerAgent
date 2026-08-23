package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"freedinner/backend/internal/agent"
	"freedinner/backend/internal/contextmgr"
	memorysvc "freedinner/backend/internal/memory"
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
	contexts       *contextmgr.Builder
	compressor     *contextmgr.Compressor
	market         *store.MarketStore
	memories       *memorysvc.Manager
	tools          ToolPort
	crypto         secret.Crypto
	openai         *OpenAIClient
}

func NewService(
	conversations *store.ConversationStore,
	agentConfigs *store.AgentConfigStore,
	modelProviders *store.ModelProviderStore,
	usage *store.LLMUsageStore,
	harness *store.HarnessStore,
	contexts *contextmgr.Builder,
	compressor *contextmgr.Compressor,
	market *store.MarketStore,
	memories *memorysvc.Manager,
	tools ToolPort,
	crypto secret.Crypto,
	openai *OpenAIClient,
) *Service {
	return &Service{
		conversations:  conversations,
		agentConfigs:   agentConfigs,
		modelProviders: modelProviders,
		usage:          usage,
		harness:        harness,
		contexts:       contexts,
		compressor:     compressor,
		market:         market,
		memories:       memories,
		tools:          tools,
		crypto:         crypto,
		openai:         openai,
	}
}

func (s *Service) SendMessage(ctx context.Context, userID, conversationID, content string) (store.SendMessageResult, error) {
	if _, err := s.conversations.EnsureWebChat(ctx, userID, conversationID); err != nil {
		return store.SendMessageResult{}, err
	}

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
		"mode":           "minimal_llm",
		"trigger_source": "web_chat",
		"provider":       provider.Provider,
		"model":          provider.DefaultChatModel,
	}))
	if _, err := s.harness.StartTurn(ctx, turn.ID, userID, conversationID); err != nil {
		return store.SendMessageResult{}, err
	}

	messages, err := s.conversations.ListMessages(ctx, userID, conversationID)
	if err != nil {
		return store.SendMessageResult{}, err
	}

	contextResult, err := s.buildContext(ctx, userID, conversationID, &userMessage.ID, &agentConfigID, &provider.ID, cfg, messages)
	if err != nil {
		return store.SendMessageResult{}, err
	}
	s.maybeAutoCompress(ctx, userID, conversationID, messages, contextResult.Report, cfg, provider, apiKey)
	input := toChatMessages(contextResult.Input)
	route := agent.RouteResult{}
	if cfg.ToolUseEnabled && s.tools != nil {
		route, _ = s.tools.RouteAgentTools(ctx, agent.ToolRouteInput{
			UserID:         userID,
			ConversationID: conversationID,
			MessageID:      &userMessage.ID,
			Query:          content,
		})
		if len(route.Selected) > 0 {
			input = append(input, ChatMessage{Role: "system", Content: agent.RenderLoopInstructions(route.Selected, nil)})
		}
	} else {
		input = append(input, ChatMessage{Role: "system", Content: agent.RenderLoopInstructions(nil, nil)})
	}
	_, _ = s.harness.AddEvent(ctx, event(turn, "context_built", map[string]any{
		"report":              contextResult.Report,
		"input_message_count": len(input),
	}))
	result, err := s.runLoop(ctx, loopInput{
		userID:         userID,
		conversationID: conversationID,
		userMessage:    userMessage,
		turn:           turn,
		cfg:            cfg,
		provider:       provider,
		apiKey:         apiKey,
		input:          input,
		tools:          route.Selected,
	})
	if err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) RespondToExistingMessage(ctx context.Context, userID, conversationID string, message store.Message) (store.SendMessageResult, error) {
	cfg, err := s.agentConfigs.GetDefault(ctx, userID)
	if err != nil {
		return store.SendMessageResult{}, err
	}
	agentConfigID := cfg.ID
	provider, err := s.modelProviders.FindDefault(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			turn, turnErr := s.createFailedTurn(ctx, userID, conversationID, &message.ID, &agentConfigID, nil, "MODEL_PROVIDER_REQUIRED", "model provider required")
			if turnErr == nil {
				return store.SendMessageResult{TurnID: turn.ID, UserMessage: message}, ErrModelProviderRequired
			}
			return store.SendMessageResult{}, ErrModelProviderRequired
		}
		return store.SendMessageResult{}, err
	}
	if provider.Provider != "openai" {
		turn, turnErr := s.createFailedTurn(ctx, userID, conversationID, &message.ID, &agentConfigID, &provider.ID, "UNSUPPORTED_PROVIDER", "unsupported model provider")
		if turnErr == nil {
			return store.SendMessageResult{TurnID: turn.ID, UserMessage: message}, ErrUnsupportedProvider
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
		UserMessageID:  &message.ID,
		AgentConfigID:  &agentConfigID,
		ProviderID:     &provider.ID,
	})
	if err != nil {
		return store.SendMessageResult{}, err
	}
	_, _ = s.harness.AddEvent(ctx, event(turn, "turn_started", map[string]any{
		"mode":            "channel_agent_loop",
		"trigger_source":  "channel",
		"provider":        provider.Provider,
		"model":           provider.DefaultChatModel,
		"channel_context": channelContextMap(message),
	}))
	if _, err := s.harness.StartTurn(ctx, turn.ID, userID, conversationID); err != nil {
		return store.SendMessageResult{}, err
	}
	messages, err := s.conversations.ListMessages(ctx, userID, conversationID)
	if err != nil {
		return store.SendMessageResult{}, err
	}
	contextResult, err := s.buildContext(ctx, userID, conversationID, &message.ID, &agentConfigID, &provider.ID, cfg, messages)
	if err != nil {
		return store.SendMessageResult{}, err
	}
	s.maybeAutoCompress(ctx, userID, conversationID, messages, contextResult.Report, cfg, provider, apiKey)
	input := toChatMessages(contextResult.Input)
	if channelContext := renderChannelContext(message); channelContext != "" {
		input = append(input, ChatMessage{Role: "system", Content: channelContext})
	}
	route := agent.RouteResult{}
	if cfg.ToolUseEnabled && s.tools != nil {
		route, _ = s.tools.RouteAgentTools(ctx, agent.ToolRouteInput{
			UserID:         userID,
			ConversationID: conversationID,
			MessageID:      &message.ID,
			Query:          message.Content,
		})
	}
	input = append(input, ChatMessage{Role: "system", Content: agent.RenderLoopInstructions(route.Selected, nil)})
	_, _ = s.harness.AddEvent(ctx, event(turn, "context_built", map[string]any{
		"report":              contextResult.Report,
		"input_message_count": len(input),
	}))
	result, err := s.runLoop(ctx, loopInput{
		userID:         userID,
		conversationID: conversationID,
		userMessage:    message,
		turn:           turn,
		cfg:            cfg,
		provider:       provider,
		apiKey:         apiKey,
		input:          input,
		tools:          route.Selected,
	})
	if err != nil {
		return result, err
	}
	return result, nil
}

func channelContextMap(message store.Message) map[string]any {
	var metadata map[string]any
	if err := json.Unmarshal(message.Metadata, &metadata); err != nil {
		return map[string]any{}
	}
	keys := []string{
		"channel",
		"channel_connection_id",
		"external_conversation_id",
		"external_conversation_type",
		"external_conversation_title",
		"external_scope_id",
		"external_scope_type",
		"external_sender_id",
		"external_sender_name",
	}
	result := make(map[string]any, len(keys))
	for _, key := range keys {
		if value, ok := metadata[key]; ok && value != nil {
			result[key] = value
		}
	}
	return result
}

func renderChannelContext(message store.Message) string {
	metadata := channelContextMap(message)
	if len(metadata) == 0 {
		return ""
	}
	provider := stringValue(metadata["channel"])
	conversationType := stringValue(metadata["external_conversation_type"])
	conversationID := stringValue(metadata["external_conversation_id"])
	scopeID := stringValue(metadata["external_scope_id"])
	title := stringValue(metadata["external_conversation_title"])
	senderID := stringValue(metadata["external_sender_id"])
	senderName := stringValue(metadata["external_sender_name"])
	if provider == "" {
		provider = "external channel"
	}
	scopeLabel := conversationType
	if title != "" {
		scopeLabel = strings.TrimSpace(scopeLabel + " " + title)
	}
	lines := []string{
		"当前消息来自外部 Channel 监听入口，不是 Web Chat 主动对话。",
		fmt.Sprintf("Channel provider: %s", provider),
	}
	if scopeLabel != "" || conversationID != "" {
		lines = append(lines, fmt.Sprintf("External conversation: %s (%s)", scopeLabel, conversationID))
	}
	if scopeID != "" {
		lines = append(lines, fmt.Sprintf("External scope id: %s", scopeID))
	}
	if senderName != "" || senderID != "" {
		lines = append(lines, fmt.Sprintf("Sender: %s (%s)", senderName, senderID))
	}
	lines = append(lines, "回复将写入 Channel Outbox，并按该 Channel 策略审批或发送。")
	return strings.Join(lines, "\n")
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case *string:
		if typed == nil {
			return ""
		}
		return strings.TrimSpace(*typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
