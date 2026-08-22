package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

func scanUserWorkspace(row pgx.Row) (UserWorkspace, error) {
	var workspace UserWorkspace
	if err := row.Scan(
		&workspace.ID,
		&workspace.UserID,
		&workspace.Status,
		&workspace.RootPath,
		&workspace.SandboxType,
		&workspace.NetworkPolicy,
		&workspace.NetworkAllowlist,
		&workspace.MaxDiskBytes,
		&workspace.MaxFileCount,
		&workspace.MaxSingleFileBytes,
		&workspace.MaxCommandSeconds,
		&workspace.MaxStdoutBytes,
		&workspace.MaxStderrBytes,
		&workspace.CPULimit,
		&workspace.MemoryLimitBytes,
		&workspace.LastActiveAt,
		&workspace.IdleAfterSeconds,
		&workspace.DestroyAfterSeconds,
		&workspace.Metadata,
		&workspace.CreatedAt,
		&workspace.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserWorkspace{}, ErrNotFound
		}
		return UserWorkspace{}, err
	}
	return workspace, nil
}

func scanWorkspaceFile(row pgx.Row) (WorkspaceFile, error) {
	var file WorkspaceFile
	if err := row.Scan(
		&file.ID,
		&file.UserID,
		&file.WorkspaceID,
		&file.RelativePath,
		&file.FileType,
		&file.SizeBytes,
		&file.ContentHash,
		&file.MimeType,
		&file.CreatedBy,
		&file.Status,
		&file.Metadata,
		&file.CreatedAt,
		&file.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkspaceFile{}, ErrNotFound
		}
		return WorkspaceFile{}, err
	}
	return file, nil
}

func scanWorkspaceEvent(row pgx.Row) (WorkspaceEvent, error) {
	var event WorkspaceEvent
	if err := row.Scan(
		&event.ID,
		&event.UserID,
		&event.WorkspaceID,
		&event.EventType,
		&event.ActorType,
		&event.ActorID,
		&event.FileID,
		&event.CommandRunID,
		&event.Metadata,
		&event.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkspaceEvent{}, ErrNotFound
		}
		return WorkspaceEvent{}, err
	}
	return event, nil
}

func scanWorkspaceCommandRun(row pgx.Row) (WorkspaceCommandRun, error) {
	var run WorkspaceCommandRun
	if err := row.Scan(
		&run.ID,
		&run.UserID,
		&run.WorkspaceID,
		&run.ConversationID,
		&run.AgentTurnID,
		&run.ToolCallID,
		&run.Command,
		&run.Args,
		&run.WorkingDir,
		&run.NetworkPolicy,
		&run.Status,
		&run.ExitCode,
		&run.StdoutPreview,
		&run.StderrPreview,
		&run.StdoutTruncated,
		&run.StderrTruncated,
		&run.DurationMS,
		&run.ErrorMessage,
		&run.Metadata,
		&run.CreatedAt,
		&run.StartedAt,
		&run.FinishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkspaceCommandRun{}, ErrNotFound
		}
		return WorkspaceCommandRun{}, err
	}
	return run, nil
}

func scanWorkspaceQuotaSnapshot(row pgx.Row) (WorkspaceQuotaSnapshot, error) {
	var snapshot WorkspaceQuotaSnapshot
	if err := row.Scan(
		&snapshot.ID,
		&snapshot.UserID,
		&snapshot.WorkspaceID,
		&snapshot.UsedDiskBytes,
		&snapshot.FileCount,
		&snapshot.CommandCount,
		&snapshot.ActiveProcessCount,
		&snapshot.Metadata,
		&snapshot.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkspaceQuotaSnapshot{}, ErrNotFound
		}
		return WorkspaceQuotaSnapshot{}, err
	}
	return snapshot, nil
}
