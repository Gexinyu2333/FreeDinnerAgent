package contextmgr

import (
	"context"
	"encoding/json"
	"strings"

	"freedinner/backend/internal/store"
)

const (
	DefaultRecentTurnLimit          = 8
	DefaultCompressionThresholdRate = 0.85
	metaSystemPrompt                = "你是 FreeDinnerAgent 的上下文引擎。你必须优先遵守系统安全规则、用户当前输入和用户私有记忆；不要泄露密钥，不要跨用户读取数据。"
	safetyRules                     = "安全规则：不要输出 API Key、访问令牌或隐私凭据；公共知识不能覆盖用户私有偏好；工具调用必须遵守用户审批策略。"
)

type Store interface {
	ListActiveSummaries(ctx context.Context, userID, conversationID string) ([]store.ConversationSummary, error)
	CreateBuildLog(ctx context.Context, input store.ContextBuildLogCreate, items []store.ContextBuildItemCreate) (store.ContextBuildLog, error)
}

type Builder struct {
	store Store
}

type BuildInput struct {
	UserID          string
	ConversationID  string
	MessageID       *string
	AgentConfigID   *string
	ProviderID      *string
	SystemPrompt    string
	MaxTokens       int
	RecentTurnLimit int
	Messages        []store.Message
	MemoryChunks    []MemoryChunk
	SkillSections   []SkillSection
}

type MemoryChunk struct {
	Layer      string
	RefID      string
	Content    string
	TokenCount int
	LoadMode   string
}

type SkillSection struct {
	RefID      string
	Title      string
	Content    string
	TokenCount int
	LoadMode   string
}

type BuildResult struct {
	Input  []PromptMessage `json:"input"`
	Report HealthReport    `json:"report"`
	Log    *store.ContextBuildLog
}

type PromptMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type HealthReport struct {
	MaxContextTokens      int      `json:"max_context_tokens"`
	EstimatedPromptTokens int      `json:"estimated_prompt_tokens"`
	SystemTokens          int      `json:"system_tokens"`
	MemoryTokens          int      `json:"memory_tokens"`
	SkillTokens           int      `json:"skill_tokens"`
	SummaryTokens         int      `json:"summary_tokens"`
	RecentMessageTokens   int      `json:"recent_message_tokens"`
	CurrentInputTokens    int      `json:"current_input_tokens"`
	RecentTurnCount       int      `json:"recent_turn_count"`
	CompressedTurnCount   int      `json:"compressed_turn_count"`
	TruncatedItemCount    int      `json:"truncated_item_count"`
	CompressionStrategy   *string  `json:"compression_strategy,omitempty"`
	UsedSections          []string `json:"used_sections"`
}

func NewBuilder(store Store) *Builder {
	return &Builder{store: store}
}

func (b *Builder) Build(ctx context.Context, input BuildInput) (BuildResult, error) {
	input = normalizeBuildInput(input)
	summaries, err := b.loadSummaries(ctx, input.UserID, input.ConversationID)
	if err != nil {
		return BuildResult{}, err
	}

	selectedMessages, compressedCount := SelectRecentMessages(input.Messages, input.RecentTurnLimit)
	budget := BudgetFor(input.MaxTokens)
	memoryText, memoryTokens, truncatedMemory := renderMemory(input.MemoryChunks, budget.MemoryTokens)
	skillText, skillTokens, truncatedSkills := renderSkills(input.SkillSections, budget.SkillTokens)
	summaryText, summaryTokens := renderSummaries(summaries)

	systemText := strings.TrimSpace(metaSystemPrompt + "\n\n" + input.SystemPrompt + "\n\n" + safetyRules)
	systemTokens := EstimateTokens(systemText)
	inputMessages := []PromptMessage{{Role: "system", Content: systemText}}
	if memoryText != "" {
		inputMessages = append(inputMessages, PromptMessage{Role: "system", Content: memoryText})
	}
	if skillText != "" {
		inputMessages = append(inputMessages, PromptMessage{Role: "system", Content: skillText})
	}
	if summaryText != "" {
		inputMessages = append(inputMessages, PromptMessage{Role: "system", Content: summaryText})
	}

	recentTokens := 0
	currentInputTokens := 0
	for index, message := range selectedMessages {
		if !isPromptRole(message.Role) {
			continue
		}
		tokenCount := messageTokenCount(message)
		recentTokens += tokenCount
		if index == len(selectedMessages)-1 && message.Role == "user" {
			currentInputTokens = tokenCount
		}
		inputMessages = append(inputMessages, PromptMessage{Role: message.Role, Content: message.Content})
	}

	strategy := compressionStrategy(input.Messages, selectedMessages, input.MaxTokens, systemTokens+memoryTokens+skillTokens+summaryTokens+recentTokens)
	report := HealthReport{
		MaxContextTokens:      input.MaxTokens,
		EstimatedPromptTokens: systemTokens + memoryTokens + skillTokens + summaryTokens + recentTokens,
		SystemTokens:          systemTokens,
		MemoryTokens:          memoryTokens,
		SkillTokens:           skillTokens,
		SummaryTokens:         summaryTokens,
		RecentMessageTokens:   recentTokens,
		CurrentInputTokens:    currentInputTokens,
		RecentTurnCount:       countUserTurns(selectedMessages),
		CompressedTurnCount:   compressedCount,
		TruncatedItemCount:    truncatedMemory + truncatedSkills,
		CompressionStrategy:   strategy,
		UsedSections:          usedSections(memoryText, skillText, summaryText),
	}

	items := buildItems(input, summaries, selectedMessages, report, truncatedMemory)
	log, err := b.createLog(ctx, input, report, items)
	if err != nil {
		return BuildResult{}, err
	}
	return BuildResult{Input: inputMessages, Report: report, Log: &log}, nil
}

func (b *Builder) loadSummaries(ctx context.Context, userID, conversationID string) ([]store.ConversationSummary, error) {
	if b.store == nil {
		return nil, nil
	}
	return b.store.ListActiveSummaries(ctx, userID, conversationID)
}

func (b *Builder) createLog(ctx context.Context, input BuildInput, report HealthReport, items []store.ContextBuildItemCreate) (store.ContextBuildLog, error) {
	if b.store == nil {
		return store.ContextBuildLog{}, nil
	}
	metadata, _ := json.Marshal(map[string]any{"used_sections": report.UsedSections})
	return b.store.CreateBuildLog(ctx, store.ContextBuildLogCreate{
		UserID:                input.UserID,
		ConversationID:        input.ConversationID,
		MessageID:             input.MessageID,
		AgentConfigID:         input.AgentConfigID,
		ProviderID:            input.ProviderID,
		MaxContextTokens:      report.MaxContextTokens,
		EstimatedPromptTokens: report.EstimatedPromptTokens,
		SystemTokens:          report.SystemTokens,
		MemoryTokens:          report.MemoryTokens,
		SkillTokens:           report.SkillTokens,
		SummaryTokens:         report.SummaryTokens,
		RecentMessageTokens:   report.RecentMessageTokens,
		CurrentInputTokens:    report.CurrentInputTokens,
		RecentTurnCount:       report.RecentTurnCount,
		CompressedTurnCount:   report.CompressedTurnCount,
		TruncatedItemCount:    report.TruncatedItemCount,
		CompressionStrategy:   report.CompressionStrategy,
		Metadata:              metadata,
	}, items)
}
