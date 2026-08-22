package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"freedinner/backend/internal/store"
)

var ErrInactiveJob = errors.New("scheduled job is not active")

type Service struct {
	jobs          *store.ScheduledJobStore
	conversations *store.ConversationStore
	responder     AgentResponder
}

type AgentResponder interface {
	SendMessage(ctx context.Context, userID, conversationID, content string) (store.SendMessageResult, error)
}

type CreateJobInput struct {
	UserID          string
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

type UpdateJobInput struct {
	UserID          string
	JobID           string
	Title           *string
	Description     **string
	JobType         *string
	ScheduleKind    *string
	CronExpr        **string
	Timezone        *string
	RunAtLocalTime  **string
	Weekdays        []int32
	WeekdaysSet     bool
	PromptTemplate  *string
	ContextPolicy   *json.RawMessage
	ToolPolicy      *json.RawMessage
	DeliveryChannel *string
	Visibility      *string
	Metadata        *json.RawMessage
}

type JobTemplate struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	JobType         string          `json:"job_type"`
	ScheduleKind    string          `json:"schedule_kind"`
	Timezone        string          `json:"timezone"`
	RunAtLocalTime  string          `json:"run_at_local_time"`
	Weekdays        []int32         `json:"weekdays"`
	PromptTemplate  string          `json:"prompt_template"`
	ContextPolicy   json.RawMessage `json:"context_policy"`
	ToolPolicy      json.RawMessage `json:"tool_policy"`
	DeliveryChannel string          `json:"delivery_channel"`
}

type RunNowResult struct {
	Job          store.ScheduledAgentJob    `json:"job"`
	Run          store.ScheduledAgentJobRun `json:"run"`
	Conversation store.Conversation         `json:"conversation"`
	Message      store.Message              `json:"message"`
}

type DueRunResult struct {
	JobID  string        `json:"job_id"`
	RunID  *string       `json:"run_id,omitempty"`
	Status string        `json:"status"`
	Error  *string       `json:"error,omitempty"`
	Result *RunNowResult `json:"result,omitempty"`
}

func NewService(jobs *store.ScheduledJobStore, conversations *store.ConversationStore, responder AgentResponder) *Service {
	return &Service{
		jobs:          jobs,
		conversations: conversations,
		responder:     responder,
	}
}

func (s *Service) CreateJob(ctx context.Context, input CreateJobInput) (store.ScheduledAgentJob, error) {
	nextRunAt := ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   input.ScheduleKind,
		CronExpr:       input.CronExpr,
		Timezone:       input.Timezone,
		RunAtLocalTime: derefString(input.RunAtLocalTime),
		Weekdays:       input.Weekdays,
		Now:            time.Now(),
	})
	return s.jobs.Create(ctx, store.ScheduledAgentJobCreate{
		UserID:          input.UserID,
		Title:           input.Title,
		Description:     input.Description,
		JobType:         input.JobType,
		ScheduleKind:    input.ScheduleKind,
		CronExpr:        input.CronExpr,
		Timezone:        input.Timezone,
		RunAtLocalTime:  input.RunAtLocalTime,
		Weekdays:        input.Weekdays,
		PromptTemplate:  input.PromptTemplate,
		ContextPolicy:   input.ContextPolicy,
		ToolPolicy:      input.ToolPolicy,
		DeliveryChannel: input.DeliveryChannel,
		Visibility:      input.Visibility,
		NextRunAt:       nextRunAt,
		Metadata:        input.Metadata,
	})
}

func (s *Service) ListJobs(ctx context.Context, userID string, status *string, limit int) ([]store.ScheduledAgentJob, error) {
	return s.jobs.List(ctx, userID, status, limit)
}

func (s *Service) ListRuns(ctx context.Context, userID, jobID string, limit int) ([]store.ScheduledAgentJobRun, error) {
	if _, err := s.jobs.FindByID(ctx, userID, jobID); err != nil {
		return nil, err
	}
	return s.jobs.ListRuns(ctx, userID, jobID, limit)
}

func (s *Service) FindRun(ctx context.Context, userID, runID string) (store.ScheduledAgentJobRun, error) {
	return s.jobs.FindRunByID(ctx, userID, runID)
}

func (s *Service) Templates() []JobTemplate {
	return DefaultTemplates()
}

func (s *Service) UpdateJob(ctx context.Context, input UpdateJobInput) (store.ScheduledAgentJob, error) {
	current, err := s.jobs.FindByID(ctx, input.UserID, input.JobID)
	if err != nil {
		return store.ScheduledAgentJob{}, err
	}
	update := scheduledUpdateFromCurrent(input, current)
	update.NextRunAt = ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   update.ScheduleKind,
		CronExpr:       update.CronExpr,
		Timezone:       update.Timezone,
		RunAtLocalTime: derefString(update.RunAtLocalTime),
		Weekdays:       update.Weekdays,
		Now:            time.Now(),
	})
	return s.jobs.Update(ctx, update)
}

func (s *Service) Pause(ctx context.Context, userID, jobID string) (store.ScheduledAgentJob, error) {
	return s.jobs.SetStatus(ctx, userID, jobID, "paused")
}

func (s *Service) Resume(ctx context.Context, userID, jobID string) (store.ScheduledAgentJob, error) {
	job, err := s.jobs.SetStatus(ctx, userID, jobID, "active")
	if err != nil {
		return store.ScheduledAgentJob{}, err
	}
	nextRunAt := ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   job.ScheduleKind,
		CronExpr:       job.CronExpr,
		Timezone:       job.Timezone,
		RunAtLocalTime: derefString(job.RunAtLocalTime),
		Weekdays:       job.Weekdays,
		Now:            time.Now(),
	})
	if err := s.jobs.SetNextRunAt(ctx, userID, jobID, nextRunAt); err != nil {
		return store.ScheduledAgentJob{}, err
	}
	if nextRunAt != nil {
		job.NextRunAt = nextRunAt
	}
	return job, nil
}

func (s *Service) Delete(ctx context.Context, userID, jobID string) (store.ScheduledAgentJob, error) {
	return s.jobs.SetStatus(ctx, userID, jobID, "deleted")
}
