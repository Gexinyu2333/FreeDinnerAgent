package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

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
