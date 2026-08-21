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
	IdempotencyKey     *string
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
	IdempotencyKey     *string         `json:"idempotency_key"`
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

type ToolApprovalRequest struct {
	ID                string          `json:"id"`
	ToolCallID        string          `json:"tool_call_id"`
	UserID            string          `json:"user_id"`
	ConversationID    string          `json:"conversation_id"`
	TurnID            *string         `json:"turn_id"`
	ApprovalReason    string          `json:"approval_reason"`
	RiskLevel         string          `json:"risk_level"`
	ProposedArguments json.RawMessage `json:"proposed_arguments"`
	Status            string          `json:"status"`
	CreatedAt         time.Time       `json:"created_at"`
	ResolvedAt        *time.Time      `json:"resolved_at"`
}

type ToolApprovalRequestCreate struct {
	ToolCallID        string
	UserID            string
	ConversationID    string
	TurnID            *string
	ApprovalReason    string
	RiskLevel         string
	ProposedArguments json.RawMessage
}

type ToolRouterLogCreate struct {
	UserID         string
	ConversationID string
	MessageID      *string
	Query          string
	CandidateTools json.RawMessage
	SelectedTools  json.RawMessage
	RouteReason    *string
	RiskLevel      string
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

func (s *ToolStore) ListAgentBoundTools(ctx context.Context, userID, agentConfigID string) ([]ToolDefinition, error) {
	rows, err := s.db.Query(ctx, `
		SELECT td.id, td.owner_user_id, td.name, td.namespace, td.display_name, td.description,
		       td.category, td.handler_type, td.handler_ref, td.visibility, td.permission_level,
		       td.requires_approval, td.timeout_ms, td.max_retries, td.retry_backoff_ms,
		       td.is_enabled, td.metadata, td.created_at, td.updated_at,
		       tv.version, tv.parameter_schema, tv.result_schema
		FROM agent_capability_bindings b
		JOIN tool_definitions td ON td.id = b.capability_ref_id
		JOIN LATERAL (
			SELECT version, parameter_schema, result_schema
			FROM tool_versions
			WHERE tool_id = td.id AND status = 'active'
			ORDER BY version DESC
			LIMIT 1
		) tv ON TRUE
		WHERE b.user_id = $1
		  AND b.agent_config_id = $2
		  AND b.capability_type = 'tool'
		  AND b.is_enabled = TRUE
		  AND td.is_enabled = TRUE
		  AND (td.visibility = 'public' OR td.owner_user_id = $1)
		ORDER BY b.priority DESC, td.namespace ASC, td.name ASC
	`, userID, agentConfigID)
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

func (s *ToolStore) CreateRouterLog(ctx context.Context, input ToolRouterLogCreate) error {
	if len(input.CandidateTools) == 0 {
		input.CandidateTools = json.RawMessage(`[]`)
	}
	if len(input.SelectedTools) == 0 {
		input.SelectedTools = json.RawMessage(`[]`)
	}
	if input.RiskLevel == "" {
		input.RiskLevel = "normal"
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO tool_router_logs (
			id, user_id, conversation_id, message_id, query, candidate_tools,
			selected_tools, route_reason, risk_level
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, uuid.NewString(), input.UserID, input.ConversationID, input.MessageID, input.Query,
		input.CandidateTools, input.SelectedTools, input.RouteReason, input.RiskLevel)
	return err
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
			idempotency_key, arguments, validated_arguments, result, status, error_type, error_message,
			attempt_count, duration_ms, requires_approval, started_at, finished_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW())
		RETURNING id, user_id, conversation_id, message_id, tool_id, tool_name, tool_version,
		          idempotency_key, arguments, validated_arguments, result, status, error_type, error_message,
		          attempt_count, duration_ms, requires_approval, approved_at, created_at, started_at, finished_at
	`
	return scanToolCallLog(s.db.QueryRow(ctx, query, uuid.NewString(), input.UserID, input.ConversationID,
		input.MessageID, input.ToolID, input.ToolName, input.ToolVersion, input.IdempotencyKey, input.Arguments, input.ValidatedArguments,
		input.Result, input.Status, input.ErrorType, input.ErrorMessage, input.AttemptCount, input.DurationMS,
		input.RequiresApproval, input.StartedAt))
}

func (s *ToolStore) FindCallLog(ctx context.Context, userID, callID string) (ToolCallLog, error) {
	return scanToolCallLog(s.db.QueryRow(ctx, `
		SELECT id, user_id, conversation_id, message_id, tool_id, tool_name, tool_version,
		       idempotency_key, arguments, validated_arguments, result, status, error_type, error_message,
		       attempt_count, duration_ms, requires_approval, approved_at, created_at, started_at, finished_at
		FROM tool_call_logs
		WHERE id = $1 AND user_id = $2
	`, callID, userID))
}

func (s *ToolStore) ListConversationCallLogs(ctx context.Context, userID, conversationID string, limit int) ([]ToolCallLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, conversation_id, message_id, tool_id, tool_name, tool_version,
		       idempotency_key, arguments, validated_arguments, result, status, error_type, error_message,
		       attempt_count, duration_ms, requires_approval, approved_at, created_at, started_at, finished_at
		FROM tool_call_logs
		WHERE user_id = $1 AND conversation_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]ToolCallLog, 0)
	for rows.Next() {
		log, err := scanToolCallLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (s *ToolStore) CreateApprovalRequest(ctx context.Context, input ToolApprovalRequestCreate) (ToolApprovalRequest, error) {
	if len(input.ProposedArguments) == 0 {
		input.ProposedArguments = json.RawMessage(`{}`)
	}
	if input.RiskLevel == "" {
		input.RiskLevel = "normal"
	}
	return scanToolApprovalRequest(s.db.QueryRow(ctx, `
		INSERT INTO tool_approval_requests (
			id, tool_call_id, user_id, conversation_id, turn_id, approval_reason, risk_level, proposed_arguments
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tool_call_id, user_id, conversation_id, turn_id, approval_reason,
		          risk_level, proposed_arguments, status, created_at, resolved_at
	`, uuid.NewString(), input.ToolCallID, input.UserID, input.ConversationID, input.TurnID,
		input.ApprovalReason, input.RiskLevel, input.ProposedArguments))
}

func (s *ToolStore) ResolveApprovalRequest(ctx context.Context, userID, approvalID, status string) (ToolApprovalRequest, error) {
	return scanToolApprovalRequest(s.db.QueryRow(ctx, `
		WITH resolved AS (
			UPDATE tool_approval_requests
			SET status = $3, resolved_at = NOW()
			WHERE id = $1 AND user_id = $2 AND status = 'pending'
			RETURNING id, tool_call_id, user_id, conversation_id, turn_id, approval_reason,
			          risk_level, proposed_arguments, status, created_at, resolved_at
		), updated_call AS (
			UPDATE tool_call_logs
			SET approved_at = CASE WHEN $3 = 'approved' THEN NOW() ELSE approved_at END,
				status = CASE WHEN $3 = 'rejected' THEN 'cancelled' ELSE status END,
				finished_at = CASE WHEN $3 = 'rejected' THEN NOW() ELSE finished_at END
			WHERE id = (SELECT tool_call_id FROM resolved)
			RETURNING id
		)
		SELECT id, tool_call_id, user_id, conversation_id, turn_id, approval_reason,
		       risk_level, proposed_arguments, status, created_at, resolved_at
		FROM resolved
	`, approvalID, userID, status))
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
		&log.IdempotencyKey,
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

func scanToolApprovalRequest(row pgx.Row) (ToolApprovalRequest, error) {
	var request ToolApprovalRequest
	if err := row.Scan(
		&request.ID,
		&request.ToolCallID,
		&request.UserID,
		&request.ConversationID,
		&request.TurnID,
		&request.ApprovalReason,
		&request.RiskLevel,
		&request.ProposedArguments,
		&request.Status,
		&request.CreatedAt,
		&request.ResolvedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ToolApprovalRequest{}, ErrNotFound
		}
		return ToolApprovalRequest{}, err
	}
	return request, nil
}
