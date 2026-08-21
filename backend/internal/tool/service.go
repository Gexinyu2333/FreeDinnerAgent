package tool

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
	tools     *store.ToolStore
	agents    *store.AgentConfigStore
	tasks     *store.TaskStore
	memories  *store.MemoryStore
	knowledge *knowledge.Service
	workspace *workspacesvc.Service
}

type ExecuteInput struct {
	UserID         string
	ConversationID string
	ToolName       string
	Arguments      json.RawMessage
	IdempotencyKey *string
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
		tools:     tools,
		agents:    agents,
		tasks:     tasks,
		memories:  memories,
		knowledge: knowledgeService,
		workspace: workspaceService,
	}
}

func BuiltinDefinitions() []store.BuiltinToolDefinition {
	return []store.BuiltinToolDefinition{
		{
			Name:             "create_task",
			Namespace:        "task",
			DisplayName:      "创建任务",
			Description:      "为当前用户创建一个待办任务，可设置描述、截止时间和优先级。",
			Category:         "task",
			HandlerRef:       "builtin.task.create",
			PermissionLevel:  "normal",
			RequiresApproval: false,
			ParameterSchema:  rawSchema(`{"type":"object","required":["title"],"properties":{"title":{"type":"string"},"description":{"type":"string"},"due_at":{"type":"string","format":"date-time"},"priority":{"type":"string","enum":["low","normal","high","urgent"]}}}`),
			ResultSchema:     rawSchema(`{"type":"object","properties":{"task":{"type":"object"}}}`),
		},
		{
			Name:             "list_tasks",
			Namespace:        "task",
			DisplayName:      "查询任务",
			Description:      "查询当前用户的任务列表，可按状态过滤。",
			Category:         "task",
			HandlerRef:       "builtin.task.list",
			PermissionLevel:  "readonly",
			RequiresApproval: false,
			ParameterSchema:  rawSchema(`{"type":"object","properties":{"status":{"type":"string","enum":["open","doing","done","cancelled"]},"limit":{"type":"integer","minimum":1,"maximum":100}}}`),
			ResultSchema:     rawSchema(`{"type":"object","properties":{"tasks":{"type":"array"}}}`),
		},
		{
			Name:             "save_profile_memory",
			Namespace:        "memory",
			DisplayName:      "保存画像记忆",
			Description:      "保存用户长期画像记忆，例如偏好、事实、目标和习惯。",
			Category:         "memory",
			HandlerRef:       "builtin.memory.save_profile",
			PermissionLevel:  "normal",
			RequiresApproval: false,
			ParameterSchema:  rawSchema(`{"type":"object","required":["memory_type","title","content"],"properties":{"memory_type":{"type":"string"},"title":{"type":"string"},"content":{"type":"string"},"evidence":{"type":"string"},"importance":{"type":"integer","minimum":1,"maximum":5},"confidence":{"type":"number","minimum":0,"maximum":1}}}`),
			ResultSchema:     rawSchema(`{"type":"object","properties":{"memory":{"type":"object"}}}`),
		},
		{
			Name:             "search_memory",
			Namespace:        "memory",
			DisplayName:      "检索记忆",
			Description:      "同时检索 Profile Memory 和 Semantic Memory 知识库，返回相关记忆与文档切片。",
			Category:         "memory",
			HandlerRef:       "builtin.memory.search",
			PermissionLevel:  "readonly",
			RequiresApproval: false,
			ParameterSchema:  rawSchema(`{"type":"object","required":["query"],"properties":{"query":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":20}}}`),
			ResultSchema:     rawSchema(`{"type":"object","properties":{"profile_memories":{"type":"array"},"semantic_memory":{"type":"object"}}}`),
		},
		{
			Name:             "run_workspace_command",
			Namespace:        "workspace",
			DisplayName:      "运行 Workspace 命令",
			Description:      "在当前用户启用的 Workspace 内执行白名单 CLI 命令，不经过 shell，并记录命令历史。",
			Category:         "system",
			HandlerRef:       "builtin.workspace.run_command",
			PermissionLevel:  "sensitive",
			RequiresApproval: true,
			ParameterSchema:  rawSchema(`{"type":"object","required":["command"],"properties":{"command":{"type":"string"},"args":{"type":"array","items":{"type":"string"}},"working_dir":{"type":"string"},"timeout_seconds":{"type":"integer","minimum":1,"maximum":120}}}`),
			ResultSchema:     rawSchema(`{"type":"object","properties":{"run":{"type":"object"}}}`),
		},
	}
}

func (s *Service) EnsureBuiltins(ctx context.Context) error {
	return s.tools.EnsureBuiltinTools(ctx, BuiltinDefinitions())
}

func (s *Service) ListTools(ctx context.Context, userID string) ([]store.ToolDefinition, error) {
	return s.tools.ListTools(ctx, userID)
}

func (s *Service) Route(ctx context.Context, input RouteInput) (agent.RouteResult, error) {
	tools, err := s.routableTools(ctx, input.UserID)
	if err != nil {
		return agent.RouteResult{}, err
	}
	candidates := toToolDescriptors(tools)
	result := agent.RouteTools(input.Query, candidates)
	candidateJSON, _ := json.Marshal(result.Candidates)
	selectedJSON, _ := json.Marshal(result.Selected)
	_ = s.tools.CreateRouterLog(ctx, store.ToolRouterLogCreate{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		MessageID:      input.MessageID,
		Query:          input.Query,
		CandidateTools: candidateJSON,
		SelectedTools:  selectedJSON,
		RouteReason:    &result.Reason,
		RiskLevel:      result.RiskLevel,
	})
	return result, nil
}

func (s *Service) routableTools(ctx context.Context, userID string) ([]store.ToolDefinition, error) {
	if s.agents != nil {
		cfg, err := s.agents.GetDefault(ctx, userID)
		if err == nil {
			bound, err := s.tools.ListAgentBoundTools(ctx, userID, cfg.ID)
			if err != nil {
				return nil, err
			}
			if len(bound) > 0 {
				return bound, nil
			}
		}
	}
	return s.tools.ListTools(ctx, userID)
}

func (s *Service) RouteAgentTools(ctx context.Context, input agent.ToolRouteInput) (agent.RouteResult, error) {
	return s.Route(ctx, RouteInput{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		MessageID:      input.MessageID,
		Query:          input.Query,
	})
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
	if shouldRequireApproval(toolDefinition, approvalPolicy) {
		duration := int(time.Since(startedAt).Milliseconds())
		return s.createPendingApproval(ctx, input, toolDefinition, approvalPolicy, duration)
	}

	result, status, errorType, errorMessage := s.executeBuiltin(ctx, input)
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
	})
	return agent.ToolExecuteResult{
		ToolCall:        result.ToolCall,
		ApprovalRequest: result.ApprovalRequest,
		Result:          result.Result,
	}, err
}

func toToolDescriptors(tools []store.ToolDefinition) []agent.ToolDescriptor {
	result := make([]agent.ToolDescriptor, 0, len(tools))
	for _, toolDefinition := range tools {
		result = append(result, agent.ToolDescriptor{
			ID:               toolDefinition.ID,
			Name:             toolDefinition.Name,
			DisplayName:      toolDefinition.DisplayName,
			Description:      toolDefinition.Description,
			PermissionLevel:  toolDefinition.PermissionLevel,
			RequiresApproval: toolDefinition.RequiresApproval,
			ParameterSchema:  toolDefinition.ParameterSchema,
		})
	}
	return result
}

func (s *Service) GetToolCall(ctx context.Context, userID, callID string) (store.ToolCallLog, error) {
	return s.tools.FindCallLog(ctx, userID, callID)
}

func (s *Service) ListConversationToolCalls(ctx context.Context, userID, conversationID string, limit int) ([]store.ToolCallLog, error) {
	return s.tools.ListConversationCallLogs(ctx, userID, conversationID, limit)
}

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

func (s *Service) executeBuiltin(ctx context.Context, input ExecuteInput) (json.RawMessage, string, *string, *string) {
	var result any
	var err error

	switch input.ToolName {
	case "create_task":
		result, err = s.createTask(ctx, input.UserID, input.Arguments)
	case "list_tasks":
		result, err = s.listTasks(ctx, input.UserID, input.Arguments)
	case "save_profile_memory":
		result, err = s.saveProfileMemory(ctx, input.UserID, input.Arguments)
	case "search_memory":
		result, err = s.searchMemory(ctx, input.UserID, input.Arguments)
	case "run_workspace_command":
		result, err = s.runWorkspaceCommand(ctx, input)
	default:
		err = ErrUnsupportedTool
	}

	if err != nil {
		errorType := "execution_error"
		message := err.Error()
		raw, _ := json.Marshal(map[string]any{"error": message})
		return raw, "failed", &errorType, &message
	}
	raw, _ := json.Marshal(result)
	return raw, "success", nil, nil
}

func (s *Service) createTask(ctx context.Context, userID string, arguments json.RawMessage) (any, error) {
	var args struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		DueAt       *string `json:"due_at"`
		Priority    string  `json:"priority"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.Title) == "" {
		return nil, errors.New("title is required")
	}

	var dueAt *time.Time
	if args.DueAt != nil && strings.TrimSpace(*args.DueAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*args.DueAt))
		if err != nil {
			return nil, errors.New("due_at must be RFC3339")
		}
		dueAt = &parsed
	}

	task, err := s.tasks.Create(ctx, store.TaskCreate{
		UserID:      userID,
		Title:       strings.TrimSpace(args.Title),
		Description: trimOptional(args.Description),
		DueAt:       dueAt,
		Priority:    normalizePriority(args.Priority),
		SourceType:  "conversation",
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"task": task}, nil
}

func (s *Service) listTasks(ctx context.Context, userID string, arguments json.RawMessage) (any, error) {
	var args struct {
		Status *string `json:"status"`
		Limit  int     `json:"limit"`
	}
	_ = json.Unmarshal(arguments, &args)
	tasks, err := s.tasks.List(ctx, userID, normalizeTaskStatus(args.Status), args.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"tasks": tasks}, nil
}

func (s *Service) saveProfileMemory(ctx context.Context, userID string, arguments json.RawMessage) (any, error) {
	var args struct {
		MemoryType string  `json:"memory_type"`
		Title      string  `json:"title"`
		Content    string  `json:"content"`
		Evidence   *string `json:"evidence"`
		Importance int     `json:"importance"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.MemoryType) == "" || strings.TrimSpace(args.Title) == "" || strings.TrimSpace(args.Content) == "" {
		return nil, errors.New("memory_type, title and content are required")
	}
	memory, err := s.memories.CreateProfileMemory(ctx, store.ProfileMemoryCreate{
		UserID:     userID,
		MemoryType: strings.TrimSpace(args.MemoryType),
		Scope:      "global",
		Title:      strings.TrimSpace(args.Title),
		Content:    strings.TrimSpace(args.Content),
		Evidence:   trimOptional(args.Evidence),
		Importance: args.Importance,
		Confidence: args.Confidence,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"memory": memory}, nil
}

func (s *Service) searchMemory(ctx context.Context, userID string, arguments json.RawMessage) (any, error) {
	var args struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.Query) == "" {
		return nil, errors.New("query is required")
	}
	limit := args.Limit
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	profile, err := s.memories.SearchProfileMemories(ctx, userID, args.Query, limit)
	if err != nil {
		return nil, err
	}
	semantic, err := s.knowledge.Search(ctx, userID, args.Query, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"profile_memories": profile,
		"semantic_memory":  semantic,
	}, nil
}

func (s *Service) runWorkspaceCommand(ctx context.Context, input ExecuteInput) (any, error) {
	if s.workspace == nil {
		return nil, errors.New("workspace service is not configured")
	}
	var args struct {
		Command        string   `json:"command"`
		Args           []string `json:"args"`
		WorkingDir     string   `json:"working_dir"`
		TimeoutSeconds int      `json:"timeout_seconds"`
	}
	if err := json.Unmarshal(input.Arguments, &args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.Command) == "" {
		return nil, errors.New("command is required")
	}
	conversationID := input.ConversationID
	result, err := s.workspace.RunCommand(ctx, workspacesvc.RunCommandInput{
		UserID:         input.UserID,
		ConversationID: &conversationID,
		Command:        args.Command,
		Args:           args.Args,
		WorkingDir:     args.WorkingDir,
		TimeoutSeconds: args.TimeoutSeconds,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func normalizePriority(value string) string {
	switch value {
	case "low", "high", "urgent":
		return value
	default:
		return "normal"
	}
}

func normalizeTaskStatus(value *string) *string {
	if value == nil {
		return nil
	}
	switch *value {
	case "open", "doing", "done", "cancelled":
		return value
	default:
		return nil
	}
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func rawSchema(value string) json.RawMessage {
	return json.RawMessage(value)
}
