package contextmgr

import (
	"context"
	"encoding/json"
	"fmt"
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

type Budget struct {
	SystemTokens        int
	MemoryTokens        int
	SkillTokens         int
	ToolTokens          int
	SummaryTokens       int
	RecentMessageTokens int
}

func BudgetFor(maxTokens int) Budget {
	if maxTokens <= 0 {
		maxTokens = 12000
	}
	return Budget{
		SystemTokens:        percent(maxTokens, 10),
		MemoryTokens:        percent(maxTokens, 35),
		SkillTokens:         percent(maxTokens, 15),
		ToolTokens:          percent(maxTokens, 10),
		SummaryTokens:       percent(maxTokens, 10),
		RecentMessageTokens: percent(maxTokens, 15),
	}
}

func SelectRecentMessages(messages []store.Message, recentTurnLimit int) ([]store.Message, int) {
	if recentTurnLimit <= 0 {
		recentTurnLimit = DefaultRecentTurnLimit
	}
	userTurnIndexes := make([]int, 0)
	for index, message := range messages {
		if message.Role == "user" {
			userTurnIndexes = append(userTurnIndexes, index)
		}
	}
	if len(userTurnIndexes) <= recentTurnLimit {
		return messages, 0
	}
	start := userTurnIndexes[len(userTurnIndexes)-recentTurnLimit]
	return messages[start:], len(userTurnIndexes) - recentTurnLimit
}

func SummarizeMessages(messages []store.Message) string {
	var builder strings.Builder
	builder.WriteString("Conversation Summary\n")
	builder.WriteString("- 用户目标：\n")
	builder.WriteString("- 已确认约束：\n")
	builder.WriteString("- 关键事实：\n")
	builder.WriteString("- 已执行工具：\n")
	builder.WriteString("- 工具结果：\n")
	builder.WriteString("- 未完成事项：\n")
	builder.WriteString("- 冲突或待确认点：\n")
	for _, message := range messages {
		if !isPromptRole(message.Role) {
			continue
		}
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		if len([]rune(content)) > 180 {
			content = string([]rune(content)[:180]) + "..."
		}
		builder.WriteString(fmt.Sprintf("- %s: %s\n", message.Role, content))
	}
	return builder.String()
}

func EstimateTokens(content string) int {
	runes := len([]rune(content))
	if runes == 0 {
		return 0
	}
	return runes/3 + 1
}

func normalizeBuildInput(input BuildInput) BuildInput {
	if input.MaxTokens <= 0 {
		input.MaxTokens = 12000
	}
	if input.RecentTurnLimit <= 0 {
		input.RecentTurnLimit = DefaultRecentTurnLimit
	}
	return input
}

func renderMemory(chunks []MemoryChunk, maxTokens int) (string, int, int) {
	if len(chunks) == 0 {
		return "", 0, 0
	}
	var builder strings.Builder
	builder.WriteString("Retrieved Memory\n")
	total := 0
	truncated := 0
	for _, chunk := range chunks {
		tokenCount := chunk.TokenCount
		if tokenCount <= 0 {
			tokenCount = EstimateTokens(chunk.Content)
		}
		if total+tokenCount > maxTokens && total > 0 {
			truncated++
			continue
		}
		total += tokenCount
		builder.WriteString(fmt.Sprintf("- [%s] %s\n", chunk.Layer, strings.TrimSpace(chunk.Content)))
	}
	return strings.TrimSpace(builder.String()), total, truncated
}

func renderSkills(sections []SkillSection, maxTokens int) (string, int, int) {
	if len(sections) == 0 {
		return "", 0, 0
	}
	var builder strings.Builder
	builder.WriteString("Procedural Skills\n")
	total := 0
	truncated := 0
	for _, section := range sections {
		tokenCount := section.TokenCount
		if tokenCount <= 0 {
			tokenCount = EstimateTokens(section.Content)
		}
		if total+tokenCount > maxTokens && total > 0 {
			truncated++
			continue
		}
		total += tokenCount
		builder.WriteString(fmt.Sprintf("- %s: %s\n", strings.TrimSpace(section.Title), strings.TrimSpace(section.Content)))
	}
	return strings.TrimSpace(builder.String()), total, truncated
}

func EstimateSkillTokens(sections []SkillSection) int {
	total := 0
	for _, section := range sections {
		if section.TokenCount > 0 {
			total += section.TokenCount
		} else {
			total += EstimateTokens(section.Content)
		}
	}
	return total
}

func renderSummaries(summaries []store.ConversationSummary) (string, int) {
	if len(summaries) == 0 {
		return "", 0
	}
	var builder strings.Builder
	builder.WriteString("Compressed Conversation Summary\n")
	total := 0
	for _, summary := range summaries {
		builder.WriteString(summary.Summary)
		builder.WriteString("\n")
		total += summary.TokenCount
	}
	return strings.TrimSpace(builder.String()), total
}

func compressionStrategy(all []store.Message, selected []store.Message, maxTokens int, estimated int) *string {
	var strategies []string
	if len(selected) < len(all) {
		strategies = append(strategies, "recent_turn_limit")
	}
	if estimated > int(float64(maxTokens)*DefaultCompressionThresholdRate) {
		strategies = append(strategies, "token_threshold")
	}
	if len(strategies) == 0 {
		return nil
	}
	strategy := strings.Join(strategies, "+")
	return &strategy
}

func buildItems(input BuildInput, summaries []store.ConversationSummary, messages []store.Message, report HealthReport, truncatedMemory int) []store.ContextBuildItemCreate {
	items := []store.ContextBuildItemCreate{
		{ItemType: "system", Title: stringPtr("Meta System Prompt + Safety"), TokenCount: report.SystemTokens, Priority: 100},
	}
	for _, chunk := range input.MemoryChunks {
		itemType := memoryItemType(chunk.Layer)
		refID := refIDPtr(chunk.RefID)
		title := stringPtr(chunk.Layer)
		items = append(items, store.ContextBuildItemCreate{ItemType: itemType, RefID: refID, Title: title, TokenCount: chunk.TokenCount, LoadMode: defaultLoadMode(chunk.LoadMode), WasTruncated: truncatedMemory > 0, Priority: 80})
	}
	for _, section := range input.SkillSections {
		refID := refIDPtr(section.RefID)
		title := stringPtr(section.Title)
		items = append(items, store.ContextBuildItemCreate{ItemType: "procedural_skill", RefID: refID, Title: title, TokenCount: section.TokenCount, LoadMode: defaultLoadMode(section.LoadMode), Priority: 75})
	}
	for _, summary := range summaries {
		refID := summary.ID
		items = append(items, store.ContextBuildItemCreate{ItemType: "summary", RefID: &refID, Title: stringPtr(summary.SummaryType), TokenCount: summary.TokenCount, LoadMode: "summary", WasCompressed: true, Priority: 60})
	}
	for index, message := range messages {
		refID := message.ID
		itemType := "recent_message"
		if index == len(messages)-1 && message.Role == "user" {
			itemType = "current_input"
		}
		items = append(items, store.ContextBuildItemCreate{ItemType: itemType, RefID: &refID, Title: stringPtr(message.Role), TokenCount: messageTokenCount(message), Priority: 90})
	}
	return items
}

func memoryItemType(layer string) string {
	switch layer {
	case "working":
		return "working_memory"
	case "profile":
		return "profile_memory"
	case "semantic":
		return "semantic_memory"
	case "episodic":
		return "episodic_memory"
	case "procedural":
		return "procedural_skill"
	default:
		return "profile_memory"
	}
}

func usedSections(memoryText, skillText, summaryText string) []string {
	sections := []string{"system", "recent_messages", "current_input"}
	if memoryText != "" {
		sections = append(sections, "memory")
	}
	if skillText != "" {
		sections = append(sections, "skills")
	}
	if summaryText != "" {
		sections = append(sections, "summary")
	}
	return sections
}

func messageTokenCount(message store.Message) int {
	if message.TokenCount > 0 {
		return message.TokenCount
	}
	return EstimateTokens(message.Content)
}

func countUserTurns(messages []store.Message) int {
	count := 0
	for _, message := range messages {
		if message.Role == "user" {
			count++
		}
	}
	return count
}

func isPromptRole(role string) bool {
	return role == "user" || role == "assistant" || role == "system"
}

func percent(value int, ratio int) int {
	return value * ratio / 100
}

func refIDPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func defaultLoadMode(value string) string {
	if value == "" {
		return "standard"
	}
	return value
}
