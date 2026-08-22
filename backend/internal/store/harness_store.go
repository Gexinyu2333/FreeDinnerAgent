package store

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentTurn struct {
	ID                 string     `json:"id"`
	UserID             string     `json:"user_id"`
	ConversationID     string     `json:"conversation_id"`
	UserMessageID      *string    `json:"user_message_id"`
	AssistantMessageID *string    `json:"assistant_message_id"`
	AgentConfigID      *string    `json:"agent_config_id"`
	ProviderID         *string    `json:"provider_id"`
	Status             string     `json:"status"`
	CancelRequested    bool       `json:"cancel_requested"`
	ContextBuildID     *string    `json:"context_build_id"`
	ErrorMessage       *string    `json:"error_message"`
	CreatedAt          time.Time  `json:"created_at"`
	StartedAt          *time.Time `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at"`
}

type AgentEvent struct {
	ID             string          `json:"id"`
	TurnID         string          `json:"turn_id"`
	UserID         string          `json:"user_id"`
	ConversationID string          `json:"conversation_id"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	SequenceNo     int             `json:"sequence_no"`
	CreatedAt      time.Time       `json:"created_at"`
}

type AgentLoopStep struct {
	ID             string     `json:"id"`
	TurnID         string     `json:"turn_id"`
	UserID         string     `json:"user_id"`
	ConversationID string     `json:"conversation_id"`
	StepNo         int        `json:"step_no"`
	StepType       string     `json:"step_type"`
	ThoughtSummary *string    `json:"thought_summary"`
	ActionType     *string    `json:"action_type"`
	ActionRefID    *string    `json:"action_ref_id"`
	Observation    *string    `json:"observation"`
	TokenCount     int        `json:"token_count"`
	Status         string     `json:"status"`
	ErrorMessage   *string    `json:"error_message"`
	CreatedAt      time.Time  `json:"created_at"`
	FinishedAt     *time.Time `json:"finished_at"`
}

type AgentTurnCreate struct {
	UserID         string
	ConversationID string
	UserMessageID  *string
	AgentConfigID  *string
	ProviderID     *string
}

type AgentEventCreate struct {
	TurnID         string
	UserID         string
	ConversationID string
	EventType      string
	Payload        json.RawMessage
}

type AgentLoopStepCreate struct {
	TurnID         string
	UserID         string
	ConversationID string
	StepNo         int
	StepType       string
	ThoughtSummary *string
	ActionType     *string
	TokenCount     int
	Status         string
}

type LLMOutputValidationCreate struct {
	TurnID         string
	LoopStepID     *string
	UserID         string
	ValidationType string
	Status         string
	FailureReason  *string
	RepairPrompt   *string
	RepairedOutput *string
	AttemptNo      int
}

type AgentFallbackEventCreate struct {
	TurnID       string
	LoopStepID   *string
	UserID       string
	FallbackType string
	Reason       string
	ActionTaken  string
}

type HarnessStore struct {
	db *pgxpool.Pool
}

func NewHarnessStore(db *pgxpool.Pool) *HarnessStore {
	return &HarnessStore{db: db}
}
