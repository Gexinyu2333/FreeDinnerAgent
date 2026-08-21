package llm

import (
	"context"
	"encoding/json"
	"errors"
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

	contextResult, err := s.buildContext(ctx, userID, conversationID, &userMessage.ID, &agentConfigID, &provider.ID, cfg, messages)
	if err != nil {
		return store.SendMessageResult{}, err
	}
	s.maybeAutoCompress(ctx, userID, conversationID, messages, contextResult.Report)
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
		"mode":     "channel_agent_loop",
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
	contextResult, err := s.buildContext(ctx, userID, conversationID, &message.ID, &agentConfigID, &provider.ID, cfg, messages)
	if err != nil {
		return store.SendMessageResult{}, err
	}
	s.maybeAutoCompress(ctx, userID, conversationID, messages, contextResult.Report)
	input := toChatMessages(contextResult.Input)
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

func (s *Service) buildContext(ctx context.Context, userID, conversationID string, messageID *string, agentConfigID *string, providerID *string, cfg store.AgentConfig, messages []store.Message) (contextmgr.BuildResult, error) {
	systemPrompt := s.resolveSystemPrompt(ctx, userID, cfg)
	if s.contexts == nil {
		return contextmgr.BuildResult{
			Input:  toPromptMessages(buildOpenAIInput(systemPrompt, messages)),
			Report: contextmgr.HealthReport{MaxContextTokens: cfg.MaxContextTokens},
		}, nil
	}
	return s.contexts.Build(ctx, contextmgr.BuildInput{
		UserID:          userID,
		ConversationID:  conversationID,
		MessageID:       messageID,
		AgentConfigID:   agentConfigID,
		ProviderID:      providerID,
		SystemPrompt:    systemPrompt,
		MaxTokens:       cfg.MaxContextTokens,
		RecentTurnLimit: contextmgr.DefaultRecentTurnLimit,
		Messages:        messages,
		MemoryChunks:    s.buildContextMemory(ctx, userID, conversationID, messageID, cfg, latestUserQuery(messages)),
		SkillSections:   s.buildContextSkills(ctx, userID, cfg, latestUserQuery(messages)),
	})
}

func (s *Service) resolveSystemPrompt(ctx context.Context, userID string, cfg store.AgentConfig) string {
	if s.market == nil {
		return cfg.SystemPrompt
	}
	version, err := s.market.ResolveAgentSystemPrompt(ctx, userID, cfg.ID)
	if err != nil || version == nil || version.Content == "" {
		return cfg.SystemPrompt
	}
	return version.Content
}

func toChatMessages(messages []contextmgr.PromptMessage) []ChatMessage {
	result := make([]ChatMessage, 0, len(messages))
	for _, message := range messages {
		result = append(result, ChatMessage{Role: message.Role, Content: message.Content})
	}
	return result
}

func toPromptMessages(messages []ChatMessage) []contextmgr.PromptMessage {
	result := make([]contextmgr.PromptMessage, 0, len(messages))
	for _, message := range messages {
		result = append(result, contextmgr.PromptMessage{Role: message.Role, Content: message.Content})
	}
	return result
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

func latestUserQuery(messages []store.Message) string {
	for index := len(messages) - 1; index >= 0; index-- {
		if messages[index].Role == "user" {
			return messages[index].Content
		}
	}
	return ""
}

func (s *Service) maybeAutoCompress(ctx context.Context, userID, conversationID string, messages []store.Message, report contextmgr.HealthReport) {
	if s.compressor == nil || report.CompressionStrategy == nil {
		return
	}
	if len(messages) < contextmgr.DefaultRecentTurnLimit*2 {
		return
	}
	_, _ = s.compressor.ManualCompress(ctx, contextmgr.ManualCompressInput{
		UserID:            userID,
		ConversationID:    conversationID,
		Messages:          messages,
		KeepRecentTurns:   contextmgr.DefaultRecentTurnLimit,
		TargetSummaryType: "turn_window",
	})
}

func (s *Service) buildContextSkills(ctx context.Context, userID string, cfg store.AgentConfig, query string) []contextmgr.SkillSection {
	if s.memories == nil || strings.TrimSpace(query) == "" {
		return nil
	}
	disclosures, err := s.memories.MatchSkills(ctx, userID, query, "light", 5)
	if err != nil {
		return nil
	}
	sections := make([]contextmgr.SkillSection, 0, len(disclosures))
	for _, disclosure := range disclosures {
		sections = append(sections, contextmgr.SkillSection{
			RefID:      disclosure.VersionID,
			Title:      disclosure.SkillName + " / " + disclosure.Title,
			Content:    disclosure.Content,
			TokenCount: disclosure.TokenCount,
			LoadMode:   disclosure.DisclosureLevel,
		})
	}
	return sections
}
