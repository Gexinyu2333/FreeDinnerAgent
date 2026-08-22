package tool

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"freedinner/backend/internal/store"
)

func (s *Service) ResolveApproval(ctx context.Context, userID, approvalID, status string) (store.ToolApprovalRequest, error) {
	if status != "approved" && status != "rejected" {
		return store.ToolApprovalRequest{}, errors.New("approval status must be approved or rejected")
	}
	return s.tools.ResolveApprovalRequest(ctx, userID, approvalID, status)
}

func (s *Service) logFailedCall(ctx context.Context, input ExecuteInput, toolDefinition store.ToolDefinition, validatedArguments json.RawMessage, errorType string, message string, duration int) (ExecuteResult, error) {
	toolID := toolDefinition.ID
	version := activeVersion(toolDefinition)
	approvalPolicy, _ := s.approvalPolicy(ctx, input.UserID)
	result, _ := json.Marshal(map[string]any{"error": message})
	callLog, err := s.tools.CreateCallLog(ctx, store.ToolCallLogCreate{
		UserID:             input.UserID,
		ConversationID:     input.ConversationID,
		ToolID:             &toolID,
		ToolName:           input.ToolName,
		ToolVersion:        &version,
		IdempotencyKey:     input.IdempotencyKey,
		Arguments:          input.Arguments,
		ValidatedArguments: validatedArguments,
		Result:             result,
		Status:             "failed",
		ErrorType:          &errorType,
		ErrorMessage:       &message,
		AttemptCount:       0,
		DurationMS:         &duration,
		RequiresApproval:   shouldRequireApproval(toolDefinition, approvalPolicy),
		StartedAt:          time.Now(),
	})
	if err != nil {
		return ExecuteResult{}, err
	}
	return ExecuteResult{ToolCall: callLog, Result: result}, errors.New(message)
}

func (s *Service) createPendingApproval(ctx context.Context, input ExecuteInput, toolDefinition store.ToolDefinition, approvalPolicy string, duration int) (ExecuteResult, error) {
	toolID := toolDefinition.ID
	version := activeVersion(toolDefinition)
	result, _ := json.Marshal(map[string]any{"status": "waiting_approval"})
	callLog, err := s.tools.CreateCallLog(ctx, store.ToolCallLogCreate{
		UserID:             input.UserID,
		ConversationID:     input.ConversationID,
		ToolID:             &toolID,
		ToolName:           input.ToolName,
		ToolVersion:        &version,
		IdempotencyKey:     input.IdempotencyKey,
		Arguments:          input.Arguments,
		ValidatedArguments: input.Arguments,
		Result:             result,
		Status:             "pending",
		AttemptCount:       0,
		DurationMS:         &duration,
		RequiresApproval:   true,
		StartedAt:          time.Now(),
	})
	if err != nil {
		return ExecuteResult{}, err
	}
	approval, err := s.tools.CreateApprovalRequest(ctx, store.ToolApprovalRequestCreate{
		ToolCallID:        callLog.ID,
		UserID:            input.UserID,
		ConversationID:    input.ConversationID,
		ApprovalReason:    approvalReason(toolDefinition, approvalPolicy),
		RiskLevel:         riskLevel(toolDefinition),
		ProposedArguments: input.Arguments,
	})
	if err != nil {
		return ExecuteResult{}, err
	}
	return ExecuteResult{ToolCall: callLog, ApprovalRequest: &approval, Result: result}, ErrApprovalRequired
}

func (s *Service) createDryRunResult(ctx context.Context, input ExecuteInput, toolDefinition store.ToolDefinition, approvalPolicy string, duration int) (ExecuteResult, error) {
	toolID := toolDefinition.ID
	version := activeVersion(toolDefinition)
	wouldRequireApproval := shouldRequireApproval(toolDefinition, approvalPolicy)
	result, _ := json.Marshal(map[string]any{
		"dry_run":                 true,
		"tool_name":               input.ToolName,
		"would_execute":           true,
		"would_require_approval":  wouldRequireApproval,
		"validated_arguments":     json.RawMessage(input.Arguments),
		"approval_policy_applied": normalizeApprovalPolicy(approvalPolicy),
	})
	callLog, err := s.tools.CreateCallLog(ctx, store.ToolCallLogCreate{
		UserID:             input.UserID,
		ConversationID:     input.ConversationID,
		ToolID:             &toolID,
		ToolName:           input.ToolName,
		ToolVersion:        &version,
		IdempotencyKey:     input.IdempotencyKey,
		Arguments:          input.Arguments,
		ValidatedArguments: input.Arguments,
		Result:             result,
		Status:             "success",
		AttemptCount:       0,
		DurationMS:         &duration,
		RequiresApproval:   wouldRequireApproval,
		StartedAt:          time.Now(),
	})
	if err != nil {
		return ExecuteResult{}, err
	}
	return ExecuteResult{ToolCall: callLog, Result: result}, nil
}

func activeVersion(toolDefinition store.ToolDefinition) int {
	if toolDefinition.ActiveVersion != nil {
		return *toolDefinition.ActiveVersion
	}
	return 1
}

func (s *Service) approvalPolicy(ctx context.Context, userID string) (string, error) {
	if s.agents == nil {
		return "sensitive_only", nil
	}
	cfg, err := s.agents.GetDefault(ctx, userID)
	if err != nil {
		return "", err
	}
	return normalizeApprovalPolicy(cfg.ToolApprovalPolicy), nil
}

func shouldRequireApproval(toolDefinition store.ToolDefinition, approvalPolicy string) bool {
	switch normalizeApprovalPolicy(approvalPolicy) {
	case "always":
		return true
	case "never":
		return false
	default:
		if toolDefinition.RequiresApproval {
			return true
		}
		return toolDefinition.PermissionLevel == "sensitive" || toolDefinition.PermissionLevel == "destructive"
	}
}

func riskLevel(toolDefinition store.ToolDefinition) string {
	switch toolDefinition.PermissionLevel {
	case "destructive":
		return "destructive"
	case "sensitive":
		return "sensitive"
	default:
		return "normal"
	}
}

func approvalReason(toolDefinition store.ToolDefinition, approvalPolicy string) string {
	if normalizeApprovalPolicy(approvalPolicy) == "always" {
		return "你已设置所有工具调用都需要确认。工具「" + toolDefinition.DisplayName + "」等待批准。"
	}
	return "工具「" + toolDefinition.DisplayName + "」需要用户确认后才能执行。"
}

func normalizeApprovalPolicy(value string) string {
	switch value {
	case "never", "always":
		return value
	default:
		return "sensitive_only"
	}
}
