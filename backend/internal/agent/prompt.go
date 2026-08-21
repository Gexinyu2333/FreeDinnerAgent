package agent

import (
	"encoding/json"
	"strings"
)

func RenderLoopInstructions(tools []ToolDescriptor, skillSections []string) string {
	var builder strings.Builder
	builder.WriteString("Agent Loop 输出协议：每次只能输出一个 JSON 对象，不要输出 Markdown。\n")
	builder.WriteString(`可选动作：{"type":"final_answer","answer":"..."}` + "\n")
	builder.WriteString(`可选动作：{"type":"tool_call","tool_name":"create_task","arguments":{...}}` + "\n")
	builder.WriteString(`可选动作：{"type":"memory_search","query":"..."}` + "\n")
	builder.WriteString(`可选动作：{"type":"ask_user","question":"..."}` + "\n")
	builder.WriteString("工具调用后要等待 observation，再基于真实 observation 继续；工具失败时不能声称成功。\n")
	if len(tools) > 0 {
		builder.WriteString("\n当前可用工具：\n")
		for _, tool := range tools {
			schema := compactJSON(tool.ParameterSchema)
			builder.WriteString("- ")
			builder.WriteString(tool.Name)
			builder.WriteString(": ")
			builder.WriteString(tool.Description)
			if schema != "" {
				builder.WriteString(" 参数 schema: ")
				builder.WriteString(schema)
			}
			builder.WriteString("\n")
		}
	}
	if len(skillSections) > 0 {
		builder.WriteString("\n已披露 Skills：\n")
		for _, section := range skillSections {
			if trimmed := strings.TrimSpace(section); trimmed != "" {
				builder.WriteString("- ")
				builder.WriteString(trimmed)
				builder.WriteString("\n")
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

func RenderObservation(observations []Observation) string {
	if len(observations) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("Tool/Memory Observations:\n")
	for _, observation := range observations {
		builder.WriteString("- ")
		builder.WriteString(observation.ActionType)
		if observation.Failed {
			builder.WriteString(" failed")
		}
		builder.WriteString(": ")
		builder.WriteString(strings.TrimSpace(observation.Text))
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return string(raw)
	}
	return string(encoded)
}
