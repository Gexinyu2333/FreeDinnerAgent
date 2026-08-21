package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Task struct {
	ID                   string     `json:"id"`
	UserID               string     `json:"user_id"`
	ProfileMemoryID      *string    `json:"profile_memory_id"`
	SourceScheduledJobID *string    `json:"source_scheduled_job_id"`
	Title                string     `json:"title"`
	Description          *string    `json:"description"`
	DueAt                *time.Time `json:"due_at"`
	Priority             string     `json:"priority"`
	SourceType           string     `json:"source_type"`
	Status               string     `json:"status"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type TaskCreate struct {
	UserID      string
	Title       string
	Description *string
	DueAt       *time.Time
	Priority    string
	SourceType  string
}

type TaskStore struct {
	db *pgxpool.Pool
}

func NewTaskStore(db *pgxpool.Pool) *TaskStore {
	return &TaskStore{db: db}
}

func (s *TaskStore) Create(ctx context.Context, input TaskCreate) (Task, error) {
	if input.Priority == "" {
		input.Priority = "normal"
	}
	if input.SourceType == "" {
		input.SourceType = "manual"
	}

	query := `
		INSERT INTO tasks (id, user_id, title, description, due_at, priority, source_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, profile_memory_id, source_scheduled_job_id, title, description,
		          due_at, priority, source_type, status, created_at, updated_at
	`
	return scanTask(s.db.QueryRow(ctx, query, uuid.NewString(), input.UserID, input.Title, input.Description,
		input.DueAt, input.Priority, input.SourceType))
}

func (s *TaskStore) List(ctx context.Context, userID string, status *string, limit int) ([]Task, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, profile_memory_id, source_scheduled_job_id, title, description,
		       due_at, priority, source_type, status, created_at, updated_at
		FROM tasks
		WHERE user_id = $1 AND ($2::text IS NULL OR status = $2)
		ORDER BY
			CASE WHEN due_at IS NULL THEN 1 ELSE 0 END,
			due_at ASC,
			updated_at DESC
		LIMIT $3
	`, userID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func scanTask(row pgx.Row) (Task, error) {
	var task Task
	if err := row.Scan(
		&task.ID,
		&task.UserID,
		&task.ProfileMemoryID,
		&task.SourceScheduledJobID,
		&task.Title,
		&task.Description,
		&task.DueAt,
		&task.Priority,
		&task.SourceType,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, err
	}
	return task, nil
}
