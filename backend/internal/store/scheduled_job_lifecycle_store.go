package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func (s *ScheduledJobStore) Create(ctx context.Context, input ScheduledAgentJobCreate) (ScheduledAgentJob, error) {
	if len(input.ContextPolicy) == 0 {
		input.ContextPolicy = json.RawMessage(`{"include_memory":true,"include_tasks":true,"include_calendar":false,"include_email":false,"max_context_tokens":6000}`)
	}
	if len(input.ToolPolicy) == 0 {
		input.ToolPolicy = json.RawMessage(`{"allow_tools":true,"requires_approval_for_write":true}`)
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}

	query := `
		INSERT INTO scheduled_agent_jobs (
			id, user_id, agent_config_id, title, description, job_type, schedule_kind, cron_expr,
			timezone, run_at_local_time, weekdays, prompt_template, context_policy, tool_policy,
			delivery_channel, visibility, next_run_at, metadata
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10::time, $11, $12, $13, $14,
			$15, $16, $17, $18
		)
		RETURNING id, user_id, agent_config_id, title, description, job_type, schedule_kind, cron_expr,
			timezone, run_at_local_time::text, weekdays, prompt_template, context_policy, tool_policy,
			delivery_channel, visibility, status, last_run_at, next_run_at, failure_count, metadata,
			created_at, updated_at
	`
	return scanScheduledAgentJob(s.db.QueryRow(ctx, query,
		uuid.NewString(), input.UserID, input.AgentConfigID, input.Title, input.Description,
		input.JobType, input.ScheduleKind, input.CronExpr, input.Timezone, input.RunAtLocalTime,
		input.Weekdays, input.PromptTemplate, input.ContextPolicy, input.ToolPolicy,
		input.DeliveryChannel, input.Visibility, input.NextRunAt, input.Metadata,
	))
}

func (s *ScheduledJobStore) Update(ctx context.Context, input ScheduledAgentJobUpdate) (ScheduledAgentJob, error) {
	if len(input.ContextPolicy) == 0 {
		input.ContextPolicy = json.RawMessage(`{"include_memory":true,"include_tasks":true,"include_calendar":false,"include_email":false,"max_context_tokens":6000}`)
	}
	if len(input.ToolPolicy) == 0 {
		input.ToolPolicy = json.RawMessage(`{"allow_tools":true,"requires_approval_for_write":true}`)
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	return scanScheduledAgentJob(s.db.QueryRow(ctx, `
		UPDATE scheduled_agent_jobs
		SET title = $3,
			description = $4,
			job_type = $5,
			schedule_kind = $6,
			cron_expr = $7,
			timezone = $8,
			run_at_local_time = $9::time,
			weekdays = $10,
			prompt_template = $11,
			context_policy = $12,
			tool_policy = $13,
			delivery_channel = $14,
			visibility = $15,
			next_run_at = $16,
			metadata = $17,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
		RETURNING id, user_id, agent_config_id, title, description, job_type, schedule_kind, cron_expr,
			timezone, run_at_local_time::text, weekdays, prompt_template, context_policy, tool_policy,
			delivery_channel, visibility, status, last_run_at, next_run_at, failure_count, metadata,
			created_at, updated_at
	`, input.JobID, input.UserID, input.Title, input.Description, input.JobType, input.ScheduleKind,
		input.CronExpr, input.Timezone, input.RunAtLocalTime, input.Weekdays, input.PromptTemplate,
		input.ContextPolicy, input.ToolPolicy, input.DeliveryChannel, input.Visibility, input.NextRunAt, input.Metadata))
}

func (s *ScheduledJobStore) List(ctx context.Context, userID string, status *string, limit int) ([]ScheduledAgentJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, agent_config_id, title, description, job_type, schedule_kind, cron_expr,
			timezone, run_at_local_time::text, weekdays, prompt_template, context_policy, tool_policy,
			delivery_channel, visibility, status, last_run_at, next_run_at, failure_count, metadata,
			created_at, updated_at
		FROM scheduled_agent_jobs
		WHERE user_id = $1
			AND status <> 'deleted'
			AND ($2::text IS NULL OR status = $2)
		ORDER BY updated_at DESC, created_at DESC
		LIMIT $3
	`, userID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]ScheduledAgentJob, 0)
	for rows.Next() {
		job, err := scanScheduledAgentJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *ScheduledJobStore) ListDue(ctx context.Context, now time.Time, limit int) ([]ScheduledAgentJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, agent_config_id, title, description, job_type, schedule_kind, cron_expr,
			timezone, run_at_local_time::text, weekdays, prompt_template, context_policy, tool_policy,
			delivery_channel, visibility, status, last_run_at, next_run_at, failure_count, metadata,
			created_at, updated_at
		FROM scheduled_agent_jobs
		WHERE status = 'active'
			AND next_run_at IS NOT NULL
			AND next_run_at <= $1
		ORDER BY next_run_at ASC, updated_at ASC
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]ScheduledAgentJob, 0)
	for rows.Next() {
		job, err := scanScheduledAgentJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *ScheduledJobStore) FindByID(ctx context.Context, userID, jobID string) (ScheduledAgentJob, error) {
	query := `
		SELECT id, user_id, agent_config_id, title, description, job_type, schedule_kind, cron_expr,
			timezone, run_at_local_time::text, weekdays, prompt_template, context_policy, tool_policy,
			delivery_channel, visibility, status, last_run_at, next_run_at, failure_count, metadata,
			created_at, updated_at
		FROM scheduled_agent_jobs
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
	`
	return scanScheduledAgentJob(s.db.QueryRow(ctx, query, jobID, userID))
}

func (s *ScheduledJobStore) MarkJobRan(ctx context.Context, userID, jobID string, ranAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE scheduled_agent_jobs
		SET last_run_at = $3,
			failure_count = 0,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, jobID, userID, ranAt)
	return err
}

func (s *ScheduledJobStore) IncrementFailureCount(ctx context.Context, userID, jobID string) (ScheduledAgentJob, error) {
	return scanScheduledAgentJob(s.db.QueryRow(ctx, `
		UPDATE scheduled_agent_jobs
		SET failure_count = failure_count + 1,
			status = CASE WHEN failure_count + 1 >= 5 THEN 'paused' ELSE status END,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
		RETURNING id, user_id, agent_config_id, title, description, job_type, schedule_kind, cron_expr,
			timezone, run_at_local_time::text, weekdays, prompt_template, context_policy, tool_policy,
			delivery_channel, visibility, status, last_run_at, next_run_at, failure_count, metadata,
			created_at, updated_at
	`, jobID, userID))
}

func (s *ScheduledJobStore) SetStatus(ctx context.Context, userID, jobID, status string) (ScheduledAgentJob, error) {
	return scanScheduledAgentJob(s.db.QueryRow(ctx, `
		UPDATE scheduled_agent_jobs
		SET status = $3,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
		RETURNING id, user_id, agent_config_id, title, description, job_type, schedule_kind, cron_expr,
			timezone, run_at_local_time::text, weekdays, prompt_template, context_policy, tool_policy,
			delivery_channel, visibility, status, last_run_at, next_run_at, failure_count, metadata,
			created_at, updated_at
	`, jobID, userID, status))
}

func (s *ScheduledJobStore) SetNextRunAt(ctx context.Context, userID, jobID string, nextRunAt *time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE scheduled_agent_jobs
		SET next_run_at = $3,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, jobID, userID, nextRunAt)
	return err
}
