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
