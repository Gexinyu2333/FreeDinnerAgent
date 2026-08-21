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

type AgentConfig struct {
	ID                    string          `json:"id"`
	UserID                string          `json:"user_id"`
	Name                  string          `json:"name"`
	SystemPrompt          string          `json:"system_prompt"`
	DefaultProviderID     *string         `json:"default_provider_id"`
	Temperature           float64         `json:"temperature"`
	MaxContextTokens      int             `json:"max_context_tokens"`
	MaxLoopSteps          int             `json:"max_loop_steps"`
	LLMRetryLimit         int             `json:"llm_retry_limit"`
	FallbackPolicy        json.RawMessage `json:"fallback_policy"`
	MemoryEnabled         bool            `json:"memory_enabled"`
	ToolUseEnabled        bool            `json:"tool_use_enabled"`
	DreamingEnabled       bool            `json:"dreaming_enabled"`
	SemanticMemoryEnabled bool            `json:"semantic_memory_enabled"`
	EmbeddingEnabled      bool            `json:"embedding_enabled"`
	EmbeddingCostPolicy   json.RawMessage `json:"embedding_cost_policy"`
	Metadata              json.RawMessage `json:"metadata"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

type AgentConfigUpdate struct {
	Name                  *string          `json:"name"`
	SystemPrompt          *string          `json:"system_prompt"`
	DefaultProviderID     **string         `json:"default_provider_id"`
	Temperature           *float64         `json:"temperature"`
	MaxContextTokens      *int             `json:"max_context_tokens"`
	MaxLoopSteps          *int             `json:"max_loop_steps"`
	LLMRetryLimit         *int             `json:"llm_retry_limit"`
	FallbackPolicy        *json.RawMessage `json:"fallback_policy"`
	MemoryEnabled         *bool            `json:"memory_enabled"`
	ToolUseEnabled        *bool            `json:"tool_use_enabled"`
	DreamingEnabled       *bool            `json:"dreaming_enabled"`
	SemanticMemoryEnabled *bool            `json:"semantic_memory_enabled"`
	EmbeddingEnabled      *bool            `json:"embedding_enabled"`
	EmbeddingCostPolicy   *json.RawMessage `json:"embedding_cost_policy"`
}

type AgentConfigStore struct {
	db *pgxpool.Pool
}

func NewAgentConfigStore(db *pgxpool.Pool) *AgentConfigStore {
	return &AgentConfigStore{db: db}
}

func (s *AgentConfigStore) GetDefault(ctx context.Context, userID string) (AgentConfig, error) {
	query := `
		SELECT id, user_id, name, system_prompt, default_provider_id, temperature, max_context_tokens,
		       max_loop_steps, llm_retry_limit, fallback_policy, memory_enabled, tool_use_enabled,
		       dreaming_enabled, semantic_memory_enabled, embedding_enabled, embedding_cost_policy,
		       metadata, created_at, updated_at
		FROM user_agent_configs
		WHERE user_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`
	cfg, err := scanAgentConfig(s.db.QueryRow(ctx, query, userID))
	if errors.Is(err, ErrNotFound) {
		if createErr := s.CreateDefault(ctx, uuid.NewString(), userID); createErr != nil {
			return AgentConfig{}, createErr
		}
		return scanAgentConfig(s.db.QueryRow(ctx, query, userID))
	}
	return cfg, err
}

func (s *AgentConfigStore) CreateDefault(ctx context.Context, id, userID string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO user_agent_configs (id, user_id) VALUES ($1, $2) ON CONFLICT (user_id, name) DO NOTHING`, id, userID)
	return err
}

func (s *AgentConfigStore) UpdateDefault(ctx context.Context, userID string, update AgentConfigUpdate) (AgentConfig, error) {
	cfg, err := s.GetDefault(ctx, userID)
	if err != nil {
		return AgentConfig{}, err
	}

	name := cfg.Name
	if update.Name != nil {
		name = *update.Name
	}
	systemPrompt := cfg.SystemPrompt
	if update.SystemPrompt != nil {
		systemPrompt = *update.SystemPrompt
	}
	defaultProviderID := cfg.DefaultProviderID
	if update.DefaultProviderID != nil {
		defaultProviderID = *update.DefaultProviderID
	}
	temperature := cfg.Temperature
	if update.Temperature != nil {
		temperature = *update.Temperature
	}
	maxContextTokens := cfg.MaxContextTokens
	if update.MaxContextTokens != nil {
		maxContextTokens = *update.MaxContextTokens
	}
	maxLoopSteps := cfg.MaxLoopSteps
	if update.MaxLoopSteps != nil {
		maxLoopSteps = *update.MaxLoopSteps
	}
	llmRetryLimit := cfg.LLMRetryLimit
	if update.LLMRetryLimit != nil {
		llmRetryLimit = *update.LLMRetryLimit
	}
	fallbackPolicy := cfg.FallbackPolicy
	if update.FallbackPolicy != nil {
		fallbackPolicy = *update.FallbackPolicy
	}
	memoryEnabled := cfg.MemoryEnabled
	if update.MemoryEnabled != nil {
		memoryEnabled = *update.MemoryEnabled
	}
	toolUseEnabled := cfg.ToolUseEnabled
	if update.ToolUseEnabled != nil {
		toolUseEnabled = *update.ToolUseEnabled
	}
	dreamingEnabled := cfg.DreamingEnabled
	if update.DreamingEnabled != nil {
		dreamingEnabled = *update.DreamingEnabled
	}
	semanticMemoryEnabled := cfg.SemanticMemoryEnabled
	if update.SemanticMemoryEnabled != nil {
		semanticMemoryEnabled = *update.SemanticMemoryEnabled
	}
	embeddingEnabled := cfg.EmbeddingEnabled
	if update.EmbeddingEnabled != nil {
		embeddingEnabled = *update.EmbeddingEnabled
	}
	embeddingCostPolicy := cfg.EmbeddingCostPolicy
	if update.EmbeddingCostPolicy != nil {
		embeddingCostPolicy = *update.EmbeddingCostPolicy
	}

	query := `
		UPDATE user_agent_configs
		SET name = $3,
		    system_prompt = $4,
		    default_provider_id = $5,
		    temperature = $6,
		    max_context_tokens = $7,
		    max_loop_steps = $8,
		    llm_retry_limit = $9,
		    fallback_policy = $10,
		    memory_enabled = $11,
		    tool_use_enabled = $12,
		    dreaming_enabled = $13,
		    semantic_memory_enabled = $14,
		    embedding_enabled = $15,
		    embedding_cost_policy = $16,
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, system_prompt, default_provider_id, temperature, max_context_tokens,
		          max_loop_steps, llm_retry_limit, fallback_policy, memory_enabled, tool_use_enabled,
		          dreaming_enabled, semantic_memory_enabled, embedding_enabled, embedding_cost_policy,
		          metadata, created_at, updated_at
	`
	return scanAgentConfig(s.db.QueryRow(ctx, query, cfg.ID, userID, name, systemPrompt, defaultProviderID, temperature,
		maxContextTokens, maxLoopSteps, llmRetryLimit, fallbackPolicy, memoryEnabled, toolUseEnabled,
		dreamingEnabled, semanticMemoryEnabled, embeddingEnabled, embeddingCostPolicy))
}

func scanAgentConfig(row pgx.Row) (AgentConfig, error) {
	var cfg AgentConfig
	if err := row.Scan(
		&cfg.ID,
		&cfg.UserID,
		&cfg.Name,
		&cfg.SystemPrompt,
		&cfg.DefaultProviderID,
		&cfg.Temperature,
		&cfg.MaxContextTokens,
		&cfg.MaxLoopSteps,
		&cfg.LLMRetryLimit,
		&cfg.FallbackPolicy,
		&cfg.MemoryEnabled,
		&cfg.ToolUseEnabled,
		&cfg.DreamingEnabled,
		&cfg.SemanticMemoryEnabled,
		&cfg.EmbeddingEnabled,
		&cfg.EmbeddingCostPolicy,
		&cfg.Metadata,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentConfig{}, ErrNotFound
		}
		return AgentConfig{}, err
	}
	return cfg, nil
}
