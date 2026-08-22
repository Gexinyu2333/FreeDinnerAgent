package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"freedinner/backend/internal/store"
)

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

func (s *Service) executeScheduledJob(ctx context.Context, job store.ScheduledAgentJob, triggerReason string, scheduledFor time.Time, now time.Time) (RunNowResult, error) {
	conversation, err := s.conversations.Create(ctx, job.UserID, "心跳任务："+job.Title)
	if err != nil {
		return RunNowResult{}, err
	}

	summary := scheduledRunSummary(job.Title, triggerReason)
	var message store.Message
	var agentTurnID *string
	status := "success"
	var errorMessage *string
	if s.responder != nil {
		result, err := s.responder.SendMessage(ctx, job.UserID, conversation.ID, scheduledPrompt(job, triggerReason, scheduledFor))
		if err != nil {
			status = "failed"
			messageText := err.Error()
			errorMessage = &messageText
			summary = scheduledRunFailedSummary(job.Title, triggerReason, messageText)
			metadata, _ := json.Marshal(map[string]any{
				"source":           "scheduled_agent_job",
				"scheduled_job_id": job.ID,
				"trigger_reason":   triggerReason,
				"scheduled_for":    scheduledFor.Format(time.RFC3339),
				"error":            messageText,
			})
			message, err = s.conversations.CreateAssistantMessage(ctx, job.UserID, conversation.ID, summary, metadata)
			if err != nil {
				return RunNowResult{}, err
			}
		} else {
			message = result.AssistantMessage
			agentTurnID = &result.TurnID
			summary = message.Content
		}
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
		Status:         status,
		TriggerReason:  triggerReason,
		InputSnapshot:  inputSnapshot,
		OutputSummary:  &summary,
		ErrorMessage:   errorMessage,
		ScheduledFor:   scheduledFor,
		StartedAt:      &now,
		FinishedAt:     &now,
	})
	if err != nil {
		return RunNowResult{}, err
	}
	if status == "success" {
		if err := s.jobs.MarkJobRan(ctx, job.UserID, job.ID, now); err != nil {
			return RunNowResult{}, err
		}
		job.LastRunAt = &now
		job.FailureCount = 0
	} else {
		updatedJob, err := s.jobs.IncrementFailureCount(ctx, job.UserID, job.ID)
		if err != nil {
			return RunNowResult{}, err
		}
		job.Status = updatedJob.Status
		job.FailureCount = updatedJob.FailureCount
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

	job.NextRunAt = nextRunAt
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

func scheduledRunFailedSummary(title string, triggerReason string, message string) string {
	if triggerReason == "manual_run" {
		return fmt.Sprintf("已手动触发心跳任务「%s」，但执行失败：%s", title, message)
	}
	return fmt.Sprintf("心跳任务「%s」已按计划触发，但执行失败：%s", title, message)
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
