package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

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
