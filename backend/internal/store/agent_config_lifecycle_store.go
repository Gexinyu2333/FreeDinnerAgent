package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

func (s *AgentConfigStore) GetDefault(ctx context.Context, userID string) (AgentConfig, error) {
	query := `
		SELECT id, user_id, name, system_prompt, default_provider_id, temperature, thinking_enabled,
		       thinking_effort, thinking_budget_tokens, max_context_tokens,
		       max_loop_steps, llm_retry_limit, fallback_policy, memory_enabled, tool_use_enabled,
		       tool_approval_policy, dreaming_enabled, semantic_memory_enabled, embedding_enabled, embedding_cost_policy,
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
		cfg, err = scanAgentConfig(s.db.QueryRow(ctx, query, userID))
	}
	if err != nil {
		return AgentConfig{}, err
	}
	settings, err := s.ListLLMFeatureSettings(ctx, userID, cfg.ID)
	if err != nil {
		return AgentConfig{}, err
	}
	cfg.LLMFeatureSettings = settings
	return cfg, nil
}

func (s *AgentConfigStore) CreateDefault(ctx context.Context, id, userID string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO user_agent_configs (id, user_id) VALUES ($1, $2) ON CONFLICT (user_id, name) DO NOTHING`, id, userID)
	if err != nil {
		return err
	}
	return s.EnsureDefaultLLMFeatureSettings(ctx, userID, id)
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
	thinkingEnabled := cfg.ThinkingEnabled
	if update.ThinkingEnabled != nil {
		thinkingEnabled = *update.ThinkingEnabled
	}
	thinkingEffort := cfg.ThinkingEffort
	if update.ThinkingEffort != nil {
		thinkingEffort = normalizeThinkingEffort(*update.ThinkingEffort)
	}
	thinkingBudgetTokens := cfg.ThinkingBudgetTokens
	if update.ThinkingBudgetTokens != nil {
		thinkingBudgetTokens = *update.ThinkingBudgetTokens
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
	toolApprovalPolicy := cfg.ToolApprovalPolicy
	if update.ToolApprovalPolicy != nil {
		toolApprovalPolicy = normalizeToolApprovalPolicy(*update.ToolApprovalPolicy)
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
		    thinking_enabled = $7,
		    thinking_effort = $8,
		    thinking_budget_tokens = $9,
		    max_context_tokens = $10,
		    max_loop_steps = $11,
		    llm_retry_limit = $12,
		    fallback_policy = $13,
		    memory_enabled = $14,
		    tool_use_enabled = $15,
		    tool_approval_policy = $16,
		    dreaming_enabled = $17,
		    semantic_memory_enabled = $18,
		    embedding_enabled = $19,
		    embedding_cost_policy = $20,
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, system_prompt, default_provider_id, temperature, thinking_enabled,
		          thinking_effort, thinking_budget_tokens, max_context_tokens,
		          max_loop_steps, llm_retry_limit, fallback_policy, memory_enabled, tool_use_enabled,
		          tool_approval_policy, dreaming_enabled, semantic_memory_enabled, embedding_enabled, embedding_cost_policy,
		          metadata, created_at, updated_at
	`
	updated, err := scanAgentConfig(s.db.QueryRow(ctx, query, cfg.ID, userID, name, systemPrompt, defaultProviderID, temperature,
		thinkingEnabled, thinkingEffort, thinkingBudgetTokens, maxContextTokens, maxLoopSteps, llmRetryLimit, fallbackPolicy, memoryEnabled, toolUseEnabled,
		toolApprovalPolicy, dreamingEnabled, semanticMemoryEnabled, embeddingEnabled, embeddingCostPolicy))
	if err != nil {
		return AgentConfig{}, err
	}
	if update.LLMFeatureSettings != nil {
		if err := s.ReplaceLLMFeatureSettings(ctx, userID, cfg.ID, *update.LLMFeatureSettings); err != nil {
			return AgentConfig{}, err
		}
	}
	settings, err := s.ListLLMFeatureSettings(ctx, userID, cfg.ID)
	if err != nil {
		return AgentConfig{}, err
	}
	updated.LLMFeatureSettings = settings
	return updated, nil
}
