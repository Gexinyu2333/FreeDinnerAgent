package agent

import (
	"encoding/json"
	"testing"
)

func TestValidateActionAcceptsFinalAnswer(t *testing.T) {
	action, result := ValidateAction(`{"type":"final_answer","answer":"好了"}`, nil)
	if !result.Passed {
		t.Fatalf("expected passed validation, got %#v", result)
	}
	if action.Type != ActionFinalAnswer || action.Answer != "好了" {
		t.Fatalf("unexpected action: %#v", action)
	}
}

func TestValidateActionRepairsFencedJSON(t *testing.T) {
	raw := "```json\n{\"type\":\"memory_search\",\"query\":\"课程资料\"}\n```"
	action, result := ValidateAction(raw, nil)
	if !result.Passed || !result.Repaired {
		t.Fatalf("expected repaired validation, got %#v", result)
	}
	if action.Type != ActionMemorySearch || action.Query != "课程资料" {
		t.Fatalf("unexpected action: %#v", action)
	}
}

func TestValidateActionRejectsUnavailableTool(t *testing.T) {
	_, result := ValidateAction(`{"type":"tool_call","tool_name":"web_search","arguments":{}}`, []ToolDescriptor{
		{Name: "create_task", ParameterSchema: json.RawMessage(`{}`)},
	})
	if result.Passed {
		t.Fatal("expected unavailable tool to fail validation")
	}
}

func TestValidateActionAcceptsDryRunToolCall(t *testing.T) {
	action, result := ValidateAction(`{"type":"tool_call","tool_name":"create_task","dry_run":true,"arguments":{"title":"测试"}}`, []ToolDescriptor{
		{Name: "create_task", ParameterSchema: json.RawMessage(`{}`)},
	})
	if !result.Passed {
		t.Fatalf("expected dry-run tool call to pass, got %#v", result)
	}
	if !action.DryRun {
		t.Fatal("expected dry_run to be preserved")
	}
}

func TestValidateFinalAnswerContractRejectsSuccessClaimAfterFailure(t *testing.T) {
	result := ValidateFinalAnswerContract("已完成，我已经帮你创建好了。", []Observation{
		{ActionType: ActionToolCall, Text: "工具 create_task 执行失败", Failed: true},
	})
	if result.Passed {
		t.Fatal("expected success claim after failed observation to fail")
	}
}

func TestValidateFinalAnswerContractAllowsHonestFailure(t *testing.T) {
	result := ValidateFinalAnswerContract("刚才创建任务失败了，需要你稍后重试。", []Observation{
		{ActionType: ActionToolCall, Text: "工具 create_task 执行失败", Failed: true},
	})
	if !result.Passed {
		t.Fatalf("expected honest failure answer to pass, got %#v", result)
	}
}
