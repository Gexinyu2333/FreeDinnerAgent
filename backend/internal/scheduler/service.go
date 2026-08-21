package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"freedinner/backend/internal/store"
)

var ErrInactiveJob = errors.New("scheduled job is not active")

type Service struct {
	jobs          *store.ScheduledJobStore
	conversations *store.ConversationStore
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

type RunNowResult struct {
	Job          store.ScheduledAgentJob    `json:"job"`
	Run          store.ScheduledAgentJobRun `json:"run"`
	Conversation store.Conversation         `json:"conversation"`
	Message      store.Message              `json:"message"`
}

func NewService(jobs *store.ScheduledJobStore, conversations *store.ConversationStore) *Service {
	return &Service{
		jobs:          jobs,
		conversations: conversations,
	}
}

func (s *Service) CreateJob(ctx context.Context, input CreateJobInput) (store.ScheduledAgentJob, error) {
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

func (s *Service) RunNow(ctx context.Context, userID, jobID string) (RunNowResult, error) {
	job, err := s.jobs.FindByID(ctx, userID, jobID)
	if err != nil {
		return RunNowResult{}, err
	}
	if job.Status != "active" {
		return RunNowResult{}, ErrInactiveJob
	}

	now := time.Now()
	conversation, err := s.conversations.Create(ctx, userID, "心跳任务："+job.Title)
	if err != nil {
		return RunNowResult{}, err
	}

	summary := fmt.Sprintf("已手动触发心跳任务「%s」。当前 MVP 已记录本次运行，并创建了对应会话；后续 Agent Loop 接入后，这里会进入完整 ReAct 执行流程。", job.Title)
	metadata, _ := json.Marshal(map[string]any{
		"source":           "scheduled_agent_job",
		"scheduled_job_id": job.ID,
		"trigger_reason":   "manual_run",
	})
	message, err := s.conversations.CreateAssistantMessage(ctx, userID, conversation.ID, summary, metadata)
	if err != nil {
		return RunNowResult{}, err
	}

	inputSnapshot, _ := json.Marshal(map[string]any{
		"job_id":          job.ID,
		"title":           job.Title,
		"job_type":        job.JobType,
		"schedule_kind":   job.ScheduleKind,
		"prompt_template": job.PromptTemplate,
		"context_policy":  json.RawMessage(job.ContextPolicy),
		"tool_policy":     json.RawMessage(job.ToolPolicy),
		"manual_run_at":   now.Format(time.RFC3339),
	})
	run, err := s.jobs.CreateRun(ctx, store.ScheduledAgentJobRunCreate{
		UserID:         userID,
		ScheduledJobID: job.ID,
		ConversationID: &conversation.ID,
		Status:         "success",
		TriggerReason:  "manual_run",
		InputSnapshot:  inputSnapshot,
		OutputSummary:  &summary,
		ScheduledFor:   now,
		StartedAt:      &now,
		FinishedAt:     &now,
	})
	if err != nil {
		return RunNowResult{}, err
	}
	if err := s.jobs.MarkJobRan(ctx, userID, job.ID, now); err != nil {
		return RunNowResult{}, err
	}

	return RunNowResult{
		Job:          job,
		Run:          run,
		Conversation: conversation,
		Message:      message,
	}, nil
}

func NormalizeJobType(value string) string {
	switch strings.TrimSpace(value) {
	case "daily_brief", "weekly_review", "follow_up_monitor", "reminder", "content_digest", "social_assist", "custom":
		return strings.TrimSpace(value)
	default:
		return "custom"
	}
}

func NormalizeScheduleKind(value string) string {
	switch strings.TrimSpace(value) {
	case "once", "daily", "weekly", "monthly", "cron":
		return strings.TrimSpace(value)
	default:
		return "once"
	}
}

func NormalizeDeliveryChannel(value string) string {
	switch strings.TrimSpace(value) {
	case "email", "webhook":
		return strings.TrimSpace(value)
	default:
		return "in_app"
	}
}

func NormalizeVisibility(value string) string {
	if strings.TrimSpace(value) == "public_template" {
		return "public_template"
	}
	return "private"
}
