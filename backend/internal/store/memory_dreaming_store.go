package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *MemoryStore) CreateDreamingSession(ctx context.Context, input DreamingSessionCreate) (DreamingSession, error) {
	if input.TriggerType == "" {
		input.TriggerType = "manual"
	}
	if input.Scope == "" {
		input.Scope = "user"
	}
	return scanDreamingSession(s.db.QueryRow(ctx, `
		INSERT INTO dreaming_sessions (id, user_id, trigger_type, scope, status, input_summary, started_at)
		VALUES ($1, $2, $3, $4, 'running', $5, NOW())
		RETURNING id, user_id, trigger_type, scope, status, input_summary, output_summary,
		          started_at, finished_at, created_at
	`, uuid.NewString(), input.UserID, input.TriggerType, input.Scope, input.InputSummary))
}

func (s *MemoryStore) FinishDreamingSession(ctx context.Context, sessionID, userID, status string, outputSummary *string) (DreamingSession, error) {
	return scanDreamingSession(s.db.QueryRow(ctx, `
		UPDATE dreaming_sessions
		SET status = $3, output_summary = $4, finished_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, trigger_type, scope, status, input_summary, output_summary,
		          started_at, finished_at, created_at
	`, sessionID, userID, status, outputSummary))
}

func (s *MemoryStore) CreateDreamingInsight(ctx context.Context, input DreamingInsightCreate) (DreamingInsight, error) {
	if input.Confidence <= 0 {
		input.Confidence = 0.75
	}
	return scanDreamingInsight(s.db.QueryRow(ctx, `
		INSERT INTO dreaming_insights (
			id, dreaming_session_id, user_id, insight_type, source_layer, source_ref_ids,
			target_layer, target_ref_id, content, confidence
		)
		VALUES ($1, $2, $3, $4, $5, $6::uuid[], $7, $8, $9, $10)
		RETURNING id, dreaming_session_id, user_id, insight_type, source_layer, source_ref_ids,
		          target_layer, target_ref_id, content, confidence, status, created_at, applied_at
	`, uuid.NewString(), input.DreamingSessionID, input.UserID, input.InsightType, input.SourceLayer,
		input.SourceRefIDs, input.TargetLayer, input.TargetRefID, input.Content, input.Confidence))
}

func (s *MemoryStore) ListDreamingInsights(ctx context.Context, userID string, status *string, limit int) ([]DreamingInsight, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, dreaming_session_id, user_id, insight_type, source_layer, source_ref_ids,
		       target_layer, target_ref_id, content, confidence, status, created_at, applied_at
		FROM dreaming_insights
		WHERE user_id = $1 AND ($2::text IS NULL OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var insights []DreamingInsight
	for rows.Next() {
		insight, err := scanDreamingInsight(rows)
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}
	return insights, rows.Err()
}

func (s *MemoryStore) FindDreamingInsight(ctx context.Context, userID, insightID string) (DreamingInsight, error) {
	return scanDreamingInsight(s.db.QueryRow(ctx, `
		SELECT id, dreaming_session_id, user_id, insight_type, source_layer, source_ref_ids,
		       target_layer, target_ref_id, content, confidence, status, created_at, applied_at
		FROM dreaming_insights
		WHERE id = $1 AND user_id = $2
	`, insightID, userID))
}

func (s *MemoryStore) SetDreamingInsightStatus(ctx context.Context, userID, insightID, status string) (DreamingInsight, error) {
	return scanDreamingInsight(s.db.QueryRow(ctx, `
		UPDATE dreaming_insights
		SET status = $3,
		    applied_at = CASE WHEN $3 = 'applied' THEN NOW() ELSE applied_at END
		WHERE id = $1 AND user_id = $2 AND status = 'proposed'
		RETURNING id, dreaming_session_id, user_id, insight_type, source_layer, source_ref_ids,
		          target_layer, target_ref_id, content, confidence, status, created_at, applied_at
	`, insightID, userID, status))
}
