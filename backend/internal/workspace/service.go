package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"freedinner/backend/internal/store"
)

var (
	ErrWorkspaceDisabled = errors.New("workspace is not active")
	ErrPathOutsideRoot   = errors.New("path escapes workspace root")
	ErrFileTooLarge      = errors.New("file exceeds workspace single-file limit")
	ErrQuotaExceeded     = errors.New("workspace quota exceeded")
	ErrCommandBlocked    = errors.New("command is not allowed by workspace policy")
)

type Service struct {
	workspaces *store.WorkspaceStore
	root       string
	runners    RunnerOptions
}

type EnableInput struct {
	UserID              string
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

type UpdatePolicyInput struct {
	UserID              string
	SandboxType         *string
	NetworkPolicy       *string
	NetworkAllowlist    []string
	NetworkAllowlistSet bool
	MaxDiskBytes        *int64
	MaxFileCount        *int
	MaxSingleFileBytes  *int64
	MaxCommandSeconds   *int
	MaxStdoutBytes      *int
	MaxStderrBytes      *int
	CPULimit            *string
	CPULimitSet         bool
	MemoryLimitBytes    *int64
	MemoryLimitSet      bool
	IdleAfterSeconds    *int
	DestroyAfterSeconds *int
}

type DestroyResult struct {
	Workspace    store.UserWorkspace `json:"workspace"`
	FilesRemoved bool                `json:"files_removed"`
}

type WorkspaceStatus struct {
	Workspace store.UserWorkspace          `json:"workspace"`
	Quota     store.WorkspaceQuotaSnapshot `json:"quota"`
}

type FileEntry struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Type         string `json:"type"`
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"`
}

type ListFilesResult struct {
	Path  string      `json:"path"`
	Items []FileEntry `json:"items"`
}

type ReadFileResult struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	SizeBytes   int64  `json:"size_bytes"`
	ContentHash string `json:"content_hash"`
	MimeType    string `json:"mime_type"`
}

type WriteFileInput struct {
	UserID  string
	Path    string
	Content string
}

type WriteFileResult struct {
	File store.WorkspaceFile `json:"file"`
}

type RunCommandInput struct {
	UserID         string
	ConversationID *string
	AgentTurnID    *string
	ToolCallID     *string
	Command        string
	Args           []string
	WorkingDir     string
	TimeoutSeconds int
}

type RunCommandResult struct {
	Run store.WorkspaceCommandRun `json:"run"`
}

type quotaStats struct {
	UsedDiskBytes int64
	FileCount     int
}

func (s *Service) UpdatePolicy(ctx context.Context, input UpdatePolicyInput) (WorkspaceStatus, error) {
	workspace, err := s.activeWorkspace(ctx, input.UserID)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	updatedInput := policyUpdateFromWorkspace(input, workspace)
	updated, err := s.workspaces.UpdatePolicy(ctx, updatedInput)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	_, _ = s.workspaces.CreateEvent(ctx, store.WorkspaceEvent{
		UserID:      input.UserID,
		WorkspaceID: updated.ID,
		EventType:   "policy_changed",
		ActorType:   "user",
		Metadata:    mustJSON(map[string]any{"sandbox_type": updated.SandboxType, "network_policy": updated.NetworkPolicy}),
	})
	return s.GetStatus(ctx, input.UserID)
}

func (s *Service) Destroy(ctx context.Context, userID string, removeFiles bool) (DestroyResult, error) {
	workspace, err := s.activeWorkspace(ctx, userID)
	if err != nil {
		return DestroyResult{}, err
	}
	if removeFiles && !isWithinRoot(s.root, workspace.RootPath) {
		return DestroyResult{}, ErrPathOutsideRoot
	}
	destroyed, err := s.workspaces.MarkDestroyed(ctx, userID, workspace.ID)
	if err != nil {
		return DestroyResult{}, err
	}
	filesRemoved := false
	if removeFiles {
		if err := os.RemoveAll(workspace.RootPath); err != nil {
			return DestroyResult{}, err
		}
		filesRemoved = true
	}
	_, _ = s.workspaces.CreateEvent(ctx, store.WorkspaceEvent{
		UserID:      userID,
		WorkspaceID: workspace.ID,
		EventType:   "destroyed",
		ActorType:   "user",
		Metadata:    mustJSON(map[string]any{"files_removed": filesRemoved}),
	})
	return DestroyResult{Workspace: destroyed, FilesRemoved: filesRemoved}, nil
}

func NewService(workspaces *store.WorkspaceStore, root string, runners RunnerOptions) *Service {
	return &Service{
		workspaces: workspaces,
		root:       filepath.Clean(root),
		runners:    runners,
	}
}

func (s *Service) GetStatus(ctx context.Context, userID string) (WorkspaceStatus, error) {
	workspace, err := s.workspaces.FindByUserID(ctx, userID)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	stats, err := scanQuota(workspace.RootPath)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	snapshot, err := s.workspaces.CreateQuotaSnapshot(ctx, store.WorkspaceQuotaSnapshot{
		UserID:        userID,
		WorkspaceID:   workspace.ID,
		UsedDiskBytes: stats.UsedDiskBytes,
		FileCount:     stats.FileCount,
	})
	if err != nil {
		return WorkspaceStatus{}, err
	}
	return WorkspaceStatus{Workspace: workspace, Quota: snapshot}, nil
}

func (s *Service) Enable(ctx context.Context, input EnableInput) (WorkspaceStatus, error) {
	input = normalizeEnableInput(input)
	rootPath, err := filepath.Abs(filepath.Join(s.root, input.UserID))
	if err != nil {
		return WorkspaceStatus{}, err
	}
	if err := os.MkdirAll(filepath.Join(rootPath, "files"), 0o700); err != nil {
		return WorkspaceStatus{}, err
	}
	for _, dir := range []string{"artifacts", "tmp", "logs"} {
		if err := os.MkdirAll(filepath.Join(rootPath, dir), 0o700); err != nil {
			return WorkspaceStatus{}, err
		}
	}

	workspace, err := s.workspaces.UpsertActive(ctx, store.WorkspaceUpsert{
		UserID:              input.UserID,
		RootPath:            rootPath,
		SandboxType:         input.SandboxType,
		NetworkPolicy:       input.NetworkPolicy,
		NetworkAllowlist:    input.NetworkAllowlist,
		MaxDiskBytes:        input.MaxDiskBytes,
		MaxFileCount:        input.MaxFileCount,
		MaxSingleFileBytes:  input.MaxSingleFileBytes,
		MaxCommandSeconds:   input.MaxCommandSeconds,
		MaxStdoutBytes:      input.MaxStdoutBytes,
		MaxStderrBytes:      input.MaxStderrBytes,
		CPULimit:            input.CPULimit,
		MemoryLimitBytes:    input.MemoryLimitBytes,
		IdleAfterSeconds:    input.IdleAfterSeconds,
		DestroyAfterSeconds: input.DestroyAfterSeconds,
	})
	if err != nil {
		return WorkspaceStatus{}, err
	}
	_, _ = s.workspaces.CreateEvent(ctx, store.WorkspaceEvent{
		UserID:      input.UserID,
		WorkspaceID: workspace.ID,
		EventType:   "enabled",
		ActorType:   "user",
	})
	return s.GetStatus(ctx, input.UserID)
}

func (s *Service) activeWorkspace(ctx context.Context, userID string) (store.UserWorkspace, error) {
	workspace, err := s.workspaces.FindByUserID(ctx, userID)
	if err != nil {
		return store.UserWorkspace{}, err
	}
	if workspace.Status != "active" && workspace.Status != "idle" {
		return store.UserWorkspace{}, ErrWorkspaceDisabled
	}
	return workspace, nil
}
