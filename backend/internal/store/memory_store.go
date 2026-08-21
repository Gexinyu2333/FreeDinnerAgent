package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MemoryTypeDefinition struct {
	MemoryType      string    `json:"memory_type"`
	DisplayName     string    `json:"display_name"`
	Description     string    `json:"description"`
	ExtractionHint  string    `json:"extraction_hint"`
	RetrievalWeight float64   `json:"retrieval_weight"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ProfileMemory struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	MemoryType      string          `json:"memory_type"`
	Scope           string          `json:"scope"`
	Title           string          `json:"title"`
	Content         string          `json:"content"`
	Evidence        *string         `json:"evidence"`
	SourceMessageID *string         `json:"source_message_id"`
	Confidence      float64         `json:"confidence"`
	Importance      int             `json:"importance"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type WorkingMemory struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	ConversationID string     `json:"conversation_id"`
	MemoryKey      string     `json:"memory_key"`
	MemoryValue    string     `json:"memory_value"`
	Category       string     `json:"category"`
	TokenCount     int        `json:"token_count"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Episode struct {
	ID                 string          `json:"id"`
	UserID             string          `json:"user_id"`
	ConversationID     string          `json:"conversation_id"`
	UserMessageID      *string         `json:"user_message_id"`
	AssistantMessageID *string         `json:"assistant_message_id"`
	UserInput          string          `json:"user_input"`
	AgentSummary       string          `json:"agent_summary"`
	FinalResponse      string          `json:"final_response"`
	TaskType           *string         `json:"task_type"`
	Status             string          `json:"status"`
	Importance         int             `json:"importance"`
	TokenCount         int             `json:"token_count"`
	Metadata           json.RawMessage `json:"metadata"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type SkillDisclosure struct {
	SkillID         string    `json:"skill_id"`
	SkillName       string    `json:"skill_name"`
	Description     string    `json:"description"`
	PermissionLevel string    `json:"permission_level"`
	VersionID       string    `json:"version_id"`
	Version         int       `json:"version"`
	DisclosureLevel string    `json:"disclosure_level"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	TokenCount      int       `json:"token_count"`
	CreatedAt       time.Time `json:"created_at"`
}

type DreamingSession struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	TriggerType   string     `json:"trigger_type"`
	Scope         string     `json:"scope"`
	Status        string     `json:"status"`
	InputSummary  *string    `json:"input_summary"`
	OutputSummary *string    `json:"output_summary"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type DreamingInsight struct {
	ID                string     `json:"id"`
	DreamingSessionID string     `json:"dreaming_session_id"`
	UserID            string     `json:"user_id"`
	InsightType       string     `json:"insight_type"`
	SourceLayer       string     `json:"source_layer"`
	SourceRefIDs      []string   `json:"source_ref_ids"`
	TargetLayer       *string    `json:"target_layer"`
	TargetRefID       *string    `json:"target_ref_id"`
	Content           string     `json:"content"`
	Confidence        float64    `json:"confidence"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	AppliedAt         *time.Time `json:"applied_at"`
}

type CuratorJob struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	JobType      string          `json:"job_type"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"`
	ErrorMessage *string         `json:"error_message"`
	CreatedAt    time.Time       `json:"created_at"`
	StartedAt    *time.Time      `json:"started_at"`
	FinishedAt   *time.Time      `json:"finished_at"`
}

type ProfileMemoryCreate struct {
	UserID          string
	MemoryType      string
	Scope           string
	Title           string
	Content         string
	Evidence        *string
	SourceMessageID *string
	Confidence      float64
	Importance      int
	Metadata        json.RawMessage
}

type EpisodeCreate struct {
	UserID             string
	ConversationID     string
	UserMessageID      *string
	AssistantMessageID *string
	UserInput          string
	AgentSummary       string
	FinalResponse      string
	TaskType           *string
	Status             string
	Importance         int
	TokenCount         int
	Metadata           json.RawMessage
	Tags               []string
}

type CuratorJobCreate struct {
	UserID  string
	JobType string
	Payload json.RawMessage
}

type DreamingSessionCreate struct {
	UserID       string
	TriggerType  string
	Scope        string
	InputSummary *string
}

type DreamingInsightCreate struct {
	DreamingSessionID string
	UserID            string
	InsightType       string
	SourceLayer       string
	SourceRefIDs      []string
	TargetLayer       *string
	TargetRefID       *string
	Content           string
	Confidence        float64
}

type WorkingMemoryUpsert struct {
	UserID         string
	ConversationID string
	MemoryKey      string
	MemoryValue    string
	Category       string
	TokenCount     int
	ExpiresAt      *time.Time
}

type MemoryRetrievalLogCreate struct {
	UserID         string
	ConversationID string
	MessageID      *string
	MemoryLayer    string
	MemoryRefID    string
	Score          *float64
	TokenCount     int
	LoadMode       string
}

type MemoryStore struct {
	db *pgxpool.Pool
}

func NewMemoryStore(db *pgxpool.Pool) *MemoryStore {
	return &MemoryStore{db: db}
}

func (s *MemoryStore) ListTypes(ctx context.Context) ([]MemoryTypeDefinition, error) {
	rows, err := s.db.Query(ctx, `
		SELECT memory_type, display_name, description, extraction_hint, retrieval_weight, is_active, created_at, updated_at
		FROM memory_type_definitions
		WHERE is_active = TRUE
		ORDER BY memory_type ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	types := make([]MemoryTypeDefinition, 0)
	for rows.Next() {
		var definition MemoryTypeDefinition
		if err := rows.Scan(
			&definition.MemoryType,
			&definition.DisplayName,
			&definition.Description,
			&definition.ExtractionHint,
			&definition.RetrievalWeight,
			&definition.IsActive,
			&definition.CreatedAt,
			&definition.UpdatedAt,
		); err != nil {
			return nil, err
		}
		types = append(types, definition)
	}
	return types, rows.Err()
}

func (s *MemoryStore) UpsertWorkingMemory(ctx context.Context, input WorkingMemoryUpsert) (WorkingMemory, error) {
	if input.Category == "" {
		input.Category = "temporary_context"
	}
	if input.TokenCount <= 0 {
		input.TokenCount = estimateStoreTokens(input.MemoryValue)
	}
	return scanWorkingMemory(s.db.QueryRow(ctx, `
		INSERT INTO session_working_memories (
			id, user_id, conversation_id, memory_key, memory_value, category, token_count, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (conversation_id, memory_key)
		DO UPDATE SET
			memory_value = EXCLUDED.memory_value,
			category = EXCLUDED.category,
			token_count = EXCLUDED.token_count,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
		RETURNING id, user_id, conversation_id, memory_key, memory_value, category, token_count, expires_at, created_at, updated_at
	`, uuid.NewString(), input.UserID, input.ConversationID, input.MemoryKey, input.MemoryValue,
		input.Category, input.TokenCount, input.ExpiresAt))
}

func (s *MemoryStore) ListWorkingMemories(ctx context.Context, userID, conversationID string, limit int) ([]WorkingMemory, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, conversation_id, memory_key, memory_value, category, token_count, expires_at, created_at, updated_at
		FROM session_working_memories
		WHERE user_id = $1
		  AND conversation_id = $2
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY updated_at DESC
		LIMIT $3
	`, userID, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memories := make([]WorkingMemory, 0)
	for rows.Next() {
		memory, err := scanWorkingMemory(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, memory)
	}
	return memories, rows.Err()
}

func (s *MemoryStore) CreateProfileMemory(ctx context.Context, input ProfileMemoryCreate) (ProfileMemory, error) {
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	if input.Scope == "" {
		input.Scope = "global"
	}
	if input.Confidence <= 0 {
		input.Confidence = 0.8
	}
	if input.Importance <= 0 {
		input.Importance = 3
	}

	query := `
		INSERT INTO profile_memories (
			id, user_id, memory_type, scope, title, content, evidence,
			source_message_id, confidence, importance, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, user_id, memory_type, scope, title, content, evidence, source_message_id,
		          confidence, importance, status, metadata, created_at, updated_at
	`
	return scanProfileMemory(s.db.QueryRow(ctx, query, uuid.NewString(), input.UserID, input.MemoryType, input.Scope,
		input.Title, input.Content, input.Evidence, input.SourceMessageID, input.Confidence, input.Importance, input.Metadata))
}

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

func (s *MemoryStore) CreateCuratorJob(ctx context.Context, input CuratorJobCreate) (CuratorJob, error) {
	if len(input.Payload) == 0 {
		input.Payload = json.RawMessage(`{}`)
	}
	return scanCuratorJob(s.db.QueryRow(ctx, `
		INSERT INTO curator_jobs (id, user_id, job_type, payload)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, job_type, payload, status, error_message, created_at, started_at, finished_at
	`, uuid.NewString(), input.UserID, input.JobType, input.Payload))
}

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

func (s *MemoryStore) CreateRetrievalLog(ctx context.Context, input MemoryRetrievalLogCreate) error {
	if input.LoadMode == "" {
		input.LoadMode = "standard"
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO memory_retrieval_logs (
			id, user_id, conversation_id, message_id, memory_layer, memory_ref_id, score, token_count, load_mode
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, uuid.NewString(), input.UserID, input.ConversationID, input.MessageID, input.MemoryLayer,
		input.MemoryRefID, input.Score, input.TokenCount, input.LoadMode)
	return err
}

func (s *MemoryStore) ListProfileMemories(ctx context.Context, userID string, memoryType *string) ([]ProfileMemory, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, memory_type, scope, title, content, evidence, source_message_id,
		       confidence, importance, status, metadata, created_at, updated_at
		FROM profile_memories
		WHERE user_id = $1 AND status = 'active' AND ($2::text IS NULL OR memory_type = $2)
		ORDER BY importance DESC, updated_at DESC
	`, userID, memoryType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProfileMemories(rows)
}

func (s *MemoryStore) SearchProfileMemories(ctx context.Context, userID, query string, limit int) ([]ProfileMemory, error) {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		terms = []string{strings.ToLower(strings.TrimSpace(query))}
	}
	rows, err := s.db.Query(ctx, `
		SELECT pm.id, pm.user_id, pm.memory_type, pm.scope, pm.title, pm.content, pm.evidence, pm.source_message_id,
		       pm.confidence, pm.importance, pm.status, pm.metadata, pm.created_at, pm.updated_at
		FROM profile_memories pm
		JOIN memory_type_definitions mtd ON mtd.memory_type = pm.memory_type
		WHERE pm.user_id = $1
		  AND pm.status = 'active'
		  AND EXISTS (
		  	SELECT 1
		  	FROM unnest($2::text[]) AS term
		  	WHERE lower(pm.title) LIKE '%' || term || '%'
		  	   OR lower(pm.content) LIKE '%' || term || '%'
		  	   OR lower(coalesce(pm.evidence, '')) LIKE '%' || term || '%'
		  )
		ORDER BY (pm.importance * mtd.retrieval_weight) DESC, pm.updated_at DESC
		LIMIT $3
	`, userID, terms, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProfileMemories(rows)
}

func scanProfileMemories(rows pgx.Rows) ([]ProfileMemory, error) {
	memories := make([]ProfileMemory, 0)
	for rows.Next() {
		memory, err := scanProfileMemory(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, memory)
	}
	return memories, rows.Err()
}

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

func normalizedTags(tags []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		result = append(result, tag)
	}
	return result
}

func estimateStoreTokens(content string) int {
	runes := len([]rune(content))
	if runes == 0 {
		return 0
	}
	return runes/4 + 1
}
