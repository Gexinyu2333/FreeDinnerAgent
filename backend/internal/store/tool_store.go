package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ToolDefinition struct {
	ID               string          `json:"id"`
	OwnerUserID      *string         `json:"owner_user_id"`
	Name             string          `json:"name"`
	Namespace        string          `json:"namespace"`
	DisplayName      string          `json:"display_name"`
	Description      string          `json:"description"`
	Category         string          `json:"category"`
	HandlerType      string          `json:"handler_type"`
	HandlerRef       string          `json:"handler_ref"`
	Visibility       string          `json:"visibility"`
	PermissionLevel  string          `json:"permission_level"`
	RequiresApproval bool            `json:"requires_approval"`
	TimeoutMS        int             `json:"timeout_ms"`
	MaxRetries       int             `json:"max_retries"`
	RetryBackoffMS   int             `json:"retry_backoff_ms"`
	IsEnabled        bool            `json:"is_enabled"`
	Metadata         json.RawMessage `json:"metadata"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	ActiveVersion    *int            `json:"active_version,omitempty"`
	ParameterSchema  json.RawMessage `json:"parameter_schema,omitempty"`
	ResultSchema     json.RawMessage `json:"result_schema,omitempty"`
}

type BuiltinToolDefinition struct {
	Name             string
	Namespace        string
	DisplayName      string
	Description      string
	Category         string
	HandlerRef       string
	PermissionLevel  string
	RequiresApproval bool
	ParameterSchema  json.RawMessage
	ResultSchema     json.RawMessage
}

type ToolCallLogCreate struct {
	UserID             string
	ConversationID     string
	MessageID          *string
	ToolID             *string
	ToolName           string
	ToolVersion        *int
	Arguments          json.RawMessage
	ValidatedArguments json.RawMessage
	Result             json.RawMessage
	Status             string
	ErrorType          *string
	ErrorMessage       *string
	AttemptCount       int
	DurationMS         *int
	RequiresApproval   bool
	StartedAt          time.Time
}

type ToolCallLog struct {
	ID                 string          `json:"id"`
	UserID             string          `json:"user_id"`
	ConversationID     string          `json:"conversation_id"`
	MessageID          *string         `json:"message_id"`
	ToolID             *string         `json:"tool_id"`
	ToolName           string          `json:"tool_name"`
	ToolVersion        *int            `json:"tool_version"`
	Arguments          json.RawMessage `json:"arguments"`
	ValidatedArguments json.RawMessage `json:"validated_arguments"`
	Result             json.RawMessage `json:"result"`
	Status             string          `json:"status"`
	ErrorType          *string         `json:"error_type"`
	ErrorMessage       *string         `json:"error_message"`
	AttemptCount       int             `json:"attempt_count"`
	DurationMS         *int            `json:"duration_ms"`
	RequiresApproval   bool            `json:"requires_approval"`
	ApprovedAt         *time.Time      `json:"approved_at"`
	CreatedAt          time.Time       `json:"created_at"`
	StartedAt          *time.Time      `json:"started_at"`
	FinishedAt         *time.Time      `json:"finished_at"`
}

type ToolStore struct {
	db *pgxpool.Pool
}

func NewToolStore(db *pgxpool.Pool) *ToolStore {
	return &ToolStore{db: db}
}

func (s *ToolStore) EnsureBuiltinTools(ctx context.Context, tools []BuiltinToolDefinition) error {
	for _, tool := range tools {
		toolID := uuid.NewString()
		row := s.db.QueryRow(ctx, `
			INSERT INTO tool_definitions (
				id, name, namespace, display_name, description, category, handler_type,
				handler_ref, visibility, permission_level, requires_approval, metadata
			)
			VALUES ($1, $2, $3, $4, $5, $6, 'builtin', $7, 'public', $8, $9, '{}'::jsonb)
			ON CONFLICT (name) DO UPDATE
			SET namespace = EXCLUDED.namespace,
			    display_name = EXCLUDED.display_name,
			    description = EXCLUDED.description,
			    category = EXCLUDED.category,
			    handler_type = EXCLUDED.handler_type,
			    handler_ref = EXCLUDED.handler_ref,
			    permission_level = EXCLUDED.permission_level,
			    requires_approval = EXCLUDED.requires_approval,
			    is_enabled = TRUE,
			    updated_at = NOW()
			RETURNING id
		`, toolID, tool.Name, tool.Namespace, tool.DisplayName, tool.Description, tool.Category,
			tool.HandlerRef, tool.PermissionLevel, tool.RequiresApproval)
		if err := row.Scan(&toolID); err != nil {
			return err
		}

		_, err := s.db.Exec(ctx, `
			INSERT INTO tool_versions (id, tool_id, version, parameter_schema, result_schema, change_note)
			VALUES ($1, $2, 1, $3, $4, 'builtin mvp')
			ON CONFLICT (tool_id, version) DO UPDATE
			SET parameter_schema = EXCLUDED.parameter_schema,
			    result_schema = EXCLUDED.result_schema,
			    status = 'active'
		`, uuid.NewString(), toolID, tool.ParameterSchema, tool.ResultSchema)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ToolStore) ListTools(ctx context.Context, userID string) ([]ToolDefinition, error) {
	rows, err := s.db.Query(ctx, `
		SELECT td.id, td.owner_user_id, td.name, td.namespace, td.display_name, td.description,
		       td.category, td.handler_type, td.handler_ref, td.visibility, td.permission_level,
		       td.requires_approval, td.timeout_ms, td.max_retries, td.retry_backoff_ms,
		       td.is_enabled, td.metadata, td.created_at, td.updated_at,
		       tv.version, tv.parameter_schema, tv.result_schema
		FROM tool_definitions td
		JOIN LATERAL (
			SELECT version, parameter_schema, result_schema
			FROM tool_versions
			WHERE tool_id = td.id AND status = 'active'
			ORDER BY version DESC
			LIMIT 1
		) tv ON TRUE
		WHERE td.is_enabled = TRUE
		  AND (td.visibility = 'public' OR td.owner_user_id = $1)
		ORDER BY td.namespace ASC, td.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tools := make([]ToolDefinition, 0)
	for rows.Next() {
		tool, err := scanToolDefinition(rows)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, rows.Err()
}

func (s *ToolStore) FindTool(ctx context.Context, userID, name string) (ToolDefinition, error) {
	query := `
		SELECT td.id, td.owner_user_id, td.name, td.namespace, td.display_name, td.description,
		       td.category, td.handler_type, td.handler_ref, td.visibility, td.permission_level,
		       td.requires_approval, td.timeout_ms, td.max_retries, td.retry_backoff_ms,
		       td.is_enabled, td.metadata, td.created_at, td.updated_at,
		       tv.version, tv.parameter_schema, tv.result_schema
		FROM tool_definitions td
		JOIN LATERAL (
			SELECT version, parameter_schema, result_schema
			FROM tool_versions
			WHERE tool_id = td.id AND status = 'active'
			ORDER BY version DESC
			LIMIT 1
		) tv ON TRUE
		WHERE td.name = $1 AND td.is_enabled = TRUE
		  AND (td.visibility = 'public' OR td.owner_user_id = $2)
	`
	return scanToolDefinition(s.db.QueryRow(ctx, query, name, userID))
}

func (s *ToolStore) CreateCallLog(ctx context.Context, input ToolCallLogCreate) (ToolCallLog, error) {
	if len(input.Arguments) == 0 {
		input.Arguments = json.RawMessage(`{}`)
	}
	if len(input.ValidatedArguments) == 0 {
		input.ValidatedArguments = input.Arguments
	}
	if len(input.Result) == 0 {
		input.Result = json.RawMessage(`{}`)
	}
	query := `
		INSERT INTO tool_call_logs (
			id, user_id, conversation_id, message_id, tool_id, tool_name, tool_version,
			arguments, validated_arguments, result, status, error_type, error_message,
			attempt_count, duration_ms, requires_approval, started_at, finished_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW())
		RETURNING id, user_id, conversation_id, message_id, tool_id, tool_name, tool_version,
		          arguments, validated_arguments, result, status, error_type, error_message,
		          attempt_count, duration_ms, requires_approval, approved_at, created_at, started_at, finished_at
	`
	return scanToolCallLog(s.db.QueryRow(ctx, query, uuid.NewString(), input.UserID, input.ConversationID,
		input.MessageID, input.ToolID, input.ToolName, input.ToolVersion, input.Arguments, input.ValidatedArguments,
		input.Result, input.Status, input.ErrorType, input.ErrorMessage, input.AttemptCount, input.DurationMS,
		input.RequiresApproval, input.StartedAt))
}

func scanToolDefinition(row pgx.Row) (ToolDefinition, error) {
	var tool ToolDefinition
	if err := row.Scan(
		&tool.ID,
		&tool.OwnerUserID,
		&tool.Name,
		&tool.Namespace,
		&tool.DisplayName,
		&tool.Description,
		&tool.Category,
		&tool.HandlerType,
		&tool.HandlerRef,
		&tool.Visibility,
		&tool.PermissionLevel,
		&tool.RequiresApproval,
		&tool.TimeoutMS,
		&tool.MaxRetries,
		&tool.RetryBackoffMS,
		&tool.IsEnabled,
		&tool.Metadata,
		&tool.CreatedAt,
		&tool.UpdatedAt,
		&tool.ActiveVersion,
		&tool.ParameterSchema,
		&tool.ResultSchema,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ToolDefinition{}, ErrNotFound
		}
		return ToolDefinition{}, err
	}
	return tool, nil
}

func scanToolCallLog(row pgx.Row) (ToolCallLog, error) {
	var log ToolCallLog
	if err := row.Scan(
		&log.ID,
		&log.UserID,
		&log.ConversationID,
		&log.MessageID,
		&log.ToolID,
		&log.ToolName,
		&log.ToolVersion,
		&log.Arguments,
		&log.ValidatedArguments,
		&log.Result,
		&log.Status,
		&log.ErrorType,
		&log.ErrorMessage,
		&log.AttemptCount,
		&log.DurationMS,
		&log.RequiresApproval,
		&log.ApprovedAt,
		&log.CreatedAt,
		&log.StartedAt,
		&log.FinishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ToolCallLog{}, ErrNotFound
		}
		return ToolCallLog{}, err
	}
	return log, nil
}
