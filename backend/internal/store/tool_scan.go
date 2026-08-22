package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

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
