package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type KnowledgeDocument struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	Title       string          `json:"title"`
	SourceType  string          `json:"source_type"`
	SourceURI   *string         `json:"source_uri"`
	Visibility  string          `json:"visibility"`
	ContentHash string          `json:"content_hash"`
	Status      string          `json:"status"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type KnowledgeChunk struct {
	ID             string          `json:"id"`
	DocumentID     string          `json:"document_id"`
	UserID         string          `json:"user_id"`
	Visibility     string          `json:"visibility"`
	ChunkIndex     int             `json:"chunk_index"`
	Content        string          `json:"content"`
	TokenCount     int             `json:"token_count"`
	Metadata       json.RawMessage `json:"metadata"`
	HasEmbedding   bool            `json:"has_embedding"`
	Similarity     *float64        `json:"similarity,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	DocumentTitle  *string         `json:"document_title,omitempty"`
	DocumentSource *string         `json:"document_source,omitempty"`
}

type KnowledgeDocumentCreate struct {
	UserID      string
	Title       string
	SourceType  string
	SourceURI   *string
	Visibility  string
	ContentHash string
	Metadata    json.RawMessage
}

type KnowledgeChunkCreate struct {
	DocumentID string
	UserID     string
	Visibility string
	ChunkIndex int
	Content    string
	TokenCount int
	Metadata   json.RawMessage
	Embedding  *string
}

type KnowledgeStore struct {
	db *pgxpool.Pool
}

func NewKnowledgeStore(db *pgxpool.Pool) *KnowledgeStore {
	return &KnowledgeStore{db: db}
}

func (s *KnowledgeStore) CreateDocument(ctx context.Context, input KnowledgeDocumentCreate, chunks []KnowledgeChunkCreate) (KnowledgeDocument, []KnowledgeChunk, error) {
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return KnowledgeDocument{}, nil, err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO knowledge_documents (id, user_id, title, source_type, source_uri, visibility, content_hash, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, title, source_type, source_uri, visibility, content_hash, status, metadata, created_at, updated_at
	`
	document, err := scanKnowledgeDocument(tx.QueryRow(ctx, query, uuid.NewString(), input.UserID, input.Title,
		input.SourceType, input.SourceURI, input.Visibility, input.ContentHash, input.Metadata))
	if err != nil {
		return KnowledgeDocument{}, nil, err
	}

	createdChunks := make([]KnowledgeChunk, 0, len(chunks))
	for _, chunk := range chunks {
		if len(chunk.Metadata) == 0 {
			chunk.Metadata = json.RawMessage(`{}`)
		}
		chunk.DocumentID = document.ID
		created, err := s.createChunk(ctx, tx, chunk)
		if err != nil {
			return KnowledgeDocument{}, nil, err
		}
		createdChunks = append(createdChunks, created)
	}

	if err := tx.Commit(ctx); err != nil {
		return KnowledgeDocument{}, nil, err
	}
	return document, createdChunks, nil
}

func (s *KnowledgeStore) ListDocuments(ctx context.Context, userID string) ([]KnowledgeDocument, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, title, source_type, source_uri, visibility, content_hash, status, metadata, created_at, updated_at
		FROM knowledge_documents
		WHERE status <> 'deleted' AND (user_id = $1 OR visibility = 'public')
		ORDER BY updated_at DESC, created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	documents := make([]KnowledgeDocument, 0)
	for rows.Next() {
		document, err := scanKnowledgeDocument(rows)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, rows.Err()
}

func (s *KnowledgeStore) KeywordSearch(ctx context.Context, userID, query string, limit int) ([]KnowledgeChunk, error) {
	like := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := s.db.Query(ctx, `
		SELECT c.id, c.document_id, c.user_id, c.visibility, c.chunk_index, c.content, c.token_count,
		       c.metadata, c.embedding IS NOT NULL AS has_embedding, NULL::double precision AS similarity,
		       c.created_at, d.title, d.source_uri
		FROM knowledge_chunks c
		JOIN knowledge_documents d ON d.id = c.document_id
		WHERE d.status = 'active'
		  AND (c.user_id = $1 OR c.visibility = 'public')
		  AND lower(c.content) LIKE $2
		ORDER BY c.created_at DESC
		LIMIT $3
	`, userID, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeChunks(rows)
}

func (s *KnowledgeStore) VectorSearch(ctx context.Context, userID, vectorLiteral string, limit int) ([]KnowledgeChunk, error) {
	rows, err := s.db.Query(ctx, `
		SELECT c.id, c.document_id, c.user_id, c.visibility, c.chunk_index, c.content, c.token_count,
		       c.metadata, c.embedding IS NOT NULL AS has_embedding,
		       1 - (c.embedding <=> $2::vector) AS similarity,
		       c.created_at, d.title, d.source_uri
		FROM knowledge_chunks c
		JOIN knowledge_documents d ON d.id = c.document_id
		WHERE d.status = 'active'
		  AND c.embedding IS NOT NULL
		  AND (c.user_id = $1 OR c.visibility = 'public')
		ORDER BY c.embedding <=> $2::vector
		LIMIT $3
	`, userID, vectorLiteral, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeChunks(rows)
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *KnowledgeStore) createChunk(ctx context.Context, q queryer, input KnowledgeChunkCreate) (KnowledgeChunk, error) {
	query := `
		INSERT INTO knowledge_chunks (id, document_id, user_id, visibility, chunk_index, content, token_count, metadata, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CASE WHEN $9::text IS NULL THEN NULL ELSE $9::vector END)
		RETURNING id, document_id, user_id, visibility, chunk_index, content, token_count, metadata,
		          embedding IS NOT NULL AS has_embedding, NULL::double precision AS similarity, created_at,
		          NULL::text AS document_title, NULL::text AS document_source
	`
	return scanKnowledgeChunk(q.QueryRow(ctx, query, uuid.NewString(), input.DocumentID, input.UserID, input.Visibility,
		input.ChunkIndex, input.Content, input.TokenCount, input.Metadata, input.Embedding))
}

func scanKnowledgeDocument(row pgx.Row) (KnowledgeDocument, error) {
	var document KnowledgeDocument
	if err := row.Scan(
		&document.ID,
		&document.UserID,
		&document.Title,
		&document.SourceType,
		&document.SourceURI,
		&document.Visibility,
		&document.ContentHash,
		&document.Status,
		&document.Metadata,
		&document.CreatedAt,
		&document.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return KnowledgeDocument{}, ErrNotFound
		}
		return KnowledgeDocument{}, err
	}
	return document, nil
}

func scanKnowledgeChunks(rows pgx.Rows) ([]KnowledgeChunk, error) {
	chunks := make([]KnowledgeChunk, 0)
	for rows.Next() {
		chunk, err := scanKnowledgeChunk(rows)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, rows.Err()
}

func scanKnowledgeChunk(row pgx.Row) (KnowledgeChunk, error) {
	var chunk KnowledgeChunk
	if err := row.Scan(
		&chunk.ID,
		&chunk.DocumentID,
		&chunk.UserID,
		&chunk.Visibility,
		&chunk.ChunkIndex,
		&chunk.Content,
		&chunk.TokenCount,
		&chunk.Metadata,
		&chunk.HasEmbedding,
		&chunk.Similarity,
		&chunk.CreatedAt,
		&chunk.DocumentTitle,
		&chunk.DocumentSource,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return KnowledgeChunk{}, ErrNotFound
		}
		return KnowledgeChunk{}, err
	}
	return chunk, nil
}

func VectorLiteral(values []float64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%.8f", value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
