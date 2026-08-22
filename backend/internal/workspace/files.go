package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"freedinner/backend/internal/store"
)

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
