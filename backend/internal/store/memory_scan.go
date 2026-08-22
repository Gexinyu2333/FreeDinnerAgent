package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

func scanWorkingMemory(row pgx.Row) (WorkingMemory, error) {
	var memory WorkingMemory
	if err := row.Scan(
		&memory.ID,
		&memory.UserID,
		&memory.ConversationID,
		&memory.MemoryKey,
		&memory.MemoryValue,
		&memory.Category,
		&memory.TokenCount,
		&memory.ExpiresAt,
		&memory.CreatedAt,
		&memory.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkingMemory{}, ErrNotFound
		}
		return WorkingMemory{}, err
	}
	return memory, nil
}

func scanProfileMemory(row pgx.Row) (ProfileMemory, error) {
	var memory ProfileMemory
	if err := row.Scan(
		&memory.ID,
		&memory.UserID,
		&memory.MemoryType,
		&memory.Scope,
		&memory.Title,
		&memory.Content,
		&memory.Evidence,
		&memory.SourceMessageID,
		&memory.Confidence,
		&memory.Importance,
		&memory.Status,
		&memory.Metadata,
		&memory.CreatedAt,
		&memory.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProfileMemory{}, ErrNotFound
		}
		return ProfileMemory{}, err
	}
	return memory, nil
}

func scanSkill(row pgx.Row) (Skill, error) {
	var skill Skill
	if err := row.Scan(&skill.ID, &skill.UserID, &skill.Name, &skill.Description,
		&skill.TriggerKeywords, &skill.Scenario, &skill.Visibility, &skill.PermissionLevel,
		&skill.Status, &skill.UseCount, &skill.SuccessCount, &skill.FailureCount,
		&skill.Metadata, &skill.CreatedAt, &skill.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Skill{}, ErrNotFound
		}
		return Skill{}, err
	}
	return skill, nil
}

func scanSkillVersion(row pgx.Row) (SkillVersion, error) {
	var version SkillVersion
	if err := row.Scan(&version.ID, &version.SkillID, &version.Version, &version.ReactSteps,
		&version.ToolSequence, &version.OutputTemplate, &version.FallbackStrategy,
		&version.CreatedFromEpisodeID, &version.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SkillVersion{}, ErrNotFound
		}
		return SkillVersion{}, err
	}
	return version, nil
}

func scanCuratorJob(row pgx.Row) (CuratorJob, error) {
	var job CuratorJob
	if err := row.Scan(&job.ID, &job.UserID, &job.JobType, &job.Payload, &job.Status,
		&job.ErrorMessage, &job.CreatedAt, &job.StartedAt, &job.FinishedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CuratorJob{}, ErrNotFound
		}
		return CuratorJob{}, err
	}
	return job, nil
}

func scanDreamingSession(row pgx.Row) (DreamingSession, error) {
	var session DreamingSession
	if err := row.Scan(&session.ID, &session.UserID, &session.TriggerType, &session.Scope,
		&session.Status, &session.InputSummary, &session.OutputSummary, &session.StartedAt,
		&session.FinishedAt, &session.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DreamingSession{}, ErrNotFound
		}
		return DreamingSession{}, err
	}
	return session, nil
}

func scanDreamingInsight(row pgx.Row) (DreamingInsight, error) {
	var insight DreamingInsight
	if err := row.Scan(&insight.ID, &insight.DreamingSessionID, &insight.UserID, &insight.InsightType,
		&insight.SourceLayer, &insight.SourceRefIDs, &insight.TargetLayer, &insight.TargetRefID,
		&insight.Content, &insight.Confidence, &insight.Status, &insight.CreatedAt, &insight.AppliedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DreamingInsight{}, ErrNotFound
		}
		return DreamingInsight{}, err
	}
	return insight, nil
}
