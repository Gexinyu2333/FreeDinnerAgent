package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversationSummary struct {
	ID                   string    `json:"id"`
	ConversationID       string    `json:"conversation_id"`
	UserID               string    `json:"user_id"`
	SummaryType          string    `json:"summary_type"`
	SourceMessageStartID *string   `json:"source_message_start_id"`
	SourceMessageEndID   *string   `json:"source_message_end_id"`
	SourceTurnCount      int       `json:"source_turn_count"`
	Summary              string    `json:"summary"`
	TokenCount           int       `json:"token_count"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type ConversationCompressionJob struct {
	ID                   string     `json:"id"`
	ConversationID       string     `json:"conversation_id"`
	UserID               string     `json:"user_id"`
	TriggerType          string     `json:"trigger_type"`
	SourceMessageStartID *string    `json:"source_message_start_id"`
	SourceMessageEndID   *string    `json:"source_message_end_id"`
	KeepRecentTurns      int        `json:"keep_recent_turns"`
	TargetSummaryType    string     `json:"target_summary_type"`
	Status               string     `json:"status"`
	SummaryID            *string    `json:"summary_id"`
	OriginalTokenCount   int        `json:"original_token_count"`
	CompressedTokenCount int        `json:"compressed_token_count"`
	ErrorMessage         *string    `json:"error_message"`
	RequestedAt          time.Time  `json:"requested_at"`
	StartedAt            *time.Time `json:"started_at"`
	FinishedAt           *time.Time `json:"finished_at"`
}

type ContextBuildLog struct {
	ID                    string          `json:"id"`
	UserID                string          `json:"user_id"`
	ConversationID        string          `json:"conversation_id"`
	MessageID             *string         `json:"message_id"`
	AgentConfigID         *string         `json:"agent_config_id"`
	ProviderID            *string         `json:"provider_id"`
	MaxContextTokens      int             `json:"max_context_tokens"`
	EstimatedPromptTokens int             `json:"estimated_prompt_tokens"`
	SystemTokens          int             `json:"system_tokens"`
	MemoryTokens          int             `json:"memory_tokens"`
	SkillTokens           int             `json:"skill_tokens"`
	ToolTokens            int             `json:"tool_tokens"`
	SummaryTokens         int             `json:"summary_tokens"`
	RecentMessageTokens   int             `json:"recent_message_tokens"`
	CurrentInputTokens    int             `json:"current_input_tokens"`
	RecentTurnCount       int             `json:"recent_turn_count"`
	CompressedTurnCount   int             `json:"compressed_turn_count"`
	TruncatedItemCount    int             `json:"truncated_item_count"`
	CompressionStrategy   *string         `json:"compression_strategy"`
	Metadata              json.RawMessage `json:"metadata"`
	CreatedAt             time.Time       `json:"created_at"`
}

type ContextBuildItem struct {
	ID             string    `json:"id"`
	ContextBuildID string    `json:"context_build_id"`
	ItemType       string    `json:"item_type"`
	RefID          *string   `json:"ref_id"`
	Title          *string   `json:"title"`
	TokenCount     int       `json:"token_count"`
	LoadMode       string    `json:"load_mode"`
	WasCompressed  bool      `json:"was_compressed"`
	WasTruncated   bool      `json:"was_truncated"`
	Priority       int       `json:"priority"`
	CreatedAt      time.Time `json:"created_at"`
}

type ConversationSummaryCreate struct {
	ConversationID       string
	UserID               string
	SummaryType          string
	SourceMessageStartID *string
	SourceMessageEndID   *string
	SourceTurnCount      int
	Summary              string
	TokenCount           int
}

type ConversationCompressionJobCreate struct {
	ConversationID       string
	UserID               string
	TriggerType          string
	SourceMessageStartID *string
	SourceMessageEndID   *string
	KeepRecentTurns      int
	TargetSummaryType    string
	OriginalTokenCount   int
	CompressedTokenCount int
	SummaryID            *string
}

type ContextBuildLogCreate struct {
	UserID                string
	ConversationID        string
	MessageID             *string
	AgentConfigID         *string
	ProviderID            *string
	MaxContextTokens      int
	EstimatedPromptTokens int
	SystemTokens          int
	MemoryTokens          int
	SkillTokens           int
	ToolTokens            int
	SummaryTokens         int
	RecentMessageTokens   int
	CurrentInputTokens    int
	RecentTurnCount       int
	CompressedTurnCount   int
	TruncatedItemCount    int
	CompressionStrategy   *string
	Metadata              json.RawMessage
}

type ContextBuildItemCreate struct {
	ContextBuildID string
	ItemType       string
	RefID          *string
	Title          *string
	TokenCount     int
	LoadMode       string
	WasCompressed  bool
	WasTruncated   bool
	Priority       int
}

type ContextStore struct {
	db *pgxpool.Pool
}

func NewContextStore(db *pgxpool.Pool) *ContextStore {
	return &ContextStore{db: db}
}

func (s *ContextStore) ListActiveSummaries(ctx context.Context, userID, conversationID string) ([]ConversationSummary, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, conversation_id, user_id, summary_type, source_message_start_id, source_message_end_id,
		       source_turn_count, summary, token_count, status, created_at, updated_at
		FROM conversation_summaries
		WHERE user_id = $1 AND conversation_id = $2 AND status = 'active'
		ORDER BY created_at ASC
	`, userID, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	summaries := make([]ConversationSummary, 0)
	for rows.Next() {
		summary, err := scanConversationSummary(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func (s *ContextStore) CreateSummary(ctx context.Context, input ConversationSummaryCreate) (ConversationSummary, error) {
	return scanConversationSummary(s.db.QueryRow(ctx, `
		INSERT INTO conversation_summaries (
			id, conversation_id, user_id, summary_type, source_message_start_id, source_message_end_id,
			source_turn_count, summary, token_count
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, conversation_id, user_id, summary_type, source_message_start_id, source_message_end_id,
		          source_turn_count, summary, token_count, status, created_at, updated_at
	`, uuid.NewString(), input.ConversationID, input.UserID, input.SummaryType, input.SourceMessageStartID,
		input.SourceMessageEndID, input.SourceTurnCount, input.Summary, input.TokenCount))
}

func (s *ContextStore) CreateCompletedCompressionJob(ctx context.Context, input ConversationCompressionJobCreate) (ConversationCompressionJob, error) {
	now := time.Now()
	if input.KeepRecentTurns <= 0 {
		input.KeepRecentTurns = 8
	}
	if input.TargetSummaryType == "" {
		input.TargetSummaryType = "turn_window"
	}
	return scanConversationCompressionJob(s.db.QueryRow(ctx, `
		INSERT INTO conversation_compression_jobs (
			id, conversation_id, user_id, trigger_type, source_message_start_id, source_message_end_id,
			keep_recent_turns, target_summary_type, status, summary_id, original_token_count,
			compressed_token_count, started_at, finished_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'success', $9, $10, $11, $12, $12)
		RETURNING id, conversation_id, user_id, trigger_type, source_message_start_id, source_message_end_id,
		          keep_recent_turns, target_summary_type, status, summary_id, original_token_count,
		          compressed_token_count, error_message, requested_at, started_at, finished_at
	`, uuid.NewString(), input.ConversationID, input.UserID, input.TriggerType, input.SourceMessageStartID,
		input.SourceMessageEndID, input.KeepRecentTurns, input.TargetSummaryType, input.SummaryID,
		input.OriginalTokenCount, input.CompressedTokenCount, now))
}

func (s *ContextStore) CreateBuildLog(ctx context.Context, input ContextBuildLogCreate, items []ContextBuildItemCreate) (ContextBuildLog, error) {
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return ContextBuildLog{}, err
	}
	defer tx.Rollback(ctx)

	log, err := scanContextBuildLog(tx.QueryRow(ctx, `
		INSERT INTO context_build_logs (
			id, user_id, conversation_id, message_id, agent_config_id, provider_id, max_context_tokens,
			estimated_prompt_tokens, system_tokens, memory_tokens, skill_tokens, tool_tokens, summary_tokens,
			recent_message_tokens, current_input_tokens, recent_turn_count, compressed_turn_count,
			truncated_item_count, compression_strategy, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id, user_id, conversation_id, message_id, agent_config_id, provider_id, max_context_tokens,
		          estimated_prompt_tokens, system_tokens, memory_tokens, skill_tokens, tool_tokens, summary_tokens,
		          recent_message_tokens, current_input_tokens, recent_turn_count, compressed_turn_count,
		          truncated_item_count, compression_strategy, metadata, created_at
	`, uuid.NewString(), input.UserID, input.ConversationID, input.MessageID, input.AgentConfigID, input.ProviderID,
		input.MaxContextTokens, input.EstimatedPromptTokens, input.SystemTokens, input.MemoryTokens,
		input.SkillTokens, input.ToolTokens, input.SummaryTokens, input.RecentMessageTokens,
		input.CurrentInputTokens, input.RecentTurnCount, input.CompressedTurnCount,
		input.TruncatedItemCount, input.CompressionStrategy, input.Metadata))
	if err != nil {
		return ContextBuildLog{}, err
	}

	for _, item := range items {
		if item.LoadMode == "" {
			item.LoadMode = "standard"
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO context_build_items (
				id, context_build_id, item_type, ref_id, title, token_count, load_mode,
				was_compressed, was_truncated, priority
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, uuid.NewString(), log.ID, item.ItemType, item.RefID, item.Title, item.TokenCount,
			item.LoadMode, item.WasCompressed, item.WasTruncated, item.Priority); err != nil {
			return ContextBuildLog{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ContextBuildLog{}, err
	}
	return log, nil
}

func scanConversationSummary(row pgx.Row) (ConversationSummary, error) {
	var summary ConversationSummary
	if err := row.Scan(&summary.ID, &summary.ConversationID, &summary.UserID, &summary.SummaryType,
		&summary.SourceMessageStartID, &summary.SourceMessageEndID, &summary.SourceTurnCount,
		&summary.Summary, &summary.TokenCount, &summary.Status, &summary.CreatedAt, &summary.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ConversationSummary{}, ErrNotFound
		}
		return ConversationSummary{}, err
	}
	return summary, nil
}

func scanConversationCompressionJob(row pgx.Row) (ConversationCompressionJob, error) {
	var job ConversationCompressionJob
	if err := row.Scan(&job.ID, &job.ConversationID, &job.UserID, &job.TriggerType,
		&job.SourceMessageStartID, &job.SourceMessageEndID, &job.KeepRecentTurns,
		&job.TargetSummaryType, &job.Status, &job.SummaryID, &job.OriginalTokenCount,
		&job.CompressedTokenCount, &job.ErrorMessage, &job.RequestedAt, &job.StartedAt,
		&job.FinishedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ConversationCompressionJob{}, ErrNotFound
		}
		return ConversationCompressionJob{}, err
	}
	return job, nil
}

func scanContextBuildLog(row pgx.Row) (ContextBuildLog, error) {
	var log ContextBuildLog
	if err := row.Scan(&log.ID, &log.UserID, &log.ConversationID, &log.MessageID,
		&log.AgentConfigID, &log.ProviderID, &log.MaxContextTokens, &log.EstimatedPromptTokens,
		&log.SystemTokens, &log.MemoryTokens, &log.SkillTokens, &log.ToolTokens,
		&log.SummaryTokens, &log.RecentMessageTokens, &log.CurrentInputTokens,
		&log.RecentTurnCount, &log.CompressedTurnCount, &log.TruncatedItemCount,
		&log.CompressionStrategy, &log.Metadata, &log.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ContextBuildLog{}, ErrNotFound
		}
		return ContextBuildLog{}, err
	}
	return log, nil
}
