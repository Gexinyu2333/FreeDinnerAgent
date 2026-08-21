package workspace

import (
	"slices"
	"testing"

	"freedinner/backend/internal/store"
)

func TestBuildContainerArgsUsesIsolationOptions(t *testing.T) {
	cpu := "1.5"
	memory := int64(536870912)
	args := buildContainerArgs(CommandRequest{
		Workspace: store.UserWorkspace{
			RootPath:         "/var/lib/freedinner/workspaces/user-1",
			SandboxType:      "docker",
			NetworkPolicy:    "disabled",
			CPULimit:         &cpu,
			MemoryLimitBytes: &memory,
		},
		Command:         "python3",
		Args:            []string{"main.py"},
		RelativeWorkDir: "project",
		TimeoutSeconds:  30,
	}, "freedinner-agent-sandbox:latest")

	assertContainsSequence(t, args, "--workdir", "/workspace/project")
	assertContainsSequence(t, args, "--volume", "/var/lib/freedinner/workspaces/user-1/files:/workspace:rw")
	assertContainsSequence(t, args, "--network", "none")
	assertContainsSequence(t, args, "--memory", "536870912")
	assertContainsSequence(t, args, "--cpus", "1.5")
	assertContainsSequence(t, args, "--cap-drop", "ALL")
	assertContainsSequence(t, args, "--security-opt", "no-new-privileges")
	assertContainsSequence(t, args, "--pids-limit", "64")
	assertContainsSequence(t, args, "freedinner-agent-sandbox:latest", "python3")
	assertContainsSequence(t, args, "python3", "main.py")
}

func TestBuildNsJailArgsUsesIsolationOptions(t *testing.T) {
	memory := int64(268435456)
	args := buildNsJailArgs(CommandRequest{
		Workspace: store.UserWorkspace{
			RootPath:         "/srv/workspaces/user-2",
			SandboxType:      "nsjail",
			NetworkPolicy:    "disabled",
			MemoryLimitBytes: &memory,
		},
		Command:         "go",
		Args:            []string{"test", "./..."},
		RelativeWorkDir: "",
		TimeoutSeconds:  12,
	})

	assertContainsSequence(t, args, "--time_limit", "12")
	assertContainsSequence(t, args, "--cwd", "/workspace")
	assertContainsSequence(t, args, "--bindmount", "/srv/workspaces/user-2/files:/workspace")
	assertContainsSequence(t, args, "--clone_newnet")
	assertContainsSequence(t, args, "--rlimit_nproc", "64")
	assertContainsSequence(t, args, "--rlimit_as", "268435456")
	assertContainsSequence(t, args, "--", "go")
	assertContainsSequence(t, args, "go", "test")
}

func TestNewRunnerRejectsUnsupportedSandbox(t *testing.T) {
	if _, err := newRunner("bare_metal", RunnerOptions{}); err == nil {
		t.Fatal("expected unsupported sandbox error")
	}
}

func assertContainsSequence(t *testing.T, args []string, first string, rest ...string) {
	t.Helper()
	index := slices.Index(args, first)
	if index < 0 {
		t.Fatalf("expected %q in args: %#v", first, args)
	}
	for offset, expected := range rest {
		actualIndex := index + offset + 1
		if actualIndex >= len(args) {
			t.Fatalf("expected %q after %q in args: %#v", expected, first, args)
		}
		if args[actualIndex] != expected {
			t.Fatalf("expected %q after %q, got %q in args: %#v", expected, first, args[actualIndex], args)
		}
	}
}
