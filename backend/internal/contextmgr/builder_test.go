package contextmgr

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"freedinner/backend/internal/store"
)

type fakeContextStore struct {
	summaries []store.ConversationSummary
	logs      []store.ContextBuildLogCreate
	items     []store.ContextBuildItemCreate
}

func (f *fakeContextStore) ListActiveSummaries(ctx context.Context, userID, conversationID string) ([]store.ConversationSummary, error) {
	return f.summaries, nil
}

func (f *fakeContextStore) CreateBuildLog(ctx context.Context, input store.ContextBuildLogCreate, items []store.ContextBuildItemCreate) (store.ContextBuildLog, error) {
	f.logs = append(f.logs, input)
	f.items = append(f.items, items...)
	return store.ContextBuildLog{
		ID:                    "ctx-1",
		UserID:                input.UserID,
		ConversationID:        input.ConversationID,
		MaxContextTokens:      input.MaxContextTokens,
		EstimatedPromptTokens: input.EstimatedPromptTokens,
		Metadata:              json.RawMessage(`{}`),
		CreatedAt:             time.Now(),
	}, nil
}

func (f *fakeContextStore) CreateSummary(ctx context.Context, input store.ConversationSummaryCreate) (store.ConversationSummary, error) {
	return store.ConversationSummary{
		ID:                   "sum-1",
		ConversationID:       input.ConversationID,
		UserID:               input.UserID,
		SummaryType:          input.SummaryType,
		SourceMessageStartID: input.SourceMessageStartID,
		SourceMessageEndID:   input.SourceMessageEndID,
		SourceTurnCount:      input.SourceTurnCount,
		Summary:              input.Summary,
		TokenCount:           input.TokenCount,
		Status:               "active",
	}, nil
}

func (f *fakeContextStore) CreateCompletedCompressionJob(ctx context.Context, input store.ConversationCompressionJobCreate) (store.ConversationCompressionJob, error) {
	return store.ConversationCompressionJob{
		ID:                   "job-1",
		ConversationID:       input.ConversationID,
		UserID:               input.UserID,
		TriggerType:          input.TriggerType,
		KeepRecentTurns:      input.KeepRecentTurns,
		TargetSummaryType:    input.TargetSummaryType,
		Status:               "success",
		SummaryID:            input.SummaryID,
		OriginalTokenCount:   input.OriginalTokenCount,
		CompressedTokenCount: input.CompressedTokenCount,
	}, nil
}

func TestBudgetFor(t *testing.T) {
	budget := BudgetFor(10000)
	if budget.MemoryTokens != 3500 {
		t.Fatalf("expected memory budget 3500, got %d", budget.MemoryTokens)
	}
	if budget.RecentMessageTokens != 1500 {
		t.Fatalf("expected recent message budget 1500, got %d", budget.RecentMessageTokens)
	}
}

func TestSelectRecentMessagesKeepsLastNTurns(t *testing.T) {
	messages := []store.Message{
		msg("1", "user", "u1"), msg("2", "assistant", "a1"),
		msg("3", "user", "u2"), msg("4", "assistant", "a2"),
		msg("5", "user", "u3"), msg("6", "assistant", "a3"),
	}
	selected, compressed := SelectRecentMessages(messages, 2)
	if compressed != 1 {
		t.Fatalf("expected 1 compressed turn, got %d", compressed)
	}
	if len(selected) != 4 || selected[0].ID != "3" {
		t.Fatalf("unexpected selected messages: %#v", selected)
	}
}

func TestBuilderCreatesHealthReportAndItems(t *testing.T) {
	storeFake := &fakeContextStore{summaries: []store.ConversationSummary{
		{ID: "sum-1", SummaryType: "turn_window", Summary: "早期摘要", TokenCount: 10},
	}}
	builder := NewBuilder(storeFake)
	result, err := builder.Build(context.Background(), BuildInput{
		UserID:          "user-1",
		ConversationID:  "conv-1",
		SystemPrompt:    "你是个人助理。",
		MaxTokens:       2000,
		RecentTurnLimit: 1,
		Messages: []store.Message{
			msg("1", "user", "早期问题"),
			msg("2", "assistant", "早期回答"),
			msg("3", "user", "当前问题"),
		},
		MemoryChunks: []MemoryChunk{{Layer: "profile", RefID: "pm-1", Content: "用户喜欢简洁回答", TokenCount: 8}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Report.CompressedTurnCount != 1 {
		t.Fatalf("expected one compressed turn, got %d", result.Report.CompressedTurnCount)
	}
	if result.Report.SummaryTokens != 10 || result.Report.MemoryTokens != 8 {
		t.Fatalf("unexpected report: %#v", result.Report)
	}
	if len(storeFake.logs) != 1 || len(storeFake.items) == 0 {
		t.Fatalf("expected build log and items")
	}
	if len(result.Input) < 3 {
		t.Fatalf("expected system, memory/summary and recent messages, got %#v", result.Input)
	}
}

func TestManualCompressCreatesSummaryAndJob(t *testing.T) {
	storeFake := &fakeContextStore{}
	compressor := NewCompressor(storeFake)
	result, err := compressor.ManualCompress(context.Background(), ManualCompressInput{
		UserID:          "user-1",
		ConversationID:  "conv-1",
		KeepRecentTurns: 1,
		Messages: []store.Message{
			msg("1", "user", "早期问题"),
			msg("2", "assistant", "早期回答"),
			msg("3", "user", "当前问题"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.CompressedMessages) != 2 || len(result.KeptMessages) != 1 {
		t.Fatalf("unexpected split: compressed=%d kept=%d", len(result.CompressedMessages), len(result.KeptMessages))
	}
	if result.Job.TriggerType != "manual" || result.Summary.SourceTurnCount != 1 {
		t.Fatalf("unexpected compression result: %#v", result)
	}
}

func TestSummarizeMessagesIncludesStructuredSections(t *testing.T) {
	summary := SummarizeMessages([]store.Message{msg("1", "user", "请记住我喜欢表格")})
	if !strings.Contains(summary, "用户目标") || !strings.Contains(summary, "请记住我喜欢表格") {
		t.Fatalf("unexpected summary: %s", summary)
	}
}

func TestEstimateTokensCountsCJKMoreDenselyThanASCII(t *testing.T) {
	chinese := EstimateTokens("我喜欢简洁回答")
	english := EstimateTokens("short answer please")
	if chinese <= english/2 {
		t.Fatalf("expected CJK token estimate to stay dense, chinese=%d english=%d", chinese, english)
	}
	if EstimateTokens("   ") != 0 {
		t.Fatal("expected blank content to estimate to zero")
	}
}

func msg(id, role, content string) store.Message {
	return store.Message{ID: id, Role: role, Content: content, TokenCount: EstimateTokens(content)}
}
