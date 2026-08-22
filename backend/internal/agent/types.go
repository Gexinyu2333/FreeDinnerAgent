package agent

import "encoding/json"

import "freedinner/backend/internal/store"

const (
	ActionFinalAnswer  = "final_answer"
	ActionToolCall     = "tool_call"
	ActionMemorySearch = "memory_search"
	ActionAskUser      = "ask_user"
)

type Action struct {
	Type      string          `json:"type"`
	Answer    string          `json:"answer,omitempty"`
	Question  string          `json:"question,omitempty"`
	ToolName  string          `json:"tool_name,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Query     string          `json:"query,omitempty"`
	DryRun    bool            `json:"dry_run,omitempty"`
}

type ValidationResult struct {
	Passed       bool
	Reason       string
	Repaired     bool
	RepairOutput string
}

type ToolDescriptor struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	DisplayName      string          `json:"display_name"`
	Description      string          `json:"description"`
	PermissionLevel  string          `json:"permission_level"`
	RequiresApproval bool            `json:"requires_approval"`
	ParameterSchema  json.RawMessage `json:"parameter_schema"`
}

type Observation struct {
	ActionType string
	Text       string
	RefID      *string
	Failed     bool
}

type ToolRouteInput struct {
	UserID         string
	ConversationID string
	MessageID      *string
	Query          string
}

type ToolExecuteInput struct {
	UserID         string
	ConversationID string
	ToolName       string
	Arguments      json.RawMessage
	IdempotencyKey *string
	DryRun         bool
}

type ToolExecuteResult struct {
	ToolCall        store.ToolCallLog
	ApprovalRequest *store.ToolApprovalRequest
	Result          json.RawMessage
}
