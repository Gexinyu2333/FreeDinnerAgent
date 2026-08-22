package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

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

func (s *ToolStore) ListApprovalRequests(ctx context.Context, userID string, status *string, limit int) ([]ToolApprovalRequest, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, tool_call_id, user_id, conversation_id, turn_id, approval_reason,
		       risk_level, proposed_arguments, status, created_at, resolved_at
		FROM tool_approval_requests
		WHERE user_id = $1 AND ($2::text IS NULL OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	requests := make([]ToolApprovalRequest, 0)
	for rows.Next() {
		request, err := scanToolApprovalRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
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
