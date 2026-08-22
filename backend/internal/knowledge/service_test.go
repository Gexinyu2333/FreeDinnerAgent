package knowledge

import "testing"

func TestSplitTextTrimsAndChunksByRune(t *testing.T) {
	chunks := splitText("  你好世界abc  ", 3)
	want := []string{"你好世", "界ab", "c"}
	if len(chunks) != len(want) {
		t.Fatalf("len(chunks) = %d, want %d: %#v", len(chunks), len(want), chunks)
	}
	for i := range want {
		if chunks[i] != want[i] {
			t.Fatalf("chunks[%d] = %q, want %q", i, chunks[i], want[i])
		}
	}
}

func TestSplitTextEmptyContent(t *testing.T) {
	if chunks := splitText("   ", 10); chunks != nil {
		t.Fatalf("blank content should return nil, got %#v", chunks)
	}
}

func TestKnowledgeHelperDefaults(t *testing.T) {
	if estimateTokens("") != 0 {
		t.Fatal("empty content should estimate to 0 tokens")
	}
	if estimateTokens("abcd") != 2 {
		t.Fatalf("estimateTokens(abcd) = %d", estimateTokens("abcd"))
	}
	if normalizeVisibility("public") != "public" {
		t.Fatal("public visibility should be preserved")
	}
	if normalizeVisibility("team") != "private" {
		t.Fatal("unknown visibility should fall back to private")
	}
	if normalizeSourceType("url") != "url" {
		t.Fatal("known source type should be preserved")
	}
	if normalizeSourceType("unknown") != "manual" {
		t.Fatal("unknown source type should fall back to manual")
	}
}

func TestContentHashIsStableAndDistinct(t *testing.T) {
	first := contentHash("hello")
	second := contentHash("hello")
	other := contentHash("world")

	if first == "" || first != second {
		t.Fatalf("hash should be stable, got %q and %q", first, second)
	}
	if first == other {
		t.Fatal("different content should produce a different hash")
	}
}
