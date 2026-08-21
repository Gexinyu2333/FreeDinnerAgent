package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"freedinner/backend/internal/store"
	workspacesvc "freedinner/backend/internal/workspace"

	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct {
	workspaces *workspacesvc.Service
}

func NewWorkspaceHandler(workspaces *workspacesvc.Service) *WorkspaceHandler {
	return &WorkspaceHandler{workspaces: workspaces}
}

type enableWorkspaceRequest struct {
	SandboxType         string   `json:"sandbox_type"`
	MaxDiskBytes        int64    `json:"max_disk_bytes"`
	MaxFileCount        int      `json:"max_file_count"`
	MaxSingleFileBytes  int64    `json:"max_single_file_bytes"`
	MaxCommandSeconds   int      `json:"max_command_seconds"`
	MaxStdoutBytes      int      `json:"max_stdout_bytes"`
	MaxStderrBytes      int      `json:"max_stderr_bytes"`
	CPULimit            *string  `json:"cpu_limit"`
	MemoryLimitBytes    *int64   `json:"memory_limit_bytes"`
	NetworkPolicy       string   `json:"network_policy"`
	NetworkAllowlist    []string `json:"network_allowlist"`
	IdleAfterSeconds    int      `json:"idle_after_seconds"`
	DestroyAfterSeconds int      `json:"destroy_after_seconds"`
}

type writeWorkspaceFileRequest struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content"`
}

type runWorkspaceCommandRequest struct {
	Command        string   `json:"command" binding:"required"`
	Args           []string `json:"args"`
	WorkingDir     string   `json:"working_dir"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

func (h *WorkspaceHandler) Status(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	status, err := h.workspaces.GetStatus(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "WORKSPACE_NOT_FOUND", "workspace is not enabled")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load workspace")
		return
	}
	OK(c, status)
}

func (h *WorkspaceHandler) Enable(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req enableWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	status, err := h.workspaces.Enable(c.Request.Context(), workspacesvc.EnableInput{
		UserID:              userID,
		SandboxType:         req.SandboxType,
		NetworkPolicy:       req.NetworkPolicy,
		NetworkAllowlist:    req.NetworkAllowlist,
		MaxDiskBytes:        req.MaxDiskBytes,
		MaxFileCount:        req.MaxFileCount,
		MaxSingleFileBytes:  req.MaxSingleFileBytes,
		MaxCommandSeconds:   req.MaxCommandSeconds,
		MaxStdoutBytes:      req.MaxStdoutBytes,
		MaxStderrBytes:      req.MaxStderrBytes,
		CPULimit:            req.CPULimit,
		MemoryLimitBytes:    req.MemoryLimitBytes,
		IdleAfterSeconds:    req.IdleAfterSeconds,
		DestroyAfterSeconds: req.DestroyAfterSeconds,
	})
	if err != nil {
		log.Printf("enable workspace failed: %v", err)
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to enable workspace")
		return
	}
	OK(c, status)
}

func (h *WorkspaceHandler) ListFiles(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	path := c.DefaultQuery("path", "/")
	result, err := h.workspaces.ListFiles(c.Request.Context(), userID, path)
	writeWorkspaceResult(c, result, err)
}

func (h *WorkspaceHandler) ReadFile(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	result, err := h.workspaces.ReadFile(c.Request.Context(), userID, c.Query("path"))
	writeWorkspaceResult(c, result, err)
}

func (h *WorkspaceHandler) WriteFile(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req writeWorkspaceFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	result, err := h.workspaces.WriteFile(c.Request.Context(), workspacesvc.WriteFileInput{
		UserID:  userID,
		Path:    req.Path,
		Content: req.Content,
	})
	writeWorkspaceResult(c, result, err)
}

func (h *WorkspaceHandler) RunCommand(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req runWorkspaceCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	result, err := h.workspaces.RunCommand(c.Request.Context(), workspacesvc.RunCommandInput{
		UserID:         userID,
		Command:        req.Command,
		Args:           req.Args,
		WorkingDir:     req.WorkingDir,
		TimeoutSeconds: req.TimeoutSeconds,
	})
	writeWorkspaceResult(c, result, err)
}

func (h *WorkspaceHandler) CommandRuns(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	limit := parseIntQuery(c, "limit", 50)
	runs, err := h.workspaces.ListCommandRuns(c.Request.Context(), userID, limit)
	writeWorkspaceResult(c, gin.H{"runs": runs}, err)
}

func writeWorkspaceResult(c *gin.Context, data any, err error) {
	if err == nil {
		OK(c, data)
		return
	}
	switch {
	case errors.Is(err, store.ErrNotFound):
		Error(c, http.StatusNotFound, "WORKSPACE_NOT_FOUND", "workspace is not enabled")
	case errors.Is(err, workspacesvc.ErrWorkspaceDisabled):
		Error(c, http.StatusConflict, "WORKSPACE_DISABLED", "workspace is not active")
	case errors.Is(err, workspacesvc.ErrPathOutsideRoot):
		Error(c, http.StatusBadRequest, "INVALID_PATH", "path escapes workspace root")
	case errors.Is(err, workspacesvc.ErrFileTooLarge):
		Error(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "file exceeds workspace limit")
	case errors.Is(err, workspacesvc.ErrQuotaExceeded):
		Error(c, http.StatusConflict, "QUOTA_EXCEEDED", "workspace quota exceeded")
	case errors.Is(err, workspacesvc.ErrCommandBlocked):
		Error(c, http.StatusForbidden, "COMMAND_BLOCKED", "command is not allowed by workspace policy")
	default:
		log.Printf("workspace operation failed: %v", err)
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "workspace operation failed")
	}
}

func parseIntQuery(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
