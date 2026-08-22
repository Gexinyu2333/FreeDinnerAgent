package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

func scanAgentConfig(row pgx.Row) (AgentConfig, error) {
	var cfg AgentConfig
	if err := row.Scan(
		&cfg.ID,
		&cfg.UserID,
		&cfg.Name,
		&cfg.SystemPrompt,
		&cfg.DefaultProviderID,
		&cfg.Temperature,
		&cfg.ThinkingEnabled,
		&cfg.ThinkingEffort,
		&cfg.ThinkingBudgetTokens,
		&cfg.MaxContextTokens,
		&cfg.MaxLoopSteps,
		&cfg.LLMRetryLimit,
		&cfg.FallbackPolicy,
		&cfg.MemoryEnabled,
		&cfg.ToolUseEnabled,
		&cfg.ToolApprovalPolicy,
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

func scanLLMFeatureSetting(row pgx.Row) (LLMFeatureSetting, error) {
	var setting LLMFeatureSetting
	if err := row.Scan(
		&setting.ID,
		&setting.AgentConfigID,
		&setting.UserID,
		&setting.FeatureKey,
		&setting.Enabled,
		&setting.ProviderID,
		&setting.ModelOverride,
		&setting.Temperature,
		&setting.Metadata,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LLMFeatureSetting{}, ErrNotFound
		}
		return LLMFeatureSetting{}, err
	}
	return setting, nil
}
