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
