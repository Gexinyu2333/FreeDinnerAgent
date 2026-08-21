package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduledAgentJob struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	AgentConfigID   *string         `json:"agent_config_id"`
	Title           string          `json:"title"`
	Description     *string         `json:"description"`
	JobType         string          `json:"job_type"`
	ScheduleKind    string          `json:"schedule_kind"`
	CronExpr        *string         `json:"cron_expr"`
	Timezone        string          `json:"timezone"`
	RunAtLocalTime  *string         `json:"run_at_local_time"`
	Weekdays        []int32         `json:"weekdays"`
	PromptTemplate  string          `json:"prompt_template"`
	ContextPolicy   json.RawMessage `json:"context_policy"`
	ToolPolicy      json.RawMessage `json:"tool_policy"`
	DeliveryChannel string          `json:"delivery_channel"`
	Visibility      string          `json:"visibility"`
	Status          string          `json:"status"`
	LastRunAt       *time.Time      `json:"last_run_at"`
	NextRunAt       *time.Time      `json:"next_run_at"`
	FailureCount    int             `json:"failure_count"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type ScheduledAgentJobRun struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	ScheduledJobID string          `json:"scheduled_job_id"`
	ConversationID *string         `json:"conversation_id"`
	AgentTurnID    *string         `json:"agent_turn_id"`
	Status         string          `json:"status"`
	TriggerReason  string          `json:"trigger_reason"`
	InputSnapshot  json.RawMessage `json:"input_snapshot"`
	OutputSummary  *string         `json:"output_summary"`
	ErrorMessage   *string         `json:"error_message"`
	ScheduledFor   time.Time       `json:"scheduled_for"`
	StartedAt      *time.Time      `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at"`
	CreatedAt      time.Time       `json:"created_at"`
}

type ScheduledAgentJobCreate struct {
	UserID          string
	AgentConfigID   *string
	Title           string
	Description     *string
	JobType         string
	ScheduleKind    string
	CronExpr        *string
	Timezone        string
	RunAtLocalTime  *string
	Weekdays        []int32
	PromptTemplate  string
	ContextPolicy   json.RawMessage
	ToolPolicy      json.RawMessage
	DeliveryChannel string
	Visibility      string
	Metadata        json.RawMessage
}

type ScheduledAgentJobRunCreate struct {
	UserID         string
	ScheduledJobID string
	ConversationID *string
	AgentTurnID    *string
	Status         string
	TriggerReason  string
	InputSnapshot  json.RawMessage
	OutputSummary  *string
	ErrorMessage   *string
	ScheduledFor   time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
}

type ScheduledJobStore struct {
	db *pgxpool.Pool
}

func NewScheduledJobStore(db *pgxpool.Pool) *ScheduledJobStore {
	return &ScheduledJobStore{db: db}
}

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
			delivery_channel, visibility, metadata
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10::time, $11, $12, $13, $14,
			$15, $16, $17
		)
		RETURNING id, user_id, agent_config_id, title, description, job_type, schedule_kind, cron_expr,
			timezone, run_at_local_time::text, weekdays, prompt_template, context_policy, tool_policy,
			delivery_channel, visibility, status, last_run_at, next_run_at, failure_count, metadata,
			created_at, updated_at
	`
	return scanScheduledAgentJob(s.db.QueryRow(ctx, query,
		uuid.NewString(),
		input.UserID,
		input.AgentConfigID,
		input.Title,
		input.Description,
		input.JobType,
		input.ScheduleKind,
		input.CronExpr,
		input.Timezone,
		input.RunAtLocalTime,
		input.Weekdays,
		input.PromptTemplate,
		input.ContextPolicy,
		input.ToolPolicy,
		input.DeliveryChannel,
		input.Visibility,
		input.Metadata,
	))
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
		uuid.NewString(),
		input.UserID,
		input.ScheduledJobID,
		input.ConversationID,
		input.AgentTurnID,
		input.Status,
		input.TriggerReason,
		input.InputSnapshot,
		input.OutputSummary,
		input.ErrorMessage,
		input.ScheduledFor,
		input.StartedAt,
		input.FinishedAt,
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

func (s *ScheduledJobStore) MarkJobRan(ctx context.Context, userID, jobID string, ranAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE scheduled_agent_jobs
		SET last_run_at = $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, jobID, userID, ranAt)
	return err
}

func scanScheduledAgentJob(row pgx.Row) (ScheduledAgentJob, error) {
	var job ScheduledAgentJob
	if err := row.Scan(
		&job.ID,
		&job.UserID,
		&job.AgentConfigID,
		&job.Title,
		&job.Description,
		&job.JobType,
		&job.ScheduleKind,
		&job.CronExpr,
		&job.Timezone,
		&job.RunAtLocalTime,
		&job.Weekdays,
		&job.PromptTemplate,
		&job.ContextPolicy,
		&job.ToolPolicy,
		&job.DeliveryChannel,
		&job.Visibility,
		&job.Status,
		&job.LastRunAt,
		&job.NextRunAt,
		&job.FailureCount,
		&job.Metadata,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ScheduledAgentJob{}, ErrNotFound
		}
		return ScheduledAgentJob{}, err
	}
	return job, nil
}

func scanScheduledAgentJobRun(row pgx.Row) (ScheduledAgentJobRun, error) {
	var run ScheduledAgentJobRun
	if err := row.Scan(
		&run.ID,
		&run.UserID,
		&run.ScheduledJobID,
		&run.ConversationID,
		&run.AgentTurnID,
		&run.Status,
		&run.TriggerReason,
		&run.InputSnapshot,
		&run.OutputSummary,
		&run.ErrorMessage,
		&run.ScheduledFor,
		&run.StartedAt,
		&run.FinishedAt,
		&run.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ScheduledAgentJobRun{}, ErrNotFound
		}
		return ScheduledAgentJobRun{}, err
	}
	return run, nil
}
