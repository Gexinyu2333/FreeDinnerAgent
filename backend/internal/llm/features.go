package llm

import (
	"context"
	"strings"

	"freedinner/backend/internal/store"
)

type llmFeatureConfig struct {
	Enabled       bool
	ProviderID    string
	ModelOverride string
	Temperature   *float64
}

func LLMFeatureEnabled(cfg store.AgentConfig, feature string) bool {
	return LLMFeature(cfg, feature).Enabled
}

func LLMFeature(cfg store.AgentConfig, feature string) llmFeatureConfig {
	for _, setting := range cfg.LLMFeatureSettings {
		if setting.FeatureKey != feature {
			continue
		}
		providerID := ""
		if setting.ProviderID != nil {
			providerID = strings.TrimSpace(*setting.ProviderID)
		}
		modelOverride := ""
		if setting.ModelOverride != nil {
			modelOverride = strings.TrimSpace(*setting.ModelOverride)
		}
		return llmFeatureConfig{
			Enabled:       setting.Enabled,
			ProviderID:    providerID,
			ModelOverride: modelOverride,
			Temperature:   setting.Temperature,
		}
	}
	return llmFeatureConfig{}
}

func (s *Service) resolveLLMFeatureProvider(ctx context.Context, userID string, cfg store.AgentConfig, feature string, fallbackProvider store.ModelProvider, fallbackAPIKey string) (store.ModelProvider, string, *float64, bool) {
	featureCfg := LLMFeature(cfg, feature)
	if !featureCfg.Enabled {
		return store.ModelProvider{}, "", nil, false
	}
	if featureCfg.ProviderID == "" || featureCfg.ProviderID == fallbackProvider.ID {
		if featureCfg.ModelOverride != "" {
			fallbackProvider.DefaultChatModel = featureCfg.ModelOverride
		}
		return fallbackProvider, fallbackAPIKey, featureCfg.Temperature, fallbackProvider.Provider == "openai" && strings.TrimSpace(fallbackAPIKey) != ""
	}
	if s.modelProviders == nil {
		return store.ModelProvider{}, "", nil, false
	}
	provider, err := s.modelProviders.FindByID(ctx, userID, featureCfg.ProviderID)
	if err != nil || provider.Status != "active" || provider.Provider != "openai" {
		return store.ModelProvider{}, "", nil, false
	}
	apiKey, err := s.crypto.Decrypt(provider.EncryptedChatAPIKey)
	if err != nil || strings.TrimSpace(apiKey) == "" {
		return store.ModelProvider{}, "", nil, false
	}
	if featureCfg.ModelOverride != "" {
		provider.DefaultChatModel = featureCfg.ModelOverride
	}
	return provider, apiKey, featureCfg.Temperature, true
}
