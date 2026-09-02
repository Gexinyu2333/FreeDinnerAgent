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
		if answer := naturalLanguageFinalAnswer(raw); answer != "" {
			return Action{Type: ActionFinalAnswer, Answer: answer}, ValidationResult{
				Passed:       true,
				Repaired:     true,
				RepairOutput: answer,
			}
		}
		return Action{}, ValidationResult{Passed: false, Reason: err.Error()}
	}
	if strings.TrimSpace(action.Type) == "" {
		if normalized, output, ok := normalizeAlternateAction(raw); ok {
			action = normalized
			repaired = true
			repairedOutput = output
		}
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

func naturalLanguageFinalAnswer(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "{") || strings.Contains(trimmed, "}") {
		return ""
	}
	if strings.Contains(strings.ToLower(trimmed), `"type"`) || strings.Contains(strings.ToLower(trimmed), "tool_call") {
		return ""
	}
	return trimmed
}

func normalizeAlternateAction(raw string) (Action, string, bool) {
	candidate := extractJSONObject(raw)
	if candidate == "" {
		candidate = strings.TrimSpace(raw)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(candidate), &payload); err != nil {
		return Action{}, "", false
	}

	kind := rawString(payload["action"])
	if kind == "" {
		kind = rawString(payload["intent"])
	}
	if kind == "" {
		return Action{}, "", false
	}
	kind = strings.ToLower(strings.TrimSpace(kind))

	var action Action
	switch kind {
	case "respond", "reply", "answer", "final", "final_answer":
		answer := firstRawString(payload, "answer", "content", "message", "text")
		if strings.TrimSpace(answer) == "" {
			return Action{}, "", false
		}
		action = Action{Type: ActionFinalAnswer, Answer: strings.TrimSpace(answer)}
	case "ask", "ask_user", "clarify", "clarification":
		question := firstRawString(payload, "question", "content", "message", "text")
		if strings.TrimSpace(question) == "" {
			return Action{}, "", false
		}
		action = Action{Type: ActionAskUser, Question: strings.TrimSpace(question)}
	case "memory_search", "search_memory":
		query := firstRawString(payload, "query", "content", "text")
		if strings.TrimSpace(query) == "" {
			return Action{}, "", false
		}
		action = Action{Type: ActionMemorySearch, Query: strings.TrimSpace(query)}
	case "tool", "tool_call", "call_tool":
		toolName := firstRawString(payload, "tool_name", "tool", "name")
		if strings.TrimSpace(toolName) == "" {
			return Action{}, "", false
		}
		args := payload["arguments"]
		if len(args) == 0 {
			args = payload["args"]
		}
		if len(args) == 0 {
			args = json.RawMessage(`{}`)
		}
		action = Action{Type: ActionToolCall, ToolName: strings.TrimSpace(toolName), Arguments: args}
	default:
		return Action{}, "", false
	}

	output, _ := json.Marshal(action)
	return action, string(output), true
}

func firstRawString(payload map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		if value := rawString(payload[key]); value != "" {
			return value
		}
	}
	return ""
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return strings.TrimSpace(value)
	}
	return ""
}

func ValidateFinalAnswerContract(answer string, observations []Observation) ValidationResult {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return ValidationResult{Passed: false, Reason: "final_answer requires answer"}
	}
	if !hasFailedObservation(observations) {
		return ValidationResult{Passed: true}
	}
	lower := strings.ToLower(answer)
	successClaims := []string{
		"已完成",
		"已经完成",
		"已创建",
		"已经创建",
		"已保存",
		"已经保存",
		"已发送",
		"已经发送",
		"done",
		"created",
		"saved",
		"sent",
		"successfully",
	}
	for _, claim := range successClaims {
		if strings.Contains(lower, strings.ToLower(claim)) {
			return ValidationResult{Passed: false, Reason: "final answer claims success after failed observation"}
		}
	}
	return ValidationResult{Passed: true}
}

func hasFailedObservation(observations []Observation) bool {
	for _, observation := range observations {
		if observation.Failed {
			return true
		}
	}
	return false
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
