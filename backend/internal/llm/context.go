package llm

import (
	"context"
	"strings"

	"freedinner/backend/internal/contextmgr"
	"freedinner/backend/internal/store"
)

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

func (s *Service) buildContextSkills(ctx context.Context, userID string, cfg store.AgentConfig, query string) []contextmgr.SkillSection {
	if s.memories == nil || strings.TrimSpace(query) == "" {
		return nil
	}
	loadMode := skillLoadMode(query)
	disclosures, err := s.memories.MatchSkills(ctx, userID, query, loadMode, 5)
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

func skillLoadMode(query string) string {
	lowered := strings.ToLower(strings.TrimSpace(query))
	fullKeywords := []string{"完整", "全部", "细节", "full", "详细展开"}
	for _, keyword := range fullKeywords {
		if strings.Contains(lowered, keyword) {
			return "full"
		}
	}
	standardKeywords := []string{"流程", "步骤", "怎么做", "如何", "详细", "方案", "standard", "计划"}
	for _, keyword := range standardKeywords {
		if strings.Contains(lowered, keyword) {
			return "standard"
		}
	}
	return "light"
}
