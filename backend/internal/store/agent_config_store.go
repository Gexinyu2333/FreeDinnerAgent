package store

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentConfig struct {
	ID                    string              `json:"id"`
	UserID                string              `json:"user_id"`
	Name                  string              `json:"name"`
	SystemPrompt          string              `json:"system_prompt"`
	DefaultProviderID     *string             `json:"default_provider_id"`
	Temperature           float64             `json:"temperature"`
	ThinkingEnabled       bool                `json:"thinking_enabled"`
	ThinkingEffort        string              `json:"thinking_effort"`
	ThinkingBudgetTokens  int                 `json:"thinking_budget_tokens"`
	MaxContextTokens      int                 `json:"max_context_tokens"`
	MaxLoopSteps          int                 `json:"max_loop_steps"`
	LLMRetryLimit         int                 `json:"llm_retry_limit"`
	FallbackPolicy        json.RawMessage     `json:"fallback_policy"`
	MemoryEnabled         bool                `json:"memory_enabled"`
	ToolUseEnabled        bool                `json:"tool_use_enabled"`
	ToolApprovalPolicy    string              `json:"tool_approval_policy"`
	DreamingEnabled       bool                `json:"dreaming_enabled"`
	SemanticMemoryEnabled bool                `json:"semantic_memory_enabled"`
	EmbeddingEnabled      bool                `json:"embedding_enabled"`
	EmbeddingCostPolicy   json.RawMessage     `json:"embedding_cost_policy"`
	LLMFeatureSettings    []LLMFeatureSetting `json:"llm_feature_settings"`
	Metadata              json.RawMessage     `json:"metadata"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

type AgentConfigUpdate struct {
	Name                  *string                    `json:"name"`
	SystemPrompt          *string                    `json:"system_prompt"`
	DefaultProviderID     **string                   `json:"default_provider_id"`
	Temperature           *float64                   `json:"temperature"`
	ThinkingEnabled       *bool                      `json:"thinking_enabled"`
	ThinkingEffort        *string                    `json:"thinking_effort"`
	ThinkingBudgetTokens  *int                       `json:"thinking_budget_tokens"`
	MaxContextTokens      *int                       `json:"max_context_tokens"`
	MaxLoopSteps          *int                       `json:"max_loop_steps"`
	LLMRetryLimit         *int                       `json:"llm_retry_limit"`
	FallbackPolicy        *json.RawMessage           `json:"fallback_policy"`
	MemoryEnabled         *bool                      `json:"memory_enabled"`
	ToolUseEnabled        *bool                      `json:"tool_use_enabled"`
	ToolApprovalPolicy    *string                    `json:"tool_approval_policy"`
	DreamingEnabled       *bool                      `json:"dreaming_enabled"`
	SemanticMemoryEnabled *bool                      `json:"semantic_memory_enabled"`
	EmbeddingEnabled      *bool                      `json:"embedding_enabled"`
	EmbeddingCostPolicy   *json.RawMessage           `json:"embedding_cost_policy"`
	LLMFeatureSettings    *[]LLMFeatureSettingUpdate `json:"llm_feature_settings"`
}

type LLMFeatureSetting struct {
	ID            string          `json:"id"`
	AgentConfigID string          `json:"agent_config_id"`
	UserID        string          `json:"user_id"`
	FeatureKey    string          `json:"feature_key"`
	Enabled       bool            `json:"enabled"`
	ProviderID    *string         `json:"provider_id"`
	ModelOverride *string         `json:"model_override"`
	Temperature   *float64        `json:"temperature"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type LLMFeatureSettingUpdate struct {
	FeatureKey    string           `json:"feature_key"`
	Enabled       *bool            `json:"enabled"`
	ProviderID    **string         `json:"provider_id"`
	ModelOverride **string         `json:"model_override"`
	Temperature   **float64        `json:"temperature"`
	Metadata      *json.RawMessage `json:"metadata"`
}

type AgentConfigStore struct {
	db *pgxpool.Pool
}

func NewAgentConfigStore(db *pgxpool.Pool) *AgentConfigStore {
	return &AgentConfigStore{db: db}
}
