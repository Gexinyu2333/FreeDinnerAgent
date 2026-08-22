package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

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
