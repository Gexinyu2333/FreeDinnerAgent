package market

import (
	"errors"
	"testing"

	"freedinner/backend/internal/store"
)

func TestRenderPromptReplacesVariables(t *testing.T) {
	got := RenderPrompt("你是 {agent_name}，使用 {language} 回答。", map[string]string{
		"agent_name": "小饭",
		"language":   "中文",
	})
	expected := "你是 小饭，使用 中文 回答。"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestNormalizeLoadMode(t *testing.T) {
	if got := normalizeLoadMode("full"); got != "full" {
		t.Fatalf("expected full, got %q", got)
	}
	if got := normalizeLoadMode("bad"); got != "auto" {
		t.Fatalf("expected auto fallback, got %q", got)
	}
}

func TestCompactStrings(t *testing.T) {
	got := compactStrings([]string{" mcp ", "skill", "mcp", "", "  "})
	if len(got) != 2 || got[0] != "mcp" || got[1] != "skill" {
		t.Fatalf("unexpected compacted strings: %#v", got)
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Fatalf("expected empty token count 0, got %d", got)
	}
	if got := EstimateTokens("abcdef"); got != 3 {
		t.Fatalf("expected rough token count 3, got %d", got)
	}
}

func TestExtractPromptVariables(t *testing.T) {
	got := ExtractPromptVariables("你是 {agent_name}，用 {language} 回答。{agent_name}")
	if len(got) != 2 || got[0] != "agent_name" || got[1] != "language" {
		t.Fatalf("unexpected variables: %#v", got)
	}
}

func TestNormalizePromptVariablesAutoAddsPlaceholders(t *testing.T) {
	got, err := normalizePromptVariables("你是 {agent_name}", nil)
	if err != nil {
		t.Fatalf("normalize variables: %v", err)
	}
	if len(got) != 1 || got[0].Name != "agent_name" || got[0].ValueType != "string" {
		t.Fatalf("unexpected variables: %#v", got)
	}
}

func TestResolvePromptVariablesRequiresValue(t *testing.T) {
	_, err := ResolvePromptVariables([]store.SystemPromptVariable{{
		Name:      "tone",
		Required:  true,
		ValueType: "string",
	}}, map[string]string{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestResolvePromptVariablesEnum(t *testing.T) {
	metadata, err := variableMetadata([]string{"formal", "casual"})
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	resolved, err := ResolvePromptVariables([]store.SystemPromptVariable{{
		Name:      "tone",
		ValueType: "enum",
		Metadata:  metadata,
	}}, map[string]string{"tone": "formal"})
	if err != nil {
		t.Fatalf("resolve variables: %v", err)
	}
	if resolved["tone"] != "formal" {
		t.Fatalf("unexpected resolved value: %#v", resolved)
	}
	_, err = ResolvePromptVariables([]store.SystemPromptVariable{{
		Name:      "tone",
		ValueType: "enum",
		Metadata:  metadata,
	}}, map[string]string{"tone": "wild"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected enum validation error, got %v", err)
	}
}
