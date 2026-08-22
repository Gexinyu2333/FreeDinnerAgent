package store

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

func (s *MemoryStore) CreateEpisode(ctx context.Context, input EpisodeCreate) (Episode, error) {
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	if input.Status == "" {
		input.Status = "success"
	}
	if input.Importance <= 0 {
		input.Importance = 3
	}
	if input.TokenCount <= 0 {
		input.TokenCount = estimateStoreTokens(input.UserInput + input.AgentSummary + input.FinalResponse)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Episode{}, err
	}
	defer tx.Rollback(ctx)
	episode := Episode{}
	err = tx.QueryRow(ctx, `
		INSERT INTO episodes (
			id, user_id, conversation_id, user_message_id, assistant_message_id,
			user_input, agent_summary, final_response, task_type, status, importance,
			token_count, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, user_id, conversation_id, user_message_id, assistant_message_id,
		          user_input, agent_summary, final_response, task_type, status, importance,
		          token_count, metadata, created_at, updated_at
	`, uuid.NewString(), input.UserID, input.ConversationID, input.UserMessageID, input.AssistantMessageID,
		input.UserInput, input.AgentSummary, input.FinalResponse, input.TaskType, input.Status,
		input.Importance, input.TokenCount, input.Metadata).Scan(
		&episode.ID, &episode.UserID, &episode.ConversationID, &episode.UserMessageID, &episode.AssistantMessageID,
		&episode.UserInput, &episode.AgentSummary, &episode.FinalResponse, &episode.TaskType, &episode.Status,
		&episode.Importance, &episode.TokenCount, &episode.Metadata, &episode.CreatedAt, &episode.UpdatedAt,
	)
	if err != nil {
		return Episode{}, err
	}
	for _, tag := range normalizedTags(input.Tags) {
		if _, err := tx.Exec(ctx, `INSERT INTO episode_tags (episode_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`, episode.ID, tag); err != nil {
			return Episode{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Episode{}, err
	}
	return episode, nil
}

func (s *MemoryStore) SearchEpisodes(ctx context.Context, userID, query string, limit int) ([]EpisodeMatch, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 && strings.TrimSpace(query) != "" {
		terms = []string{strings.ToLower(strings.TrimSpace(query))}
	}
	rows, err := s.db.Query(ctx, `
		WITH scored AS (
			SELECT e.id,
			       (
			           (CASE WHEN e.status = 'success' THEN 0.20 ELSE 0 END) +
			           (e.importance::float / 10.0) +
			           (CASE WHEN EXISTS (
			               SELECT 1 FROM unnest($2::text[]) AS term
			               WHERE lower(e.user_input) LIKE '%' || term || '%'
			                  OR lower(e.agent_summary) LIKE '%' || term || '%'
			                  OR lower(e.final_response) LIKE '%' || term || '%'
			                  OR lower(coalesce(e.task_type, '')) LIKE '%' || term || '%'
			           ) THEN 0.45 ELSE 0 END) +
			           (CASE WHEN EXISTS (
			               SELECT 1
			               FROM episode_tags et
			               JOIN unnest($2::text[]) AS term ON lower(et.tag) LIKE '%' || term || '%'
			               WHERE et.episode_id = e.id
			           ) THEN 0.30 ELSE 0 END)
			       ) AS score
			FROM episodes e
			WHERE e.user_id = $1
		)
		SELECT e.id, e.user_id, e.conversation_id, e.user_message_id, e.assistant_message_id,
		       e.user_input, e.agent_summary, e.final_response, e.task_type, e.status, e.importance,
		       e.token_count, e.metadata, e.created_at, e.updated_at,
		       COALESCE(array_agg(et.tag ORDER BY et.tag) FILTER (WHERE et.tag IS NOT NULL), '{}') AS tags,
		       scored.score
		FROM scored
		JOIN episodes e ON e.id = scored.id
		LEFT JOIN episode_tags et ON et.episode_id = e.id
		WHERE scored.score > 0
		GROUP BY e.id, scored.score
		ORDER BY scored.score DESC, e.updated_at DESC
		LIMIT $3
	`, userID, terms, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches := make([]EpisodeMatch, 0)
	for rows.Next() {
		var match EpisodeMatch
		if err := rows.Scan(&match.ID, &match.UserID, &match.ConversationID, &match.UserMessageID,
			&match.AssistantMessageID, &match.UserInput, &match.AgentSummary, &match.FinalResponse,
			&match.TaskType, &match.Status, &match.Importance, &match.TokenCount, &match.Metadata,
			&match.CreatedAt, &match.UpdatedAt, &match.Tags, &match.Score); err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}
	return matches, rows.Err()
}
