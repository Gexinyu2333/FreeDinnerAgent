package agent

import (
	"errors"
	"testing"
)

func TestNormalizeLoopLimits(t *testing.T) {
	if got := NormalizeMaxLoopSteps(0); got != 6 {
		t.Fatalf("expected default max steps 6, got %d", got)
	}
	if got := NormalizeMaxLoopSteps(99); got != 12 {
		t.Fatalf("expected capped max steps 12, got %d", got)
	}
	if got := NormalizeRetryLimit(99); got != 3 {
		t.Fatalf("expected capped retry limit 3, got %d", got)
	}
}

func TestIsRetryableLLMError(t *testing.T) {
	if !IsRetryableLLMError(errors.New("openai status 503")) {
		t.Fatal("expected 503 to be retryable")
	}
	if IsRetryableLLMError(errors.New("missing api key")) {
		t.Fatal("did not expect missing api key to be retryable")
	}
}
