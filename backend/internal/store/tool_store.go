package store

import (
	"encoding/json"
	"time"

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

type MCPToolDefinition struct {
	OwnerUserID      *string
	Name             string
	Namespace        string
	DisplayName      string
	Description      string
	HandlerRef       string
	PermissionLevel  string
	RequiresApproval bool
	ParameterSchema  json.RawMessage
	Metadata         json.RawMessage
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
