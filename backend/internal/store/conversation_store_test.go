package store

import "testing"

func TestDetectAnchor(t *testing.T) {
	isAnchor, reason := detectAnchor("以后回答我尽量简洁，这个很重要")
	if !isAnchor {
		t.Fatal("expected anchor")
	}
	if reason == nil || *reason == "" {
		t.Fatal("expected anchor reason")
	}
}

func TestDetectAnchorCorrectionOrConflict(t *testing.T) {
	isAnchor, reason := detectAnchor("不对，我的默认语言改成中文")
	if !isAnchor {
		t.Fatal("expected correction anchor")
	}
	if reason == nil || *reason != "correction_or_conflict" {
		t.Fatalf("expected correction_or_conflict, got %v", reason)
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	if got := estimateMessageTokens("你好世界"); got <= 0 {
		t.Fatalf("expected positive token estimate, got %d", got)
	}
}
