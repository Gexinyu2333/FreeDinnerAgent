package tool

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"freedinner/backend/internal/store"
	workspacesvc "freedinner/backend/internal/workspace"
)

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
