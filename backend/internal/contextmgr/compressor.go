package contextmgr

import (
	"context"

	"freedinner/backend/internal/store"
)

type CompressionStore interface {
	CreateSummary(ctx context.Context, input store.ConversationSummaryCreate) (store.ConversationSummary, error)
	CreateCompletedCompressionJob(ctx context.Context, input store.ConversationCompressionJobCreate) (store.ConversationCompressionJob, error)
}

type Compressor struct {
	store CompressionStore
}

type ManualCompressInput struct {
	UserID            string
	ConversationID    string
	Messages          []store.Message
	KeepRecentTurns   int
	TargetSummaryType string
	SummaryText       *string
}

type ManualCompressResult struct {
	Summary            store.ConversationSummary        `json:"summary"`
	Job                store.ConversationCompressionJob `json:"job"`
	CompressedMessages []store.Message                  `json:"compressed_messages"`
	KeptMessages       []store.Message                  `json:"kept_messages"`
}

func NewCompressor(store CompressionStore) *Compressor {
	return &Compressor{store: store}
}

func (c *Compressor) ManualCompress(ctx context.Context, input ManualCompressInput) (ManualCompressResult, error) {
	if input.KeepRecentTurns <= 0 {
		input.KeepRecentTurns = DefaultRecentTurnLimit
	}
	if input.TargetSummaryType == "" {
		input.TargetSummaryType = "turn_window"
	}
	kept, _ := SelectRecentMessages(input.Messages, input.KeepRecentTurns)
	compressed := input.Messages[:len(input.Messages)-len(kept)]
	if len(compressed) == 0 {
		compressed = nil
	}

	summaryText := SummarizeMessages(compressed)
	if input.SummaryText != nil && *input.SummaryText != "" {
		summaryText = *input.SummaryText
	}
	originalTokens := messagesTokenCount(compressed)
	compressedTokens := EstimateTokens(summaryText)
	var startID, endID *string
	if len(compressed) > 0 {
		startID = &compressed[0].ID
		endID = &compressed[len(compressed)-1].ID
	}

	summary, err := c.store.CreateSummary(ctx, store.ConversationSummaryCreate{
		ConversationID:       input.ConversationID,
		UserID:               input.UserID,
		SummaryType:          input.TargetSummaryType,
		SourceMessageStartID: startID,
		SourceMessageEndID:   endID,
		SourceTurnCount:      countUserTurns(compressed),
		Summary:              summaryText,
		TokenCount:           compressedTokens,
	})
	if err != nil {
		return ManualCompressResult{}, err
	}
	summaryID := summary.ID
	job, err := c.store.CreateCompletedCompressionJob(ctx, store.ConversationCompressionJobCreate{
		ConversationID:       input.ConversationID,
		UserID:               input.UserID,
		TriggerType:          "manual",
		SourceMessageStartID: startID,
		SourceMessageEndID:   endID,
		KeepRecentTurns:      input.KeepRecentTurns,
		TargetSummaryType:    input.TargetSummaryType,
		OriginalTokenCount:   originalTokens,
		CompressedTokenCount: compressedTokens,
		SummaryID:            &summaryID,
	})
	if err != nil {
		return ManualCompressResult{}, err
	}
	return ManualCompressResult{Summary: summary, Job: job, CompressedMessages: compressed, KeptMessages: kept}, nil
}

func messagesTokenCount(messages []store.Message) int {
	total := 0
	for _, message := range messages {
		total += messageTokenCount(message)
	}
	return total
}
