package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserWorkspace struct {
	ID                  string          `json:"id"`
	UserID              string          `json:"user_id"`
	Status              string          `json:"status"`
	RootPath            string          `json:"root_path"`
	SandboxType         string          `json:"sandbox_type"`
	NetworkPolicy       string          `json:"network_policy"`
	NetworkAllowlist    []string        `json:"network_allowlist"`
	MaxDiskBytes        int64           `json:"max_disk_bytes"`
	MaxFileCount        int             `json:"max_file_count"`
	MaxSingleFileBytes  int64           `json:"max_single_file_bytes"`
	MaxCommandSeconds   int             `json:"max_command_seconds"`
	MaxStdoutBytes      int             `json:"max_stdout_bytes"`
	MaxStderrBytes      int             `json:"max_stderr_bytes"`
	CPULimit            *string         `json:"cpu_limit"`
	MemoryLimitBytes    *int64          `json:"memory_limit_bytes"`
	LastActiveAt        *time.Time      `json:"last_active_at"`
	IdleAfterSeconds    int             `json:"idle_after_seconds"`
	DestroyAfterSeconds int             `json:"destroy_after_seconds"`
	Metadata            json.RawMessage `json:"metadata"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type WorkspaceFile struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	WorkspaceID  string          `json:"workspace_id"`
	RelativePath string          `json:"relative_path"`
	FileType     string          `json:"file_type"`
	SizeBytes    int64           `json:"size_bytes"`
	ContentHash  *string         `json:"content_hash"`
	MimeType     *string         `json:"mime_type"`
	CreatedBy    string          `json:"created_by"`
	Status       string          `json:"status"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type WorkspaceEvent struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	WorkspaceID  string          `json:"workspace_id"`
	EventType    string          `json:"event_type"`
	ActorType    string          `json:"actor_type"`
	ActorID      *string         `json:"actor_id"`
	FileID       *string         `json:"file_id"`
	CommandRunID *string         `json:"command_run_id"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
}

type WorkspaceQuotaSnapshot struct {
	ID                 string          `json:"id"`
	UserID             string          `json:"user_id"`
	WorkspaceID        string          `json:"workspace_id"`
	UsedDiskBytes      int64           `json:"used_disk_bytes"`
	FileCount          int             `json:"file_count"`
	CommandCount       int             `json:"command_count"`
	ActiveProcessCount int             `json:"active_process_count"`
	Metadata           json.RawMessage `json:"metadata"`
	CreatedAt          time.Time       `json:"created_at"`
}

type WorkspaceCommandRun struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	WorkspaceID     string          `json:"workspace_id"`
	ConversationID  *string         `json:"conversation_id"`
	AgentTurnID     *string         `json:"agent_turn_id"`
	ToolCallID      *string         `json:"tool_call_id"`
	Command         string          `json:"command"`
	Args            json.RawMessage `json:"args"`
	WorkingDir      string          `json:"working_dir"`
	NetworkPolicy   string          `json:"network_policy"`
	Status          string          `json:"status"`
	ExitCode        *int            `json:"exit_code"`
	StdoutPreview   *string         `json:"stdout_preview"`
	StderrPreview   *string         `json:"stderr_preview"`
	StdoutTruncated bool            `json:"stdout_truncated"`
	StderrTruncated bool            `json:"stderr_truncated"`
	DurationMS      *int            `json:"duration_ms"`
	ErrorMessage    *string         `json:"error_message"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	StartedAt       *time.Time      `json:"started_at"`
	FinishedAt      *time.Time      `json:"finished_at"`
}

type WorkspaceUpsert struct {
	UserID              string
	RootPath            string
	SandboxType         string
	NetworkPolicy       string
	NetworkAllowlist    []string
	MaxDiskBytes        int64
	MaxFileCount        int
	MaxSingleFileBytes  int64
	MaxCommandSeconds   int
	MaxStdoutBytes      int
	MaxStderrBytes      int
	CPULimit            *string
	MemoryLimitBytes    *int64
	IdleAfterSeconds    int
	DestroyAfterSeconds int
	Metadata            json.RawMessage
}

type WorkspaceFileUpsert struct {
	UserID       string
	WorkspaceID  string
	RelativePath string
	FileType     string
	SizeBytes    int64
	ContentHash  *string
	MimeType     *string
	CreatedBy    string
	Metadata     json.RawMessage
}

type WorkspaceCommandRunCreate struct {
	UserID         string
	WorkspaceID    string
	ConversationID *string
	AgentTurnID    *string
	ToolCallID     *string
	Command        string
	Args           json.RawMessage
	WorkingDir     string
	NetworkPolicy  string
	Metadata       json.RawMessage
}

type WorkspaceCommandRunFinish struct {
	ID              string
	UserID          string
	Status          string
	ExitCode        *int
	StdoutPreview   *string
	StderrPreview   *string
	StdoutTruncated bool
	StderrTruncated bool
	DurationMS      *int
	ErrorMessage    *string
}

type WorkspaceStore struct {
	db *pgxpool.Pool
}

func NewWorkspaceStore(db *pgxpool.Pool) *WorkspaceStore {
	return &WorkspaceStore{db: db}
}

func (s *WorkspaceStore) FindByUserID(ctx context.Context, userID string) (UserWorkspace, error) {
	return scanUserWorkspace(s.db.QueryRow(ctx, `
		SELECT id, user_id, status, root_path, sandbox_type, network_policy, network_allowlist,
			max_disk_bytes, max_file_count, max_single_file_bytes, max_command_seconds,
			max_stdout_bytes, max_stderr_bytes, cpu_limit, memory_limit_bytes, last_active_at,
			idle_after_seconds, destroy_after_seconds, metadata, created_at, updated_at
		FROM user_workspaces
		WHERE user_id = $1 AND status <> 'destroyed'
	`, userID))
}

func (s *WorkspaceStore) UpsertActive(ctx context.Context, input WorkspaceUpsert) (UserWorkspace, error) {
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	return scanUserWorkspace(s.db.QueryRow(ctx, `
		INSERT INTO user_workspaces (
			id, user_id, status, root_path, sandbox_type, network_policy, network_allowlist,
			max_disk_bytes, max_file_count, max_single_file_bytes, max_command_seconds,
			max_stdout_bytes, max_stderr_bytes, cpu_limit, memory_limit_bytes,
			idle_after_seconds, destroy_after_seconds, metadata, last_active_at
		)
		VALUES (
			$1, $2, 'active', $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, $17, NOW()
		)
		ON CONFLICT (user_id) DO UPDATE SET
			status = 'active',
			root_path = EXCLUDED.root_path,
			sandbox_type = EXCLUDED.sandbox_type,
			network_policy = EXCLUDED.network_policy,
			network_allowlist = EXCLUDED.network_allowlist,
			max_disk_bytes = EXCLUDED.max_disk_bytes,
			max_file_count = EXCLUDED.max_file_count,
			max_single_file_bytes = EXCLUDED.max_single_file_bytes,
			max_command_seconds = EXCLUDED.max_command_seconds,
			max_stdout_bytes = EXCLUDED.max_stdout_bytes,
			max_stderr_bytes = EXCLUDED.max_stderr_bytes,
			cpu_limit = EXCLUDED.cpu_limit,
			memory_limit_bytes = EXCLUDED.memory_limit_bytes,
			idle_after_seconds = EXCLUDED.idle_after_seconds,
			destroy_after_seconds = EXCLUDED.destroy_after_seconds,
			metadata = EXCLUDED.metadata,
			last_active_at = NOW(),
			updated_at = NOW()
		RETURNING id, user_id, status, root_path, sandbox_type, network_policy, network_allowlist,
			max_disk_bytes, max_file_count, max_single_file_bytes, max_command_seconds,
			max_stdout_bytes, max_stderr_bytes, cpu_limit, memory_limit_bytes, last_active_at,
			idle_after_seconds, destroy_after_seconds, metadata, created_at, updated_at
	`, uuid.NewString(), input.UserID, input.RootPath, input.SandboxType, input.NetworkPolicy, input.NetworkAllowlist,
		input.MaxDiskBytes, input.MaxFileCount, input.MaxSingleFileBytes, input.MaxCommandSeconds,
		input.MaxStdoutBytes, input.MaxStderrBytes, input.CPULimit, input.MemoryLimitBytes,
		input.IdleAfterSeconds, input.DestroyAfterSeconds, input.Metadata))
}

func (s *WorkspaceStore) Touch(ctx context.Context, userID, workspaceID string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE user_workspaces
		SET last_active_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, workspaceID, userID)
	return err
}

func (s *WorkspaceStore) UpsertFile(ctx context.Context, input WorkspaceFileUpsert) (WorkspaceFile, error) {
	if input.CreatedBy == "" {
		input.CreatedBy = "agent"
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	return scanWorkspaceFile(s.db.QueryRow(ctx, `
		INSERT INTO workspace_files (
			id, user_id, workspace_id, relative_path, file_type, size_bytes,
			content_hash, mime_type, created_by, status, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', $10)
		ON CONFLICT (workspace_id, relative_path) DO UPDATE SET
			file_type = EXCLUDED.file_type,
			size_bytes = EXCLUDED.size_bytes,
			content_hash = EXCLUDED.content_hash,
			mime_type = EXCLUDED.mime_type,
			status = 'active',
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, user_id, workspace_id, relative_path, file_type, size_bytes,
			content_hash, mime_type, created_by, status, metadata, created_at, updated_at
	`, uuid.NewString(), input.UserID, input.WorkspaceID, input.RelativePath, input.FileType, input.SizeBytes,
		input.ContentHash, input.MimeType, input.CreatedBy, input.Metadata))
}

func (s *WorkspaceStore) CreateEvent(ctx context.Context, event WorkspaceEvent) (WorkspaceEvent, error) {
	if event.ActorType == "" {
		event.ActorType = "system"
	}
	if len(event.Metadata) == 0 {
		event.Metadata = json.RawMessage(`{}`)
	}
	return scanWorkspaceEvent(s.db.QueryRow(ctx, `
		INSERT INTO workspace_events (
			id, user_id, workspace_id, event_type, actor_type, actor_id, file_id, command_run_id, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, workspace_id, event_type, actor_type, actor_id, file_id, command_run_id, metadata, created_at
	`, uuid.NewString(), event.UserID, event.WorkspaceID, event.EventType, event.ActorType,
		event.ActorID, event.FileID, event.CommandRunID, event.Metadata))
}

func (s *WorkspaceStore) CreateCommandRun(ctx context.Context, input WorkspaceCommandRunCreate) (WorkspaceCommandRun, error) {
	if len(input.Args) == 0 {
		input.Args = json.RawMessage(`[]`)
	}
	if input.WorkingDir == "" {
		input.WorkingDir = "."
	}
	if input.NetworkPolicy == "" {
		input.NetworkPolicy = "disabled"
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	return scanWorkspaceCommandRun(s.db.QueryRow(ctx, `
		INSERT INTO workspace_command_runs (
			id, user_id, workspace_id, conversation_id, agent_turn_id, tool_call_id,
			command, args, working_dir, network_policy, status, metadata, started_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'running', $11, NOW())
		RETURNING id, user_id, workspace_id, conversation_id, agent_turn_id, tool_call_id,
			command, args, working_dir, network_policy, status, exit_code, stdout_preview,
			stderr_preview, stdout_truncated, stderr_truncated, duration_ms, error_message,
			metadata, created_at, started_at, finished_at
	`, uuid.NewString(), input.UserID, input.WorkspaceID, input.ConversationID, input.AgentTurnID,
		input.ToolCallID, input.Command, input.Args, input.WorkingDir, input.NetworkPolicy, input.Metadata))
}

func (s *WorkspaceStore) FinishCommandRun(ctx context.Context, input WorkspaceCommandRunFinish) (WorkspaceCommandRun, error) {
	return scanWorkspaceCommandRun(s.db.QueryRow(ctx, `
		UPDATE workspace_command_runs
		SET status = $3,
			exit_code = $4,
			stdout_preview = $5,
			stderr_preview = $6,
			stdout_truncated = $7,
			stderr_truncated = $8,
			duration_ms = $9,
			error_message = $10,
			finished_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, workspace_id, conversation_id, agent_turn_id, tool_call_id,
			command, args, working_dir, network_policy, status, exit_code, stdout_preview,
			stderr_preview, stdout_truncated, stderr_truncated, duration_ms, error_message,
			metadata, created_at, started_at, finished_at
	`, input.ID, input.UserID, input.Status, input.ExitCode, input.StdoutPreview, input.StderrPreview,
		input.StdoutTruncated, input.StderrTruncated, input.DurationMS, input.ErrorMessage))
}

func (s *WorkspaceStore) ListCommandRuns(ctx context.Context, userID, workspaceID string, limit int) ([]WorkspaceCommandRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, workspace_id, conversation_id, agent_turn_id, tool_call_id,
			command, args, working_dir, network_policy, status, exit_code, stdout_preview,
			stderr_preview, stdout_truncated, stderr_truncated, duration_ms, error_message,
			metadata, created_at, started_at, finished_at
		FROM workspace_command_runs
		WHERE user_id = $1 AND workspace_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]WorkspaceCommandRun, 0)
	for rows.Next() {
		run, err := scanWorkspaceCommandRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *WorkspaceStore) CreateQuotaSnapshot(ctx context.Context, snapshot WorkspaceQuotaSnapshot) (WorkspaceQuotaSnapshot, error) {
	if len(snapshot.Metadata) == 0 {
		snapshot.Metadata = json.RawMessage(`{}`)
	}
	return scanWorkspaceQuotaSnapshot(s.db.QueryRow(ctx, `
		INSERT INTO workspace_quota_snapshots (
			id, user_id, workspace_id, used_disk_bytes, file_count, command_count, active_process_count, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, workspace_id, used_disk_bytes, file_count, command_count, active_process_count, metadata, created_at
	`, uuid.NewString(), snapshot.UserID, snapshot.WorkspaceID, snapshot.UsedDiskBytes,
		snapshot.FileCount, snapshot.CommandCount, snapshot.ActiveProcessCount, snapshot.Metadata))
}

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
