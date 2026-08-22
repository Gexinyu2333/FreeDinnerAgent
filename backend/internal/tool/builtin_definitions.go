package tool

import (
	"encoding/json"

	"freedinner/backend/internal/store"
)

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

func rawSchema(value string) json.RawMessage {
	return json.RawMessage(value)
}
