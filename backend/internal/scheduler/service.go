package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

func (s *Service) RunNow(ctx context.Context, userID, jobID string) (RunNowResult, error) {
	job, err := s.jobs.FindByID(ctx, userID, jobID)
	if err != nil {
		return RunNowResult{}, err
	}
	if job.Status != "active" {
		return RunNowResult{}, ErrInactiveJob
	}

	now := time.Now()
	return s.executeScheduledJob(ctx, job, "manual_run", now, now)
}

func (s *Service) RunDue(ctx context.Context, now time.Time, limit int) ([]DueRunResult, error) {
	if now.IsZero() {
		now = time.Now()
	}
	jobs, err := s.jobs.ListDue(ctx, now, limit)
	if err != nil {
		return nil, err
	}

	results := make([]DueRunResult, 0, len(jobs))
	for _, job := range jobs {
		scheduledFor := now
		if job.NextRunAt != nil {
			scheduledFor = *job.NextRunAt
		}
		result, err := s.executeScheduledJob(ctx, job, "schedule_due", scheduledFor, now)
		if err != nil {
			failedRun, runErr := s.recordFailedRun(ctx, job, "schedule_due", scheduledFor, now, err)
			updatedJob, updateErr := s.jobs.IncrementFailureCount(ctx, job.UserID, job.ID)
			errText := err.Error()
			var runID *string
			if runErr != nil {
				errText = errText + "; failed to record failed run: " + runErr.Error()
			} else {
				runID = &failedRun.ID
			}
			if updateErr != nil {
				errText = errText + "; failed to update failure count: " + updateErr.Error()
			}
			results = append(results, DueRunResult{
				JobID:  job.ID,
				RunID:  runID,
				Status: failureStatus(updatedJob),
				Error:  &errText,
			})
			continue
		}
		results = append(results, DueRunResult{
			JobID:  job.ID,
			RunID:  &result.Run.ID,
			Status: result.Run.Status,
			Result: &result,
		})
	}
	return results, nil
}

type ScheduleSpec struct {
	ScheduleKind   string
	CronExpr       *string
	Timezone       string
	RunAtLocalTime string
	Weekdays       []int32
	Now            time.Time
}

func (s *Service) executeScheduledJob(ctx context.Context, job store.ScheduledAgentJob, triggerReason string, scheduledFor time.Time, now time.Time) (RunNowResult, error) {
	conversation, err := s.conversations.Create(ctx, job.UserID, "心跳任务："+job.Title)
	if err != nil {
		return RunNowResult{}, err
	}

	summary := scheduledRunSummary(job.Title, triggerReason)
	var message store.Message
	var agentTurnID *string
	if s.responder != nil {
		result, err := s.responder.SendMessage(ctx, job.UserID, conversation.ID, scheduledPrompt(job, triggerReason, scheduledFor))
		if err != nil {
			return RunNowResult{}, err
		}
		message = result.AssistantMessage
		agentTurnID = &result.TurnID
		summary = message.Content
	} else {
		metadata, _ := json.Marshal(map[string]any{
			"source":           "scheduled_agent_job",
			"scheduled_job_id": job.ID,
			"trigger_reason":   triggerReason,
			"scheduled_for":    scheduledFor.Format(time.RFC3339),
		})
		message, err = s.conversations.CreateAssistantMessage(ctx, job.UserID, conversation.ID, summary, metadata)
		if err != nil {
			return RunNowResult{}, err
		}
	}

	inputSnapshot, _ := json.Marshal(map[string]any{
		"job_id":          job.ID,
		"title":           job.Title,
		"job_type":        job.JobType,
		"schedule_kind":   job.ScheduleKind,
		"prompt_template": job.PromptTemplate,
		"context_policy":  json.RawMessage(job.ContextPolicy),
		"tool_policy":     json.RawMessage(job.ToolPolicy),
		"trigger_reason":  triggerReason,
		"scheduled_for":   scheduledFor.Format(time.RFC3339),
	})
	run, err := s.jobs.CreateRun(ctx, store.ScheduledAgentJobRunCreate{
		UserID:         job.UserID,
		ScheduledJobID: job.ID,
		ConversationID: &conversation.ID,
		AgentTurnID:    agentTurnID,
		Status:         "success",
		TriggerReason:  triggerReason,
		InputSnapshot:  inputSnapshot,
		OutputSummary:  &summary,
		ScheduledFor:   scheduledFor,
		StartedAt:      &now,
		FinishedAt:     &now,
	})
	if err != nil {
		return RunNowResult{}, err
	}
	if err := s.jobs.MarkJobRan(ctx, job.UserID, job.ID, now); err != nil {
		return RunNowResult{}, err
	}
	nextRunAt := ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   job.ScheduleKind,
		CronExpr:       job.CronExpr,
		Timezone:       job.Timezone,
		RunAtLocalTime: derefString(job.RunAtLocalTime),
		Weekdays:       job.Weekdays,
		Now:            now,
	})
	_ = s.jobs.SetNextRunAt(ctx, job.UserID, job.ID, nextRunAt)

	job.LastRunAt = &now
	job.NextRunAt = nextRunAt
	job.FailureCount = 0
	return RunNowResult{
		Job:          job,
		Run:          run,
		Conversation: conversation,
		Message:      message,
	}, nil
}

func (s *Service) recordFailedRun(ctx context.Context, job store.ScheduledAgentJob, triggerReason string, scheduledFor time.Time, now time.Time, runErr error) (store.ScheduledAgentJobRun, error) {
	errorMessage := runErr.Error()
	inputSnapshot, _ := json.Marshal(map[string]any{
		"job_id":          job.ID,
		"title":           job.Title,
		"job_type":        job.JobType,
		"schedule_kind":   job.ScheduleKind,
		"prompt_template": job.PromptTemplate,
		"context_policy":  json.RawMessage(job.ContextPolicy),
		"tool_policy":     json.RawMessage(job.ToolPolicy),
		"trigger_reason":  triggerReason,
		"scheduled_for":   scheduledFor.Format(time.RFC3339),
	})
	return s.jobs.CreateRun(ctx, store.ScheduledAgentJobRunCreate{
		UserID:         job.UserID,
		ScheduledJobID: job.ID,
		Status:         "failed",
		TriggerReason:  triggerReason,
		InputSnapshot:  inputSnapshot,
		ErrorMessage:   &errorMessage,
		ScheduledFor:   scheduledFor,
		StartedAt:      &now,
		FinishedAt:     &now,
	})
}

func scheduledRunSummary(title string, triggerReason string) string {
	if triggerReason == "manual_run" {
		return fmt.Sprintf("已手动触发心跳任务「%s」。", title)
	}
	return fmt.Sprintf("心跳任务「%s」已按计划触发。", title)
}

func scheduledPrompt(job store.ScheduledAgentJob, triggerReason string, scheduledFor time.Time) string {
	var builder strings.Builder
	builder.WriteString("请执行这个心跳任务。\n")
	builder.WriteString("任务标题：")
	builder.WriteString(job.Title)
	builder.WriteString("\n任务类型：")
	builder.WriteString(job.JobType)
	builder.WriteString("\n触发原因：")
	builder.WriteString(triggerReason)
	builder.WriteString("\n计划时间：")
	builder.WriteString(scheduledFor.Format(time.RFC3339))
	if job.Description != nil && strings.TrimSpace(*job.Description) != "" {
		builder.WriteString("\n任务说明：")
		builder.WriteString(strings.TrimSpace(*job.Description))
	}
	builder.WriteString("\n用户配置的任务提示词：\n")
	builder.WriteString(job.PromptTemplate)
	return builder.String()
}

func failureStatus(job store.ScheduledAgentJob) string {
	if job.Status == "paused" {
		return "paused_after_failures"
	}
	return "failed"
}

func ComputeNextRunAt(spec ScheduleSpec) *time.Time {
	kind := NormalizeScheduleKind(spec.ScheduleKind)
	if kind == "cron" {
		return nextCronRun(spec)
	}
	location, err := time.LoadLocation(defaultString(spec.Timezone, "Asia/Shanghai"))
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	now := spec.Now
	if now.IsZero() {
		now = time.Now()
	}
	localNow := now.In(location)
	hour, minute, second := parseRunAtLocalTime(spec.RunAtLocalTime)
	candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, second, 0, location)

	switch kind {
	case "once":
		if candidate.After(localNow) {
			return timePtr(candidate)
		}
		return nil
	case "daily":
		if !candidate.After(localNow) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		return timePtr(candidate)
	case "weekly":
		return nextWeeklyRun(localNow, candidate, spec.Weekdays)
	case "monthly":
		if !candidate.After(localNow) {
			candidate = candidate.AddDate(0, 1, 0)
		}
		return timePtr(candidate)
	default:
		return nil
	}
}

func nextCronRun(spec ScheduleSpec) *time.Time {
	if spec.CronExpr == nil || strings.TrimSpace(*spec.CronExpr) == "" {
		return nil
	}
	location, err := time.LoadLocation(defaultString(spec.Timezone, "Asia/Shanghai"))
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	now := spec.Now
	if now.IsZero() {
		now = time.Now()
	}
	localNow := now.In(location).Truncate(time.Minute)
	fields := strings.Fields(strings.TrimSpace(*spec.CronExpr))
	if len(fields) != 5 {
		return nil
	}
	minutes, ok := parseCronField(fields[0], 0, 59)
	if !ok {
		return nil
	}
	hours, ok := parseCronField(fields[1], 0, 23)
	if !ok {
		return nil
	}
	days, ok := parseCronField(fields[2], 1, 31)
	if !ok {
		return nil
	}
	months, ok := parseCronField(fields[3], 1, 12)
	if !ok {
		return nil
	}
	weekdays, ok := parseCronField(fields[4], 0, 7)
	if !ok {
		return nil
	}
	for offset := 1; offset <= 366*24*60; offset++ {
		candidate := localNow.Add(time.Duration(offset) * time.Minute)
		weekday := int(candidate.Weekday())
		if weekdays[7] && weekday == 0 {
			weekday = 7
		}
		if minutes[candidate.Minute()] && hours[candidate.Hour()] &&
			days[candidate.Day()] && months[int(candidate.Month())] &&
			(weekdays[int(candidate.Weekday())] || weekdays[weekday]) {
			return timePtr(candidate)
		}
	}
	return nil
}

func parseCronField(value string, min int, max int) (map[int]bool, bool) {
	result := map[int]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		step := 1
		if strings.Contains(part, "/") {
			pieces := strings.Split(part, "/")
			if len(pieces) != 2 {
				return nil, false
			}
			part = pieces[0]
			parsedStep, err := strconv.Atoi(pieces[1])
			if err != nil || parsedStep <= 0 {
				return nil, false
			}
			step = parsedStep
		}
		start, end := min, max
		if part != "*" {
			if strings.Contains(part, "-") {
				pieces := strings.Split(part, "-")
				if len(pieces) != 2 {
					return nil, false
				}
				var err error
				start, err = strconv.Atoi(pieces[0])
				if err != nil {
					return nil, false
				}
				end, err = strconv.Atoi(pieces[1])
				if err != nil {
					return nil, false
				}
			} else {
				parsed, err := strconv.Atoi(part)
				if err != nil {
					return nil, false
				}
				start, end = parsed, parsed
			}
		}
		if start < min || end > max || start > end {
			return nil, false
		}
		for current := start; current <= end; current += step {
			result[current] = true
		}
	}
	return result, len(result) > 0
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

func DefaultTemplates() []JobTemplate {
	return []JobTemplate{
		{
			ID:              "daily_brief",
			Title:           "每日简报",
			Description:     "工作日早上汇总任务、记忆和重要事项。",
			JobType:         "daily_brief",
			ScheduleKind:    "weekly",
			Timezone:        "Asia/Shanghai",
			RunAtLocalTime:  "08:00:00",
			Weekdays:        []int32{1, 2, 3, 4, 5},
			PromptTemplate:  "请根据我的任务、记忆和最近对话生成今天的个人简报。",
			ContextPolicy:   json.RawMessage(`{"include_memory":true,"include_tasks":true,"include_calendar":false,"include_email":false,"max_context_tokens":6000}`),
			ToolPolicy:      json.RawMessage(`{"allow_tools":true,"allowed_tools":["list_tasks","search_memory"],"requires_approval_for_write":true}`),
			DeliveryChannel: "in_app",
		},
		{
			ID:              "weekly_review",
			Title:           "每周回顾",
			Description:     "每周五整理完成事项、阻塞问题和下周计划。",
			JobType:         "weekly_review",
			ScheduleKind:    "weekly",
			Timezone:        "Asia/Shanghai",
			RunAtLocalTime:  "16:00:00",
			Weekdays:        []int32{5},
			PromptTemplate:  "请总结我本周的任务、对话和关键进展，并提炼下周计划。",
			ContextPolicy:   json.RawMessage(`{"include_memory":true,"include_tasks":true,"include_calendar":false,"include_email":false,"max_context_tokens":8000}`),
			ToolPolicy:      json.RawMessage(`{"allow_tools":true,"allowed_tools":["list_tasks","search_memory","save_profile_memory"],"requires_approval_for_write":true}`),
			DeliveryChannel: "in_app",
		},
		{
			ID:              "follow_up_monitor",
			Title:           "跟进监控",
			Description:     "工作日上午检查未跟进事项，只生成建议和草稿。",
			JobType:         "follow_up_monitor",
			ScheduleKind:    "weekly",
			Timezone:        "Asia/Shanghai",
			RunAtLocalTime:  "09:00:00",
			Weekdays:        []int32{1, 2, 3, 4, 5},
			PromptTemplate:  "请检查最近任务和对话中是否存在需要我关注的未跟进事项，给出建议但不要自动创建大量任务。",
			ContextPolicy:   json.RawMessage(`{"include_memory":true,"include_tasks":true,"include_calendar":false,"include_email":false,"max_context_tokens":6000}`),
			ToolPolicy:      json.RawMessage(`{"allow_tools":true,"allowed_tools":["list_tasks","search_memory","create_task"],"requires_approval_for_write":true}`),
			DeliveryChannel: "in_app",
		},
	}
}

func scheduledUpdateFromCurrent(input UpdateJobInput, current store.ScheduledAgentJob) store.ScheduledAgentJobUpdate {
	update := store.ScheduledAgentJobUpdate{
		UserID:          input.UserID,
		JobID:           input.JobID,
		Title:           current.Title,
		Description:     current.Description,
		JobType:         current.JobType,
		ScheduleKind:    current.ScheduleKind,
		CronExpr:        current.CronExpr,
		Timezone:        current.Timezone,
		RunAtLocalTime:  current.RunAtLocalTime,
		Weekdays:        current.Weekdays,
		PromptTemplate:  current.PromptTemplate,
		ContextPolicy:   current.ContextPolicy,
		ToolPolicy:      current.ToolPolicy,
		DeliveryChannel: current.DeliveryChannel,
		Visibility:      current.Visibility,
		Metadata:        current.Metadata,
	}
	if input.Title != nil && strings.TrimSpace(*input.Title) != "" {
		update.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		update.Description = trimOptional(*input.Description)
	}
	if input.JobType != nil {
		update.JobType = NormalizeJobType(*input.JobType)
	}
	if input.ScheduleKind != nil {
		update.ScheduleKind = NormalizeScheduleKind(*input.ScheduleKind)
	}
	if input.CronExpr != nil {
		update.CronExpr = trimOptional(*input.CronExpr)
	}
	if input.Timezone != nil && strings.TrimSpace(*input.Timezone) != "" {
		update.Timezone = strings.TrimSpace(*input.Timezone)
	}
	if input.RunAtLocalTime != nil {
		update.RunAtLocalTime = trimOptional(*input.RunAtLocalTime)
	}
	if input.WeekdaysSet {
		update.Weekdays = input.Weekdays
	}
	if input.PromptTemplate != nil && strings.TrimSpace(*input.PromptTemplate) != "" {
		update.PromptTemplate = strings.TrimSpace(*input.PromptTemplate)
	}
	if input.ContextPolicy != nil {
		update.ContextPolicy = *input.ContextPolicy
	}
	if input.ToolPolicy != nil {
		update.ToolPolicy = *input.ToolPolicy
	}
	if input.DeliveryChannel != nil {
		update.DeliveryChannel = NormalizeDeliveryChannel(*input.DeliveryChannel)
	}
	if input.Visibility != nil {
		update.Visibility = NormalizeVisibility(*input.Visibility)
	}
	if input.Metadata != nil {
		update.Metadata = *input.Metadata
	}
	return update
}

func nextWeeklyRun(localNow time.Time, todayAtTime time.Time, weekdays []int32) *time.Time {
	if len(weekdays) == 0 {
		weekdays = []int32{isoWeekday(localNow.Weekday())}
	}
	for dayOffset := 0; dayOffset <= 7; dayOffset++ {
		candidate := todayAtTime.AddDate(0, 0, dayOffset)
		if !containsWeekday(weekdays, isoWeekday(candidate.Weekday())) {
			continue
		}
		if candidate.After(localNow) {
			return timePtr(candidate)
		}
	}
	return nil
}

func parseRunAtLocalTime(value string) (int, int, int) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"15:04:05", "15:04"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Hour(), parsed.Minute(), parsed.Second()
		}
	}
	return 9, 0, 0
}

func isoWeekday(value time.Weekday) int32 {
	if value == time.Sunday {
		return 7
	}
	return int32(value)
}

func containsWeekday(values []int32, target int32) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func timePtr(value time.Time) *time.Time {
	return &value
}
