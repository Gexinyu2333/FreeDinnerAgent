package tool

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"freedinner/backend/internal/agent"
	"freedinner/backend/internal/knowledge"
	"freedinner/backend/internal/store"
	workspacesvc "freedinner/backend/internal/workspace"
)

var (
	ErrUnsupportedTool  = errors.New("unsupported tool")
	ErrApprovalRequired = errors.New("tool approval required")
)

type Service struct {
	tools      *store.ToolStore
	agents     *store.AgentConfigStore
	tasks      *store.TaskStore
	memories   *store.MemoryStore
	knowledge  *knowledge.Service
	workspace  *workspacesvc.Service
	httpClient *http.Client
}

type ExecuteInput struct {
	UserID         string
	ConversationID string
	ToolName       string
	Arguments      json.RawMessage
	IdempotencyKey *string
	DryRun         bool
}

type ExecuteResult struct {
	ToolCall        store.ToolCallLog          `json:"tool_call"`
	ApprovalRequest *store.ToolApprovalRequest `json:"approval_request,omitempty"`
	Result          json.RawMessage            `json:"result"`
}

type RouteInput struct {
	UserID         string
	ConversationID string
	MessageID      *string
	Query          string
}

func NewService(tools *store.ToolStore, agents *store.AgentConfigStore, tasks *store.TaskStore, memories *store.MemoryStore, knowledgeService *knowledge.Service, workspaceService *workspacesvc.Service) *Service {
	return &Service{
		tools:      tools,
		agents:     agents,
		tasks:      tasks,
		memories:   memories,
		knowledge:  knowledgeService,
		workspace:  workspaceService,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) EnsureBuiltins(ctx context.Context) error {
	return s.tools.EnsureBuiltinTools(ctx, BuiltinDefinitions())
}

func (s *Service) ListTools(ctx context.Context, userID string) ([]store.ToolDefinition, error) {
	return s.tools.ListTools(ctx, userID)
}

func (s *Service) ListApprovals(ctx context.Context, userID string, status *string, limit int) ([]store.ToolApprovalRequest, error) {
	return s.tools.ListApprovalRequests(ctx, userID, status, limit)
}

func (s *Service) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	startedAt := time.Now()
	toolDefinition, err := s.tools.FindTool(ctx, input.UserID, input.ToolName)
	if err != nil {
		return ExecuteResult{}, err
	}
	validatedArguments, validationErr := validateArguments(toolDefinition.ParameterSchema, input.Arguments)
	if validationErr != nil {
		duration := int(time.Since(startedAt).Milliseconds())
		return s.logFailedCall(ctx, input, toolDefinition, input.Arguments, "validation_error", validationErr.Error(), duration)
	}
	input.Arguments = validatedArguments

	approvalPolicy, err := s.approvalPolicy(ctx, input.UserID)
	if err != nil {
		return ExecuteResult{}, err
	}
	if input.DryRun {
		duration := int(time.Since(startedAt).Milliseconds())
		return s.createDryRunResult(ctx, input, toolDefinition, approvalPolicy, duration)
	}
	if shouldRequireApproval(toolDefinition, approvalPolicy) {
		duration := int(time.Since(startedAt).Milliseconds())
		return s.createPendingApproval(ctx, input, toolDefinition, approvalPolicy, duration)
	}

	result, status, errorType, errorMessage := s.executeTool(ctx, toolDefinition, input)
	duration := int(time.Since(startedAt).Milliseconds())
	toolID := toolDefinition.ID
	version := 1
	if toolDefinition.ActiveVersion != nil {
		version = *toolDefinition.ActiveVersion
	}

	callLog, logErr := s.tools.CreateCallLog(ctx, store.ToolCallLogCreate{
		UserID:             input.UserID,
		ConversationID:     input.ConversationID,
		ToolID:             &toolID,
		ToolName:           input.ToolName,
		ToolVersion:        &version,
		IdempotencyKey:     input.IdempotencyKey,
		Arguments:          input.Arguments,
		ValidatedArguments: input.Arguments,
		Result:             result,
		Status:             status,
		ErrorType:          errorType,
		ErrorMessage:       errorMessage,
		AttemptCount:       1,
		DurationMS:         &duration,
		RequiresApproval:   shouldRequireApproval(toolDefinition, approvalPolicy),
		StartedAt:          startedAt,
	})
	if logErr != nil {
		return ExecuteResult{}, logErr
	}
	if status != "success" {
		return ExecuteResult{ToolCall: callLog, Result: result}, errors.New(*errorMessage)
	}
	return ExecuteResult{ToolCall: callLog, Result: result}, nil
}

func (s *Service) ExecuteAgentTool(ctx context.Context, input agent.ToolExecuteInput) (agent.ToolExecuteResult, error) {
	result, err := s.Execute(ctx, ExecuteInput{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		ToolName:       input.ToolName,
		Arguments:      input.Arguments,
		IdempotencyKey: input.IdempotencyKey,
		DryRun:         input.DryRun,
	})
	return agent.ToolExecuteResult{
		ToolCall:        result.ToolCall,
		ApprovalRequest: result.ApprovalRequest,
		Result:          result.Result,
	}, err
}

func (s *Service) GetToolCall(ctx context.Context, userID, callID string) (store.ToolCallLog, error) {
	return s.tools.FindCallLog(ctx, userID, callID)
}

func (s *Service) ListConversationToolCalls(ctx context.Context, userID, conversationID string, limit int) ([]store.ToolCallLog, error) {
	return s.tools.ListConversationCallLogs(ctx, userID, conversationID, limit)
}

func (s *Service) executeTool(ctx context.Context, toolDefinition store.ToolDefinition, input ExecuteInput) (json.RawMessage, string, *string, *string) {
	if toolDefinition.HandlerType == "mcp" {
		return s.executeMCPTool(ctx, toolDefinition, input)
	}
	return s.executeBuiltin(ctx, input)
}
