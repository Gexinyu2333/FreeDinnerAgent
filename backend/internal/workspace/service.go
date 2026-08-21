package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
		message := err.Error()
		return s.recordBlockedCommand(ctx, workspace, input, command, args, message)
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

func (s *Service) ListFiles(ctx context.Context, userID, requestedPath string) (ListFilesResult, error) {
	workspace, err := s.activeWorkspace(ctx, userID)
	if err != nil {
		return ListFilesResult{}, err
	}
	fullPath, relativePath, err := s.resolve(workspace, requestedPath)
	if err != nil {
		return ListFilesResult{}, err
	}
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return ListFilesResult{}, err
	}

	items := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return ListFilesResult{}, err
		}
		fileType := "file"
		if entry.IsDir() {
			fileType = "directory"
		}
		entryRelative := filepath.ToSlash(filepath.Join(relativePath, entry.Name()))
		items = append(items, FileEntry{
			Name:         entry.Name(),
			Path:         "/" + entryRelative,
			Type:         fileType,
			SizeBytes:    info.Size(),
			LastModified: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	_ = s.workspaces.Touch(ctx, userID, workspace.ID)
	_, _ = s.workspaces.CreateEvent(ctx, store.WorkspaceEvent{
		UserID:      userID,
		WorkspaceID: workspace.ID,
		EventType:   "file_read",
		ActorType:   "user",
		Metadata:    mustJSON(map[string]any{"path": "/" + relativePath, "operation": "list"}),
	})
	if relativePath == "." {
		relativePath = ""
	}
	return ListFilesResult{Path: "/" + relativePath, Items: items}, nil
}

func (s *Service) ReadFile(ctx context.Context, userID, requestedPath string) (ReadFileResult, error) {
	workspace, err := s.activeWorkspace(ctx, userID)
	if err != nil {
		return ReadFileResult{}, err
	}
	fullPath, relativePath, err := s.resolve(workspace, requestedPath)
	if err != nil {
		return ReadFileResult{}, err
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return ReadFileResult{}, err
	}
	if info.IsDir() {
		return ReadFileResult{}, fs.ErrInvalid
	}
	if info.Size() > workspace.MaxSingleFileBytes {
		return ReadFileResult{}, ErrFileTooLarge
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return ReadFileResult{}, err
	}
	hash := sha256.Sum256(content)
	mimeType := http.DetectContentType(content)
	file, err := s.workspaces.UpsertFile(ctx, store.WorkspaceFileUpsert{
		UserID:       userID,
		WorkspaceID:  workspace.ID,
		RelativePath: relativePath,
		FileType:     "file",
		SizeBytes:    info.Size(),
		ContentHash:  stringPtr(hex.EncodeToString(hash[:])),
		MimeType:     &mimeType,
		CreatedBy:    "user",
	})
	if err != nil {
		return ReadFileResult{}, err
	}
	_, _ = s.workspaces.CreateEvent(ctx, store.WorkspaceEvent{
		UserID:      userID,
		WorkspaceID: workspace.ID,
		EventType:   "file_read",
		ActorType:   "user",
		FileID:      &file.ID,
		Metadata:    mustJSON(map[string]any{"path": "/" + relativePath}),
	})
	_ = s.workspaces.Touch(ctx, userID, workspace.ID)
	return ReadFileResult{
		Path:        "/" + relativePath,
		Content:     string(content),
		SizeBytes:   info.Size(),
		ContentHash: hex.EncodeToString(hash[:]),
		MimeType:    mimeType,
	}, nil
}

func (s *Service) WriteFile(ctx context.Context, input WriteFileInput) (WriteFileResult, error) {
	workspace, err := s.activeWorkspace(ctx, input.UserID)
	if err != nil {
		return WriteFileResult{}, err
	}
	contentBytes := []byte(input.Content)
	if int64(len(contentBytes)) > workspace.MaxSingleFileBytes {
		return WriteFileResult{}, ErrFileTooLarge
	}
	stats, err := scanQuota(workspace.RootPath)
	if err != nil {
		return WriteFileResult{}, err
	}
	if stats.UsedDiskBytes+int64(len(contentBytes)) > workspace.MaxDiskBytes {
		return WriteFileResult{}, ErrQuotaExceeded
	}

	fullPath, relativePath, err := s.resolveForWrite(workspace, input.Path)
	if err != nil {
		return WriteFileResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
		return WriteFileResult{}, err
	}
	if err := os.WriteFile(fullPath, contentBytes, 0o600); err != nil {
		return WriteFileResult{}, err
	}
	hash := sha256.Sum256(contentBytes)
	mimeType := http.DetectContentType(contentBytes)
	file, err := s.workspaces.UpsertFile(ctx, store.WorkspaceFileUpsert{
		UserID:       input.UserID,
		WorkspaceID:  workspace.ID,
		RelativePath: relativePath,
		FileType:     "file",
		SizeBytes:    int64(len(contentBytes)),
		ContentHash:  stringPtr(hex.EncodeToString(hash[:])),
		MimeType:     &mimeType,
		CreatedBy:    "user",
	})
	if err != nil {
		return WriteFileResult{}, err
	}
	_, _ = s.workspaces.CreateEvent(ctx, store.WorkspaceEvent{
		UserID:      input.UserID,
		WorkspaceID: workspace.ID,
		EventType:   "file_updated",
		ActorType:   "user",
		FileID:      &file.ID,
		Metadata:    mustJSON(map[string]any{"path": "/" + relativePath, "size_bytes": len(contentBytes)}),
	})
	_ = s.workspaces.Touch(ctx, input.UserID, workspace.ID)
	return WriteFileResult{File: file}, nil
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

func (s *Service) resolve(workspace store.UserWorkspace, requestedPath string) (string, string, error) {
	fullPath, relativePath, err := s.resolveRaw(workspace, requestedPath)
	if err != nil {
		return "", "", err
	}
	resolved, err := filepath.EvalSymlinks(fullPath)
	if err == nil && !isWithinRoot(workspace.RootPath, resolved) {
		return "", "", ErrPathOutsideRoot
	}
	return fullPath, relativePath, nil
}

func (s *Service) resolveForWrite(workspace store.UserWorkspace, requestedPath string) (string, string, error) {
	fullPath, relativePath, err := s.resolveRaw(workspace, requestedPath)
	if err != nil {
		return "", "", err
	}
	parent := filepath.Dir(fullPath)
	if _, err := os.Stat(parent); err == nil {
		resolvedParent, err := filepath.EvalSymlinks(parent)
		if err == nil && !isWithinRoot(workspace.RootPath, resolvedParent) {
			return "", "", ErrPathOutsideRoot
		}
	}
	return fullPath, relativePath, nil
}

func (s *Service) resolveRaw(workspace store.UserWorkspace, requestedPath string) (string, string, error) {
	root := filepath.Clean(workspace.RootPath)
	rawPath := strings.TrimSpace(requestedPath)
	if hasParentSegment(rawPath) {
		return "", "", ErrPathOutsideRoot
	}
	cleaned := filepath.Clean(rawPath)
	if cleaned == "." || cleaned == "/" {
		cleaned = "files"
	} else {
		cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
		cleaned = filepath.Join("files", cleaned)
	}
	fullPath := filepath.Clean(filepath.Join(root, cleaned))
	if !isWithinRoot(root, fullPath) {
		return "", "", ErrPathOutsideRoot
	}
	relativePath, err := filepath.Rel(filepath.Join(root, "files"), fullPath)
	if err != nil {
		return "", "", err
	}
	relativePath = filepath.ToSlash(relativePath)
	if relativePath == "." {
		relativePath = ""
	}
	if strings.HasPrefix(relativePath, "../") || relativePath == ".." {
		return "", "", ErrPathOutsideRoot
	}
	return fullPath, relativePath, nil
}

func isWithinRoot(root, path string) bool {
	originalRoot := filepath.Clean(root)
	cleanRoot := originalRoot
	cleanPath := filepath.Clean(path)
	if resolvedRoot, err := filepath.EvalSymlinks(originalRoot); err == nil {
		cleanRoot = resolvedRoot
		if relativeToOriginal, relErr := filepath.Rel(originalRoot, cleanPath); relErr == nil &&
			relativeToOriginal != ".." && !strings.HasPrefix(relativeToOriginal, "../") {
			cleanPath = filepath.Join(resolvedRoot, relativeToOriginal)
		}
	}
	if resolvedPath, err := filepath.EvalSymlinks(cleanPath); err == nil {
		cleanPath = resolvedPath
	}
	relative, err := filepath.Rel(cleanRoot, cleanPath)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, "../")
}

func hasParentSegment(path string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

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

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func truncate(value string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value, false
	}
	return value[:maxBytes], true
}

func relativeWorkingDirOrDot(path string) string {
	if path == "" {
		return "."
	}
	return path
}

func mergeMetadata(base map[string]any, extra map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range extra {
		result[key] = value
	}
	return result
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

func scanQuota(root string) (quotaStats, error) {
	var stats quotaStats
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		stats.FileCount++
		stats.UsedDiskBytes += info.Size()
		return nil
	})
	return stats, err
}

func mustJSON(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func stringPtr(value string) *string {
	return &value
}
