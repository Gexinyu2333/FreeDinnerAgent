package llm

import (
	"testing"

	"freedinner/backend/internal/store"
)

func TestSkillLoadMode(t *testing.T) {
	if got := skillLoadMode("写周报"); got != "light" {
		t.Fatalf("expected light, got %q", got)
	}
	if got := skillLoadMode("给我一个详细流程"); got != "standard" {
		t.Fatalf("expected standard, got %q", got)
	}
	if got := skillLoadMode("完整展开所有细节"); got != "full" {
		t.Fatalf("expected full, got %q", got)
	}
}

func TestLLMFeatureEnabledDefaultsToFalse(t *testing.T) {
	if LLMFeatureEnabled(store.AgentConfig{}, "auto_compression_llm") {
		t.Fatal("expected missing policy to disable extra llm feature")
	}
	cfg := store.AgentConfig{LLMFeatureSettings: []store.LLMFeatureSetting{{FeatureKey: "auto_compression_llm", Enabled: true}}}
	if !LLMFeatureEnabled(cfg, "auto_compression_llm") {
		t.Fatal("expected configured feature to be enabled")
	}
	if LLMFeatureEnabled(cfg, "dreaming_llm") {
		t.Fatal("expected unrelated feature to stay disabled")
	}
}

func TestLLMFeatureSupportsProviderOverride(t *testing.T) {
	providerID := "provider-cheap"
	dreamProviderID := "provider-dream"
	modelOverride := "cheap-model"
	temperature := 0.1
	cfg := store.AgentConfig{LLMFeatureSettings: []store.LLMFeatureSetting{
		{
			FeatureKey:    "auto_compression_llm",
			Enabled:       true,
			ProviderID:    &providerID,
			ModelOverride: &modelOverride,
			Temperature:   &temperature,
		},
		{
			FeatureKey: "dreaming_llm",
			Enabled:    false,
			ProviderID: &dreamProviderID,
		},
	}}

	compression := LLMFeature(cfg, "auto_compression_llm")
	if !compression.Enabled {
		t.Fatal("expected compression feature to be enabled")
	}
	if compression.ProviderID != "provider-cheap" {
		t.Fatalf("expected provider override, got %q", compression.ProviderID)
	}
	if compression.ModelOverride != "cheap-model" {
		t.Fatalf("expected model override, got %q", compression.ModelOverride)
	}
	if compression.Temperature == nil || *compression.Temperature != 0.1 {
		t.Fatalf("expected temperature override, got %#v", compression.Temperature)
	}

	dreaming := LLMFeature(cfg, "dreaming_llm")
	if dreaming.Enabled {
		t.Fatal("expected dreaming feature to stay disabled")
	}
	if dreaming.ProviderID != "provider-dream" {
		t.Fatalf("expected disabled feature to still parse provider id for UI roundtrip, got %q", dreaming.ProviderID)
	}
}
