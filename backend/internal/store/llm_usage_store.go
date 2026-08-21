package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LLMUsageCreate struct {
	UserID         string
	ConversationID *string
	MessageID      *string
	ProviderID     *string
	Provider       string
	Model          string
	InputTokens    int
	OutputTokens   int
	TotalTokens    int
	LatencyMS      *int
	Status         string
	ErrorMessage   *string
}

type LLMUsageStore struct {
	db *pgxpool.Pool
}

func NewLLMUsageStore(db *pgxpool.Pool) *LLMUsageStore {
	return &LLMUsageStore{db: db}
}

func (s *LLMUsageStore) Create(ctx context.Context, input LLMUsageCreate) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO llm_usage_logs (
			id, user_id, conversation_id, message_id, provider_id, provider, model,
			input_tokens, output_tokens, total_tokens, latency_ms, status, error_message
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, uuid.NewString(), input.UserID, input.ConversationID, input.MessageID, input.ProviderID, input.Provider, input.Model,
		input.InputTokens, input.OutputTokens, input.TotalTokens, input.LatencyMS, input.Status, input.ErrorMessage)
	return err
}
