package workspace

import (
	"context"
	"io/fs"
	"os"
	"strings"

	"freedinner/backend/internal/store"
)

func (s *Service) RunCommand(ctx context.Context, input RunCommandInput) (RunCommandResult, error) {
	workspace, err := s.activeWorkspace(ctx, input.UserID)
	if err != nil {
		return RunCommandResult{}, err
	}
	command := strings.TrimSpace(input.Command)
	args := normalizeArgs(input.Args)
	if !isAllowedCommand(command, args) || hasUnsafeArg(args) {
		return s.recordBlockedCommand(ctx, workspace, input, command, args, ErrCommandBlocked.Error())
	}

	workingDir := strings.TrimSpace(input.WorkingDir)
	if workingDir == "" {
		workingDir = "/"
	}
	fullWorkingDir, relativeWorkingDir, err := s.resolve(workspace, workingDir)
	if err != nil {
		return RunCommandResult{}, err
	}
	info, err := os.Stat(fullWorkingDir)
	if err != nil {
		return RunCommandResult{}, err
	}
	if !info.IsDir() {
		return RunCommandResult{}, fs.ErrInvalid
	}

	timeout := input.TimeoutSeconds
	if timeout <= 0 || timeout > workspace.MaxCommandSeconds {
		timeout = workspace.MaxCommandSeconds
	}
	runner, err := newRunner(workspace.SandboxType, s.runners)
	if err != nil {
		return s.recordBlockedCommand(ctx, workspace, input, command, args, err.Error())
	}
	run, err := s.workspaces.CreateCommandRun(ctx, store.WorkspaceCommandRunCreate{
		UserID:         input.UserID,
		WorkspaceID:    workspace.ID,
		ConversationID: input.ConversationID,
		AgentTurnID:    input.AgentTurnID,
		ToolCallID:     input.ToolCallID,
		Command:        command,
		Args:           mustJSON(args),
		WorkingDir:     relativeWorkingDirOrDot(relativeWorkingDir),
		NetworkPolicy:  workspace.NetworkPolicy,
		Metadata:       mustJSON(map[string]any{"timeout_seconds": timeout}),
	})
	if err != nil {
		return RunCommandResult{}, err
	}
	_, _ = s.workspaces.CreateEvent(ctx, store.WorkspaceEvent{
		UserID:       input.UserID,
		WorkspaceID:  workspace.ID,
		EventType:    "command_started",
		ActorType:    "user",
		CommandRunID: &run.ID,
		Metadata:     mustJSON(map[string]any{"command": command, "args": args, "working_dir": "/" + relativeWorkingDir}),
	})

	execution, err := runner.Execute(ctx, CommandRequest{
		Workspace:       workspace,
		Command:         command,
		Args:            args,
		FullWorkingDir:  fullWorkingDir,
		RelativeWorkDir: relativeWorkingDir,
		TimeoutSeconds:  timeout,
	})
	if err != nil {
		return RunCommandResult{}, err
	}
	finished, finishErr := s.workspaces.FinishCommandRun(ctx, store.WorkspaceCommandRunFinish{
		ID:              run.ID,
		UserID:          input.UserID,
		Status:          execution.Status,
		ExitCode:        execution.ExitCode,
		StdoutPreview:   stringPtr(execution.Stdout),
		StderrPreview:   stringPtr(execution.Stderr),
		StdoutTruncated: execution.StdoutTruncated,
		StderrTruncated: execution.StderrTruncated,
		DurationMS:      &execution.DurationMS,
		ErrorMessage:    execution.ErrorMessage,
	})
	if finishErr != nil {
		return RunCommandResult{}, finishErr
	}
	_, _ = s.workspaces.CreateEvent(ctx, store.WorkspaceEvent{
		UserID:       input.UserID,
		WorkspaceID:  workspace.ID,
		EventType:    "command_finished",
		ActorType:    "user",
		CommandRunID: &finished.ID,
		Metadata:     mustJSON(mergeMetadata(execution.Metadata, map[string]any{"status": execution.Status, "duration_ms": execution.DurationMS})),
	})
	_ = s.workspaces.Touch(ctx, input.UserID, workspace.ID)
	return RunCommandResult{Run: finished}, nil
}

func (s *Service) ListCommandRuns(ctx context.Context, userID string, limit int) ([]store.WorkspaceCommandRun, error) {
	workspace, err := s.activeWorkspace(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.workspaces.ListCommandRuns(ctx, userID, workspace.ID, limit)
}

func (s *Service) recordBlockedCommand(ctx context.Context, workspace store.UserWorkspace, input RunCommandInput, command string, args []string, message string) (RunCommandResult, error) {
	run, err := s.workspaces.CreateCommandRun(ctx, store.WorkspaceCommandRunCreate{
		UserID:         input.UserID,
		WorkspaceID:    workspace.ID,
		ConversationID: input.ConversationID,
		AgentTurnID:    input.AgentTurnID,
		ToolCallID:     input.ToolCallID,
		Command:        command,
		Args:           mustJSON(args),
		WorkingDir:     ".",
		NetworkPolicy:  workspace.NetworkPolicy,
	})
	if err != nil {
		return RunCommandResult{}, err
	}
	duration := 0
	finished, err := s.workspaces.FinishCommandRun(ctx, store.WorkspaceCommandRunFinish{
		ID:           run.ID,
		UserID:       input.UserID,
		Status:       "blocked",
		DurationMS:   &duration,
		ErrorMessage: &message,
	})
	if err != nil {
		return RunCommandResult{}, err
	}
	_, _ = s.workspaces.CreateEvent(ctx, store.WorkspaceEvent{
		UserID:       input.UserID,
		WorkspaceID:  workspace.ID,
		EventType:    "security_blocked",
		ActorType:    "user",
		CommandRunID: &finished.ID,
		Metadata:     mustJSON(map[string]any{"command": command, "args": args, "reason": message}),
	})
	return RunCommandResult{Run: finished}, ErrCommandBlocked
}
