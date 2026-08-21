package agent

import "testing"

func TestRouteToolsSelectsRelevantTool(t *testing.T) {
	tools := []ToolDescriptor{
		{Name: "create_task", DisplayName: "创建任务", Description: "创建一个待办任务", PermissionLevel: "normal"},
		{Name: "search_memory", DisplayName: "检索记忆", Description: "查询用户记忆", PermissionLevel: "readonly"},
	}
	result := RouteTools("帮我创建一个待办", tools)
	if len(result.Selected) == 0 {
		t.Fatal("expected selected tools")
	}
	if result.Selected[0].Name != "create_task" {
		t.Fatalf("expected create_task, got %s", result.Selected[0].Name)
	}
	if result.RiskLevel != "normal" {
		t.Fatalf("expected normal risk, got %s", result.RiskLevel)
	}
}

func TestRouteToolsFallsBackToReadonly(t *testing.T) {
	tools := []ToolDescriptor{
		{Name: "list_tasks", Description: "列出任务", PermissionLevel: "readonly"},
		{Name: "run_workspace_command", Description: "执行命令", PermissionLevel: "sensitive"},
	}
	result := RouteTools("随便聊聊", tools)
	if len(result.Selected) != 1 || result.Selected[0].Name != "list_tasks" {
		t.Fatalf("expected readonly fallback, got %#v", result.Selected)
	}
	if result.RiskLevel != "low" {
		t.Fatalf("expected low risk, got %s", result.RiskLevel)
	}
}
