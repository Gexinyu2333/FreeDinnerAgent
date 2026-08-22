package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

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

func (s *WorkspaceStore) UpdatePolicy(ctx context.Context, input WorkspacePolicyUpdate) (UserWorkspace, error) {
	return scanUserWorkspace(s.db.QueryRow(ctx, `
		UPDATE user_workspaces
		SET sandbox_type = $3,
			network_policy = $4,
			network_allowlist = $5,
			max_disk_bytes = $6,
			max_file_count = $7,
			max_single_file_bytes = $8,
			max_command_seconds = $9,
			max_stdout_bytes = $10,
			max_stderr_bytes = $11,
			cpu_limit = $12,
			memory_limit_bytes = $13,
			idle_after_seconds = $14,
			destroy_after_seconds = $15,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status <> 'destroyed'
		RETURNING id, user_id, status, root_path, sandbox_type, network_policy, network_allowlist,
			max_disk_bytes, max_file_count, max_single_file_bytes, max_command_seconds,
			max_stdout_bytes, max_stderr_bytes, cpu_limit, memory_limit_bytes, last_active_at,
			idle_after_seconds, destroy_after_seconds, metadata, created_at, updated_at
	`, input.WorkspaceID, input.UserID, input.SandboxType, input.NetworkPolicy, input.NetworkAllowlist,
		input.MaxDiskBytes, input.MaxFileCount, input.MaxSingleFileBytes, input.MaxCommandSeconds,
		input.MaxStdoutBytes, input.MaxStderrBytes, input.CPULimit, input.MemoryLimitBytes,
		input.IdleAfterSeconds, input.DestroyAfterSeconds))
}

func (s *WorkspaceStore) MarkDestroyed(ctx context.Context, userID, workspaceID string) (UserWorkspace, error) {
	return scanUserWorkspace(s.db.QueryRow(ctx, `
		UPDATE user_workspaces
		SET status = 'destroyed',
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status <> 'destroyed'
		RETURNING id, user_id, status, root_path, sandbox_type, network_policy, network_allowlist,
			max_disk_bytes, max_file_count, max_single_file_bytes, max_command_seconds,
			max_stdout_bytes, max_stderr_bytes, cpu_limit, memory_limit_bytes, last_active_at,
			idle_after_seconds, destroy_after_seconds, metadata, created_at, updated_at
	`, workspaceID, userID))
}
