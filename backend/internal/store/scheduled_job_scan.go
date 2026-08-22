package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

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
