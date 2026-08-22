package llm

import (
	"context"
	"strings"

	"freedinner/backend/internal/contextmgr"
	"freedinner/backend/internal/store"
)

func (s *Service) maybeAutoCompress(ctx context.Context, userID, conversationID string, messages []store.Message, report contextmgr.HealthReport, cfg store.AgentConfig, provider store.ModelProvider, apiKey string) {
	if s.compressor == nil || report.CompressionStrategy == nil {
		return
	}
	if len(messages) < contextmgr.DefaultRecentTurnLimit*2 {
		return
	}
	var summary *string
	if LLMFeatureEnabled(cfg, "auto_compression_llm") {
		featureProvider, featureAPIKey, featureTemperature, ok := s.resolveLLMFeatureProvider(ctx, userID, cfg, "auto_compression_llm", provider, apiKey)
		if ok {
			summary = s.summarizeCompressedMessages(ctx, messages, cfg, featureProvider, featureAPIKey, featureTemperature)
		}
	}
	_, _ = s.compressor.ManualCompress(ctx, contextmgr.ManualCompressInput{
		UserID:            userID,
		ConversationID:    conversationID,
		Messages:          messages,
		KeepRecentTurns:   contextmgr.DefaultRecentTurnLimit,
		TargetSummaryType: "turn_window",
		SummaryText:       summary,
	})
}

func (s *Service) summarizeCompressedMessages(ctx context.Context, messages []store.Message, cfg store.AgentConfig, provider store.ModelProvider, apiKey string, featureTemperature *float64) *string {
	if s.openai == nil || strings.TrimSpace(apiKey) == "" || provider.Provider != "openai" {
		return nil
	}
	kept, _ := contextmgr.SelectRecentMessages(messages, contextmgr.DefaultRecentTurnLimit)
	compressed := messages[:len(messages)-len(kept)]
	if len(compressed) == 0 {
		return nil
	}
	prompt := renderCompressionPrompt(compressed)
	temperature := 0.2
	if featureTemperature != nil {
		temperature = *featureTemperature
	}
	response, err := s.openai.Generate(ctx, GenerateRequest{
		APIKey:      apiKey,
		BaseURL:     provider.ChatBaseURL,
		Model:       provider.DefaultChatModel,
		Temperature: temperature,
		Input: []ChatMessage{
			{Role: "system", Content: "你是 FreeDinnerAgent 的上下文压缩器。请输出紧凑、忠实、结构化的中文摘要，必须保留用户明确偏好、约束、待办、工具结果和冲突/更正。不要添加不存在的信息。"},
			{Role: "user", Content: prompt},
		},
		ThinkingEnabled:      cfg.ThinkingEnabled,
		ThinkingEffort:       cfg.ThinkingEffort,
		ThinkingBudgetTokens: minInt(cfg.ThinkingBudgetTokens, 1024),
	})
	if err != nil || strings.TrimSpace(response.Text) == "" {
		return nil
	}
	summary := strings.TrimSpace(response.Text)
	return &summary
}

func renderCompressionPrompt(messages []store.Message) string {
	var builder strings.Builder
	builder.WriteString("请压缩以下早期对话，只保留后续回复必须知道的信息。\n\n")
	for _, message := range messages {
		if message.Role != "user" && message.Role != "assistant" && message.Role != "system" {
			continue
		}
		builder.WriteString(strings.ToUpper(message.Role))
		if message.IsAnchor && message.AnchorReason != nil {
			builder.WriteString(" [ANCHOR:")
			builder.WriteString(*message.AnchorReason)
			builder.WriteString("]")
		}
		builder.WriteString(": ")
		builder.WriteString(trimRunes(message.Content, 800))
		builder.WriteString("\n")
	}
	return builder.String()
}

func minInt(a, b int) int {
	if a <= 0 || a > b {
		return b
	}
	return a
}
