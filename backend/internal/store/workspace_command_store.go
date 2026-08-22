package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

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
