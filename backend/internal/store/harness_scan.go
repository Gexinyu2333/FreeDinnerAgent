package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

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
