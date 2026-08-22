package store

import (
	"encoding/json"
	"time"

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
	NextRunAt       *time.Time
	Metadata        json.RawMessage
}

type ScheduledAgentJobUpdate struct {
	UserID          string
	JobID           string
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
	NextRunAt       *time.Time
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
