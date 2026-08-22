package store

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

func (s *MemoryStore) MatchSkillDisclosures(ctx context.Context, userID, query, loadMode string, limit int) ([]SkillDisclosure, error) {
	if limit <= 0 || limit > 10 {
		limit = 5
	}
	if loadMode == "" || loadMode == "auto" {
		loadMode = "light"
	}
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		terms = []string{strings.ToLower(strings.TrimSpace(query))}
	}
	rows, err := s.db.Query(ctx, `
		WITH matched_skills AS (
			SELECT s.id
			FROM skills s
			WHERE s.status = 'active'
			  AND (s.user_id = $1 OR s.visibility = 'public')
			  AND (
			  	EXISTS (
			  		SELECT 1 FROM unnest(s.trigger_keywords) AS keyword
			  		WHERE lower($2) LIKE '%' || lower(keyword) || '%'
			  		   OR EXISTS (SELECT 1 FROM unnest($3::text[]) AS term WHERE lower(keyword) LIKE '%' || term || '%')
			  	)
			  	OR lower(s.name) LIKE '%' || lower($2) || '%'
			  	OR lower(s.description) LIKE '%' || lower($2) || '%'
			  )
			ORDER BY s.success_count DESC, s.use_count DESC, s.updated_at DESC
			LIMIT $4
		)
		SELECT s.id, s.name, s.description, s.permission_level,
		       sv.id, sv.version, d.disclosure_level, d.title, d.content, d.token_count, d.created_at
		FROM matched_skills ms
		JOIN skills s ON s.id = ms.id
		JOIN LATERAL (
			SELECT id, version
			FROM skill_versions
			WHERE skill_id = s.id
			ORDER BY version DESC
			LIMIT 1
		) sv ON TRUE
		JOIN skill_disclosure_sections d ON d.skill_version_id = sv.id AND d.disclosure_level = $5
		ORDER BY s.success_count DESC, s.use_count DESC, d.created_at DESC
	`, userID, strings.ToLower(strings.TrimSpace(query)), terms, limit, loadMode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SkillDisclosure, 0)
	for rows.Next() {
		var item SkillDisclosure
		if err := rows.Scan(&item.SkillID, &item.SkillName, &item.Description, &item.PermissionLevel,
			&item.VersionID, &item.Version, &item.DisclosureLevel, &item.Title, &item.Content,
			&item.TokenCount, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) CreateSkillFromEpisode(ctx context.Context, input SkillDistillationInput) (SkillDistillationResult, error) {
	if len(input.Keywords) == 0 {
		input.Keywords = normalizedTags([]string{input.Name})
	}
	if strings.TrimSpace(input.ReactSteps) == "" {
		input.ReactSteps = "1. 理解用户目标。\n2. 复用历史成功处理方式。\n3. 必要时调用工具。\n4. 输出结构化结果。"
	}
	metadata, _ := json.Marshal(map[string]any{"source": "episode_distillation", "episode_id": input.EpisodeID})
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return SkillDistillationResult{}, err
	}
	defer tx.Rollback(ctx)

	skill, err := scanSkill(tx.QueryRow(ctx, `
		INSERT INTO skills (
			id, user_id, name, description, trigger_keywords, scenario, visibility,
			permission_level, success_count, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'private', 'normal', 1, $7)
		ON CONFLICT (user_id, name) DO UPDATE SET
			description = EXCLUDED.description,
			trigger_keywords = EXCLUDED.trigger_keywords,
			success_count = skills.success_count + 1,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, user_id, name, description, trigger_keywords, scenario, visibility,
		          permission_level, status, use_count, success_count, failure_count, metadata,
		          created_at, updated_at
	`, uuid.NewString(), input.UserID, input.Name, input.Description, normalizedTags(input.Keywords),
		input.Description, metadata))
	if err != nil {
		return SkillDistillationResult{}, err
	}

	var nextVersion int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM skill_versions WHERE skill_id = $1`, skill.ID).Scan(&nextVersion); err != nil {
		return SkillDistillationResult{}, err
	}
	version, err := scanSkillVersion(tx.QueryRow(ctx, `
		INSERT INTO skill_versions (
			id, skill_id, version, react_steps, tool_sequence, output_template, fallback_strategy, created_from_episode_id
		)
		VALUES ($1, $2, $3, $4, '[]'::jsonb, $5, $6, $7)
		RETURNING id, skill_id, version, react_steps, tool_sequence, output_template,
		          fallback_strategy, created_from_episode_id, created_at
	`, uuid.NewString(), skill.ID, nextVersion, input.ReactSteps, input.OutputTemplate,
		stringPtr("如果上下文不足，先向用户确认关键约束。"), input.EpisodeID))
	if err != nil {
		return SkillDistillationResult{}, err
	}
	disclosures := []struct {
		level   string
		title   string
		content string
	}{
		{"light", "适用场景", input.Description},
		{"standard", "推荐流程", input.ReactSteps},
		{"full", "完整技能", input.ReactSteps + "\n\n输出模板：\n" + derefString(input.OutputTemplate)},
	}
	for _, disclosure := range disclosures {
		_, err := tx.Exec(ctx, `
			INSERT INTO skill_disclosure_sections (
				id, skill_version_id, disclosure_level, title, content, token_count
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (skill_version_id, disclosure_level) DO UPDATE SET
				title = EXCLUDED.title,
				content = EXCLUDED.content,
				token_count = EXCLUDED.token_count
		`, uuid.NewString(), version.ID, disclosure.level, disclosure.title, disclosure.content, estimateStoreTokens(disclosure.content))
		if err != nil {
			return SkillDistillationResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SkillDistillationResult{}, err
	}
	return SkillDistillationResult{Skill: skill, Version: version}, nil
}
