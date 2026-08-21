package workspace

import (
	"path/filepath"
	"testing"

	"freedinner/backend/internal/store"
)

func TestResolveRawRejectsParentTraversal(t *testing.T) {
	service := &Service{}
	workspace := store.UserWorkspace{RootPath: t.TempDir()}

	if _, _, err := service.resolveRaw(workspace, "../escape.txt"); err == nil {
		t.Fatal("expected parent traversal to be rejected")
	}
	if _, _, err := service.resolveRaw(workspace, "/notes/../../escape.txt"); err == nil {
		t.Fatal("expected nested parent traversal to be rejected")
	}
}

func TestResolveRawScopesPathsUnderFilesDirectory(t *testing.T) {
	service := &Service{}
	root := t.TempDir()
	workspace := store.UserWorkspace{RootPath: root}

	fullPath, relativePath, err := service.resolveRaw(workspace, "/notes/todo.md")
	if err != nil {
		t.Fatalf("resolveRaw returned error: %v", err)
	}
	expected := filepath.Join(root, "files", "notes", "todo.md")
	if fullPath != expected {
		t.Fatalf("expected full path %q, got %q", expected, fullPath)
	}
	if relativePath != "notes/todo.md" {
		t.Fatalf("expected relative path notes/todo.md, got %q", relativePath)
	}
}

func TestCommandPolicyBlocksUnsafeCommandsAndArgs(t *testing.T) {
	if isAllowedCommand("rm", []string{"notes/todo.md"}) {
		t.Fatal("rm should not be allowed")
	}
	if isAllowedCommand("python3", []string{"-c", "print(1)"}) {
		t.Fatal("python -c should not be allowed")
	}
	if !isAllowedCommand("go", []string{"test", "./..."}) {
		t.Fatal("go test should be allowed")
	}
	if !hasUnsafeArg([]string{"../secret.txt"}) {
		t.Fatal("parent traversal arg should be unsafe")
	}
	if !hasUnsafeArg([]string{"/etc/passwd"}) {
		t.Fatal("absolute arg should be unsafe")
	}
	if hasUnsafeArg([]string{"notes/todo.md", "--verbose"}) {
		t.Fatal("normal relative args should be safe")
	}
}

func TestPolicyUpdateFromWorkspacePreservesOmittedFields(t *testing.T) {
	cpu := "1.0"
	memory := int64(536870912)
	workspace := store.UserWorkspace{
		ID:                  "workspace-1",
		UserID:              "user-1",
		SandboxType:         "local_dir",
		NetworkPolicy:       "disabled",
		NetworkAllowlist:    []string{"example.com"},
		MaxDiskBytes:        1000,
		MaxFileCount:        10,
		MaxSingleFileBytes:  100,
		MaxCommandSeconds:   5,
		MaxStdoutBytes:      200,
		MaxStderrBytes:      300,
		CPULimit:            &cpu,
		MemoryLimitBytes:    &memory,
		IdleAfterSeconds:    60,
		DestroyAfterSeconds: 120,
	}

	network := "allowlist"
	maxCommandSeconds := 9
	update := policyUpdateFromWorkspace(UpdatePolicyInput{
		UserID:            "user-1",
		NetworkPolicy:     &network,
		MaxCommandSeconds: &maxCommandSeconds,
	}, workspace)

	if update.SandboxType != "local_dir" {
		t.Fatalf("expected sandbox type to be preserved, got %q", update.SandboxType)
	}
	if update.NetworkPolicy != "allowlist" {
		t.Fatalf("expected network policy to update, got %q", update.NetworkPolicy)
	}
	if update.MaxCommandSeconds != 9 {
		t.Fatalf("expected max command seconds to update, got %d", update.MaxCommandSeconds)
	}
	if update.MaxDiskBytes != 1000 || update.MaxFileCount != 10 {
		t.Fatalf("expected omitted quotas to be preserved: %#v", update)
	}
	if update.CPULimit == nil || *update.CPULimit != "1.0" {
		t.Fatalf("expected cpu limit to be preserved: %#v", update.CPULimit)
	}
	if update.MemoryLimitBytes == nil || *update.MemoryLimitBytes != 536870912 {
		t.Fatalf("expected memory limit to be preserved: %#v", update.MemoryLimitBytes)
	}
}
