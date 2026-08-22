package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *ScheduledJobStore) FindRunByID(ctx context.Context, userID, runID string) (ScheduledAgentJobRun, error) {
	return scanScheduledAgentJobRun(s.db.QueryRow(ctx, `
		SELECT id, user_id, scheduled_job_id, conversation_id, agent_turn_id, status, trigger_reason,
			input_snapshot, output_summary, error_message, scheduled_for, started_at, finished_at, created_at
		FROM scheduled_agent_job_runs
		WHERE id = $1 AND user_id = $2
	`, runID, userID))
}

func (s *ScheduledJobStore) CreateRun(ctx context.Context, input ScheduledAgentJobRunCreate) (ScheduledAgentJobRun, error) {
	if len(input.InputSnapshot) == 0 {
		input.InputSnapshot = json.RawMessage(`{}`)
	}

	query := `
		INSERT INTO scheduled_agent_job_runs (
			id, user_id, scheduled_job_id, conversation_id, agent_turn_id, status, trigger_reason,
			input_snapshot, output_summary, error_message, scheduled_for, started_at, finished_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, user_id, scheduled_job_id, conversation_id, agent_turn_id, status, trigger_reason,
			input_snapshot, output_summary, error_message, scheduled_for, started_at, finished_at, created_at
	`
	return scanScheduledAgentJobRun(s.db.QueryRow(ctx, query,
		uuid.NewString(), input.UserID, input.ScheduledJobID, input.ConversationID,
		input.AgentTurnID, input.Status, input.TriggerReason, input.InputSnapshot,
		input.OutputSummary, input.ErrorMessage, input.ScheduledFor, input.StartedAt, input.FinishedAt,
	))
}

func (s *ScheduledJobStore) ListRuns(ctx context.Context, userID, jobID string, limit int) ([]ScheduledAgentJobRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, scheduled_job_id, conversation_id, agent_turn_id, status, trigger_reason,
			input_snapshot, output_summary, error_message, scheduled_for, started_at, finished_at, created_at
		FROM scheduled_agent_job_runs
		WHERE user_id = $1 AND scheduled_job_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]ScheduledAgentJobRun, 0)
	for rows.Next() {
		run, err := scanScheduledAgentJobRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}
