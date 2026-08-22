package workspace

import (
	"path/filepath"
	"strings"

	"freedinner/backend/internal/store"
)

func normalizeArgs(args []string) []string {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		result = append(result, strings.TrimSpace(arg))
	}
	return result
}

func hasUnsafeArg(args []string) bool {
	for _, arg := range args {
		if arg == "" {
			continue
		}
		if filepath.IsAbs(arg) || hasParentSegment(arg) {
			return true
		}
		if strings.ContainsAny(arg, "\x00\n\r") {
			return true
		}
	}
	return false
}

func isAllowedCommand(command string, args []string) bool {
	switch command {
	case "pwd", "ls", "cat", "mkdir", "touch":
		return true
	case "node":
		return len(args) > 0 && !strings.HasPrefix(args[0], "-")
	case "python", "python3":
		return len(args) > 0 && !strings.HasPrefix(args[0], "-")
	case "go":
		return len(args) > 0 && oneOf(args[0], "version", "env", "test", "run")
	case "npm":
		return len(args) > 0 && oneOf(args[0], "test", "run", "--version", "-v")
	default:
		return false
	}
}

func policyUpdateFromWorkspace(input UpdatePolicyInput, workspace store.UserWorkspace) store.WorkspacePolicyUpdate {
	update := store.WorkspacePolicyUpdate{
		UserID:              input.UserID,
		WorkspaceID:         workspace.ID,
		SandboxType:         workspace.SandboxType,
		NetworkPolicy:       workspace.NetworkPolicy,
		NetworkAllowlist:    workspace.NetworkAllowlist,
		MaxDiskBytes:        workspace.MaxDiskBytes,
		MaxFileCount:        workspace.MaxFileCount,
		MaxSingleFileBytes:  workspace.MaxSingleFileBytes,
		MaxCommandSeconds:   workspace.MaxCommandSeconds,
		MaxStdoutBytes:      workspace.MaxStdoutBytes,
		MaxStderrBytes:      workspace.MaxStderrBytes,
		CPULimit:            workspace.CPULimit,
		MemoryLimitBytes:    workspace.MemoryLimitBytes,
		IdleAfterSeconds:    workspace.IdleAfterSeconds,
		DestroyAfterSeconds: workspace.DestroyAfterSeconds,
	}
	if input.SandboxType != nil && strings.TrimSpace(*input.SandboxType) != "" {
		update.SandboxType = strings.TrimSpace(*input.SandboxType)
	}
	if input.NetworkPolicy != nil && strings.TrimSpace(*input.NetworkPolicy) != "" {
		update.NetworkPolicy = strings.TrimSpace(*input.NetworkPolicy)
	}
	if input.NetworkAllowlistSet {
		update.NetworkAllowlist = input.NetworkAllowlist
	}
	if input.MaxDiskBytes != nil && *input.MaxDiskBytes > 0 {
		update.MaxDiskBytes = *input.MaxDiskBytes
	}
	if input.MaxFileCount != nil && *input.MaxFileCount > 0 {
		update.MaxFileCount = *input.MaxFileCount
	}
	if input.MaxSingleFileBytes != nil && *input.MaxSingleFileBytes > 0 {
		update.MaxSingleFileBytes = *input.MaxSingleFileBytes
	}
	if input.MaxCommandSeconds != nil && *input.MaxCommandSeconds > 0 {
		update.MaxCommandSeconds = *input.MaxCommandSeconds
	}
	if input.MaxStdoutBytes != nil && *input.MaxStdoutBytes > 0 {
		update.MaxStdoutBytes = *input.MaxStdoutBytes
	}
	if input.MaxStderrBytes != nil && *input.MaxStderrBytes > 0 {
		update.MaxStderrBytes = *input.MaxStderrBytes
	}
	if input.CPULimitSet {
		update.CPULimit = trimStringPointer(input.CPULimit)
	}
	if input.MemoryLimitSet {
		update.MemoryLimitBytes = input.MemoryLimitBytes
	}
	if input.IdleAfterSeconds != nil && *input.IdleAfterSeconds > 0 {
		update.IdleAfterSeconds = *input.IdleAfterSeconds
	}
	if input.DestroyAfterSeconds != nil && *input.DestroyAfterSeconds > 0 {
		update.DestroyAfterSeconds = *input.DestroyAfterSeconds
	}
	return update
}

func normalizeEnableInput(input EnableInput) EnableInput {
	if input.SandboxType == "" {
		input.SandboxType = "local_dir"
	}
	if input.NetworkPolicy == "" {
		input.NetworkPolicy = "disabled"
	}
	if input.NetworkAllowlist == nil {
		input.NetworkAllowlist = []string{}
	}
	if input.MaxDiskBytes <= 0 {
		input.MaxDiskBytes = 1073741824
	}
	if input.MaxFileCount <= 0 {
		input.MaxFileCount = 5000
	}
	if input.MaxSingleFileBytes <= 0 {
		input.MaxSingleFileBytes = 52428800
	}
	if input.MaxCommandSeconds <= 0 {
		input.MaxCommandSeconds = 30
	}
	if input.MaxStdoutBytes <= 0 {
		input.MaxStdoutBytes = 262144
	}
	if input.MaxStderrBytes <= 0 {
		input.MaxStderrBytes = 262144
	}
	if input.IdleAfterSeconds <= 0 {
		input.IdleAfterSeconds = 604800
	}
	if input.DestroyAfterSeconds <= 0 {
		input.DestroyAfterSeconds = 2592000
	}
	return input
}
