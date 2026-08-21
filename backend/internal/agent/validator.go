package agent

import (
	"encoding/json"
	"errors"
	"strings"
)

var ErrInvalidAction = errors.New("invalid agent action")

func ValidateAction(raw string, tools []ToolDescriptor) (Action, ValidationResult) {
	action, repaired, repairedOutput, err := parseAction(raw)
	if err != nil {
		return Action{}, ValidationResult{Passed: false, Reason: err.Error()}
	}
	if strings.TrimSpace(action.Type) == "" {
		return Action{}, ValidationResult{Passed: false, Reason: "missing action type", Repaired: repaired, RepairOutput: repairedOutput}
	}

	switch action.Type {
	case ActionFinalAnswer:
		if strings.TrimSpace(action.Answer) == "" {
			return Action{}, ValidationResult{Passed: false, Reason: "final_answer requires answer", Repaired: repaired, RepairOutput: repairedOutput}
		}
	case ActionAskUser:
		if strings.TrimSpace(action.Question) == "" {
			return Action{}, ValidationResult{Passed: false, Reason: "ask_user requires question", Repaired: repaired, RepairOutput: repairedOutput}
		}
	case ActionMemorySearch:
		if strings.TrimSpace(action.Query) == "" {
			return Action{}, ValidationResult{Passed: false, Reason: "memory_search requires query", Repaired: repaired, RepairOutput: repairedOutput}
		}
	case ActionToolCall:
		if strings.TrimSpace(action.ToolName) == "" {
			return Action{}, ValidationResult{Passed: false, Reason: "tool_call requires tool_name", Repaired: repaired, RepairOutput: repairedOutput}
		}
		if !toolExists(action.ToolName, tools) {
			return Action{}, ValidationResult{Passed: false, Reason: "tool is not available: " + action.ToolName, Repaired: repaired, RepairOutput: repairedOutput}
		}
		if len(action.Arguments) == 0 {
			action.Arguments = json.RawMessage(`{}`)
		}
	default:
		return Action{}, ValidationResult{Passed: false, Reason: "unsupported action type: " + action.Type, Repaired: repaired, RepairOutput: repairedOutput}
	}

	return action, ValidationResult{Passed: true, Repaired: repaired, RepairOutput: repairedOutput}
}

func parseAction(raw string) (Action, bool, string, error) {
	trimmed := strings.TrimSpace(raw)
	var action Action
	if err := json.Unmarshal([]byte(trimmed), &action); err == nil {
		return action, false, "", nil
	}

	candidate := extractJSONObject(trimmed)
	if candidate == "" {
		return Action{}, false, "", ErrInvalidAction
	}
	if err := json.Unmarshal([]byte(candidate), &action); err != nil {
		return Action{}, false, "", err
	}
	return action, true, candidate, nil
}

func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return ""
	}
	return raw[start : end+1]
}

func toolExists(name string, tools []ToolDescriptor) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
