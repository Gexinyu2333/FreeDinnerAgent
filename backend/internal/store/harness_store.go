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

type HarnessStore struct {
	db *pgxpool.Pool
}

func NewHarnessStore(db *pgxpool.Pool) *HarnessStore {
	return &HarnessStore{db: db}
}

func (s *HarnessStore) CreateTurn(ctx context.Context, input AgentTurnCreate) (AgentTurn, error) {
	query := `
		INSERT INTO agent_turns (id, user_id, conversation_id, user_message_id, agent_config_id, provider_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, conversation_id, user_message_id, assistant_message_id, agent_config_id,
		          provider_id, status, cancel_requested, context_build_id, error_message,
		          created_at, started_at, finished_at
	`
	return scanAgentTurn(s.db.QueryRow(ctx, query, uuid.NewString(), input.UserID, input.ConversationID,
		input.UserMessageID, input.AgentConfigID, input.ProviderID))
}

func (s *HarnessStore) StartTurn(ctx context.Context, turnID, userID, conversationID string) (AgentTurn, error) {
	query := `
		UPDATE agent_turns
		SET status = 'running', started_at = COALESCE(started_at, NOW())
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
		RETURNING id, user_id, conversation_id, user_message_id, assistant_message_id, agent_config_id,
		          provider_id, status, cancel_requested, context_build_id, error_message,
		          created_at, started_at, finished_at
	`
	return scanAgentTurn(s.db.QueryRow(ctx, query, turnID, userID, conversationID))
}

func (s *HarnessStore) FinishTurn(ctx context.Context, turnID, userID, conversationID, status string, assistantMessageID *string, errorMessage *string) (AgentTurn, error) {
	query := `
		UPDATE agent_turns
		SET status = $4,
		    assistant_message_id = COALESCE($5, assistant_message_id),
		    error_message = $6,
		    finished_at = NOW()
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
		RETURNING id, user_id, conversation_id, user_message_id, assistant_message_id, agent_config_id,
		          provider_id, status, cancel_requested, context_build_id, error_message,
		          created_at, started_at, finished_at
	`
	return scanAgentTurn(s.db.QueryRow(ctx, query, turnID, userID, conversationID, status, assistantMessageID, errorMessage))
}

func (s *HarnessStore) AddEvent(ctx context.Context, input AgentEventCreate) (AgentEvent, error) {
	if len(input.Payload) == 0 {
		input.Payload = json.RawMessage(`{}`)
	}
	query := `
		WITH next_sequence AS (
			SELECT COALESCE(MAX(sequence_no), 0) + 1 AS sequence_no
			FROM agent_events
			WHERE turn_id = $1
		)
		INSERT INTO agent_events (id, turn_id, user_id, conversation_id, event_type, payload, sequence_no)
		SELECT $2, $1, $3, $4, $5, $6, sequence_no
		FROM next_sequence
		RETURNING id, turn_id, user_id, conversation_id, event_type, payload, sequence_no, created_at
	`
	return scanAgentEvent(s.db.QueryRow(ctx, query, input.TurnID, uuid.NewString(), input.UserID,
		input.ConversationID, input.EventType, input.Payload))
}

func (s *HarnessStore) CreateLoopStep(ctx context.Context, input AgentLoopStepCreate) (AgentLoopStep, error) {
	if input.Status == "" {
		input.Status = "running"
	}
	query := `
		INSERT INTO agent_loop_steps (
			id, turn_id, user_id, conversation_id, step_no, step_type,
			thought_summary, action_type, token_count, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, turn_id, user_id, conversation_id, step_no, step_type, thought_summary,
		          action_type, action_ref_id, observation, token_count, status, error_message,
		          created_at, finished_at
	`
	return scanAgentLoopStep(s.db.QueryRow(ctx, query, uuid.NewString(), input.TurnID, input.UserID,
		input.ConversationID, input.StepNo, input.StepType, input.ThoughtSummary, input.ActionType,
		input.TokenCount, input.Status))
}

func (s *HarnessStore) FinishLoopStep(ctx context.Context, stepID, userID, conversationID, status string, observation *string, errorMessage *string) (AgentLoopStep, error) {
	query := `
		UPDATE agent_loop_steps
		SET status = $4, observation = $5, error_message = $6, finished_at = NOW()
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
		RETURNING id, turn_id, user_id, conversation_id, step_no, step_type, thought_summary,
		          action_type, action_ref_id, observation, token_count, status, error_message,
		          created_at, finished_at
	`
	return scanAgentLoopStep(s.db.QueryRow(ctx, query, stepID, userID, conversationID, status, observation, errorMessage))
}

func (s *HarnessStore) GetTurn(ctx context.Context, userID, conversationID, turnID string) (AgentTurn, error) {
	query := `
		SELECT id, user_id, conversation_id, user_message_id, assistant_message_id, agent_config_id,
		       provider_id, status, cancel_requested, context_build_id, error_message,
		       created_at, started_at, finished_at
		FROM agent_turns
		WHERE id = $1 AND user_id = $2 AND conversation_id = $3
	`
	return scanAgentTurn(s.db.QueryRow(ctx, query, turnID, userID, conversationID))
}

func (s *HarnessStore) ListEvents(ctx context.Context, userID, conversationID string, turnID *string) ([]AgentEvent, error) {
	query := `
		SELECT id, turn_id, user_id, conversation_id, event_type, payload, sequence_no, created_at
		FROM agent_events
		WHERE user_id = $1 AND conversation_id = $2 AND ($3::uuid IS NULL OR turn_id = $3)
		ORDER BY created_at ASC, sequence_no ASC
	`
	rows, err := s.db.Query(ctx, query, userID, conversationID, turnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]AgentEvent, 0)
	for rows.Next() {
		event, err := scanAgentEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *HarnessStore) ListLoopSteps(ctx context.Context, userID, conversationID, turnID string) ([]AgentLoopStep, error) {
	if _, err := s.GetTurn(ctx, userID, conversationID, turnID); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, turn_id, user_id, conversation_id, step_no, step_type, thought_summary,
		       action_type, action_ref_id, observation, token_count, status, error_message,
		       created_at, finished_at
		FROM agent_loop_steps
		WHERE user_id = $1 AND conversation_id = $2 AND turn_id = $3
		ORDER BY step_no ASC
	`, userID, conversationID, turnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	steps := make([]AgentLoopStep, 0)
	for rows.Next() {
		step, err := scanAgentLoopStep(rows)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func scanAgentTurn(row pgx.Row) (AgentTurn, error) {
	var turn AgentTurn
	if err := row.Scan(
		&turn.ID,
		&turn.UserID,
		&turn.ConversationID,
		&turn.UserMessageID,
		&turn.AssistantMessageID,
		&turn.AgentConfigID,
		&turn.ProviderID,
		&turn.Status,
		&turn.CancelRequested,
		&turn.ContextBuildID,
		&turn.ErrorMessage,
		&turn.CreatedAt,
		&turn.StartedAt,
		&turn.FinishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentTurn{}, ErrNotFound
		}
		return AgentTurn{}, err
	}
	return turn, nil
}

func scanAgentEvent(row pgx.Row) (AgentEvent, error) {
	var event AgentEvent
	if err := row.Scan(
		&event.ID,
		&event.TurnID,
		&event.UserID,
		&event.ConversationID,
		&event.EventType,
		&event.Payload,
		&event.SequenceNo,
		&event.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentEvent{}, ErrNotFound
		}
		return AgentEvent{}, err
	}
	return event, nil
}

func scanAgentLoopStep(row pgx.Row) (AgentLoopStep, error) {
	var step AgentLoopStep
	if err := row.Scan(
		&step.ID,
		&step.TurnID,
		&step.UserID,
		&step.ConversationID,
		&step.StepNo,
		&step.StepType,
		&step.ThoughtSummary,
		&step.ActionType,
		&step.ActionRefID,
		&step.Observation,
		&step.TokenCount,
		&step.Status,
		&step.ErrorMessage,
		&step.CreatedAt,
		&step.FinishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentLoopStep{}, ErrNotFound
		}
		return AgentLoopStep{}, err
	}
	return step, nil
}
