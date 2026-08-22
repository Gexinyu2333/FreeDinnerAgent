package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

func DefaultLLMFeatureKeys() []string {
	return []string{
		"auto_compression_llm",
		"dreaming_llm",
		"curator_llm",
		"shadow_validator_llm",
		"skill_router_llm",
	}
}

func validLLMFeatureKey(feature string) bool {
	for _, candidate := range DefaultLLMFeatureKeys() {
		if feature == candidate {
			return true
		}
	}
	return false
}

func (s *AgentConfigStore) EnsureDefaultLLMFeatureSettings(ctx context.Context, userID, agentConfigID string) error {
	for _, feature := range DefaultLLMFeatureKeys() {
		_, err := s.db.Exec(ctx, `
			INSERT INTO agent_llm_feature_settings (id, agent_config_id, user_id, feature_key)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (agent_config_id, feature_key) DO NOTHING
		`, uuid.NewString(), agentConfigID, userID, feature)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *AgentConfigStore) ListLLMFeatureSettings(ctx context.Context, userID, agentConfigID string) ([]LLMFeatureSetting, error) {
	if err := s.EnsureDefaultLLMFeatureSettings(ctx, userID, agentConfigID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, agent_config_id, user_id, feature_key, enabled, provider_id, model_override,
		       temperature, metadata, created_at, updated_at
		FROM agent_llm_feature_settings
		WHERE user_id = $1 AND agent_config_id = $2
		ORDER BY feature_key ASC
	`, userID, agentConfigID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	settings := make([]LLMFeatureSetting, 0)
	for rows.Next() {
		setting, err := scanLLMFeatureSetting(rows)
		if err != nil {
			return nil, err
		}
		settings = append(settings, setting)
	}
	return settings, rows.Err()
}

func (s *AgentConfigStore) ReplaceLLMFeatureSettings(ctx context.Context, userID, agentConfigID string, updates []LLMFeatureSettingUpdate) error {
	if err := s.EnsureDefaultLLMFeatureSettings(ctx, userID, agentConfigID); err != nil {
		return err
	}
	for _, update := range updates {
		if !validLLMFeatureKey(update.FeatureKey) {
			return errors.New("invalid llm feature key")
		}
		current, err := s.findLLMFeatureSetting(ctx, userID, agentConfigID, update.FeatureKey)
		if err != nil {
			return err
		}
		enabled := current.Enabled
		if update.Enabled != nil {
			enabled = *update.Enabled
		}
		providerID := current.ProviderID
		if update.ProviderID != nil {
			providerID = *update.ProviderID
		}
		modelOverride := current.ModelOverride
		if update.ModelOverride != nil {
			modelOverride = *update.ModelOverride
		}
		temperature := current.Temperature
		if update.Temperature != nil {
			temperature = *update.Temperature
		}
		metadata := current.Metadata
		if update.Metadata != nil {
			metadata = *update.Metadata
		}
		_, err = s.db.Exec(ctx, `
			UPDATE agent_llm_feature_settings
			SET enabled = $4,
			    provider_id = $5,
			    model_override = $6,
			    temperature = $7,
			    metadata = $8,
			    updated_at = NOW()
			WHERE user_id = $1 AND agent_config_id = $2 AND feature_key = $3
		`, userID, agentConfigID, update.FeatureKey, enabled, providerID, modelOverride, temperature, metadata)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *AgentConfigStore) findLLMFeatureSetting(ctx context.Context, userID, agentConfigID, featureKey string) (LLMFeatureSetting, error) {
	return scanLLMFeatureSetting(s.db.QueryRow(ctx, `
		SELECT id, agent_config_id, user_id, feature_key, enabled, provider_id, model_override,
		       temperature, metadata, created_at, updated_at
		FROM agent_llm_feature_settings
		WHERE user_id = $1 AND agent_config_id = $2 AND feature_key = $3
	`, userID, agentConfigID, featureKey))
}
