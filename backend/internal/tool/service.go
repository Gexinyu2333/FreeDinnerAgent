package tool

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"freedinner/backend/internal/knowledge"
	"freedinner/backend/internal/store"
	workspacesvc "freedinner/backend/internal/workspace"
)

var ErrUnsupportedTool = errors.New("unsupported tool")

type Service struct {
	tools     *store.ToolStore
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
}

type ExecuteResult struct {
	ToolCall store.ToolCallLog `json:"tool_call"`
	Result   json.RawMessage   `json:"result"`
}

func NewService(tools *store.ToolStore, tasks *store.TaskStore, memories *store.MemoryStore, knowledgeService *knowledge.Service, workspaceService *workspacesvc.Service) *Service {
	return &Service{
		tools:     tools,
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
			RequiresApproval: false,
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

func (s *Service) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	startedAt := time.Now()
	toolDefinition, err := s.tools.FindTool(ctx, input.UserID, input.ToolName)
	if err != nil {
		return ExecuteResult{}, err
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
		Arguments:          input.Arguments,
		ValidatedArguments: input.Arguments,
		Result:             result,
		Status:             status,
		ErrorType:          errorType,
		ErrorMessage:       errorMessage,
		AttemptCount:       1,
		DurationMS:         &duration,
		RequiresApproval:   toolDefinition.RequiresApproval,
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
