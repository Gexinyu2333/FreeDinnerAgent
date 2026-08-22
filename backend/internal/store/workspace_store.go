package store

import (
	"encoding/json"
	"time"

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

type WorkspacePolicyUpdate struct {
	UserID              string
	WorkspaceID         string
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
