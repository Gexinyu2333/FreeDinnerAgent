package tool

import (
	"testing"

	"freedinner/backend/internal/store"
)

func TestShouldRequireApproval(t *testing.T) {
	cases := []struct {
		name   string
		policy string
		tool   store.ToolDefinition
		want   bool
	}{
		{
			name:   "readonly does not require approval by default",
			policy: "sensitive_only",
			tool:   store.ToolDefinition{PermissionLevel: "readonly"},
			want:   false,
		},
		{
			name:   "explicit approval flag wins by default",
			policy: "sensitive_only",
			tool:   store.ToolDefinition{PermissionLevel: "normal", RequiresApproval: true},
			want:   true,
		},
		{
			name:   "sensitive requires approval by default",
			policy: "sensitive_only",
			tool:   store.ToolDefinition{PermissionLevel: "sensitive"},
			want:   true,
		},
		{
			name:   "destructive requires approval by default",
			policy: "sensitive_only",
			tool:   store.ToolDefinition{PermissionLevel: "destructive"},
			want:   true,
		},
		{
			name:   "always requires approval for readonly",
			policy: "always",
			tool:   store.ToolDefinition{PermissionLevel: "readonly"},
			want:   true,
		},
		{
			name:   "never skips explicit approval flag",
			policy: "never",
			tool:   store.ToolDefinition{PermissionLevel: "destructive", RequiresApproval: true},
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRequireApproval(tc.tool, tc.policy); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestNormalizeApprovalPolicy(t *testing.T) {
	if got := normalizeApprovalPolicy("always"); got != "always" {
		t.Fatalf("expected always, got %q", got)
	}
	if got := normalizeApprovalPolicy("never"); got != "never" {
		t.Fatalf("expected never, got %q", got)
	}
	if got := normalizeApprovalPolicy("weird"); got != "sensitive_only" {
		t.Fatalf("expected sensitive_only, got %q", got)
	}
}

func TestRiskLevel(t *testing.T) {
	if got := riskLevel(store.ToolDefinition{PermissionLevel: "readonly"}); got != "normal" {
		t.Fatalf("expected normal, got %q", got)
	}
	if got := riskLevel(store.ToolDefinition{PermissionLevel: "sensitive"}); got != "sensitive" {
		t.Fatalf("expected sensitive, got %q", got)
	}
	if got := riskLevel(store.ToolDefinition{PermissionLevel: "destructive"}); got != "destructive" {
		t.Fatalf("expected destructive, got %q", got)
	}
}

func TestActiveVersionDefaultsToOne(t *testing.T) {
	if got := activeVersion(store.ToolDefinition{}); got != 1 {
		t.Fatalf("expected default version 1, got %d", got)
	}
	version := 3
	if got := activeVersion(store.ToolDefinition{ActiveVersion: &version}); got != 3 {
		t.Fatalf("expected version 3, got %d", got)
	}
}

func TestResolveApprovalRejectsInvalidStatus(t *testing.T) {
	service := &Service{}
	if _, err := service.ResolveApproval(nil, "user-1", "approval-1", "pending"); err == nil {
		t.Fatal("expected invalid approval status error")
	}
}
