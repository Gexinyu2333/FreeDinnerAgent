package agent

import (
	"strings"
)

type RouteResult struct {
	Candidates []ToolDescriptor
	Selected   []ToolDescriptor
	Reason     string
	RiskLevel  string
}

func RouteTools(query string, tools []ToolDescriptor) RouteResult {
	trimmed := strings.ToLower(strings.TrimSpace(query))
	if trimmed == "" || len(tools) == 0 {
		return RouteResult{Candidates: tools, Selected: nil, Reason: "empty query or no tools", RiskLevel: "low"}
	}
	selected := make([]ToolDescriptor, 0, len(tools))
	for _, tool := range tools {
		haystack := strings.ToLower(tool.Name + " " + tool.DisplayName + " " + tool.Description)
		if containsAny(trimmed, strings.Fields(haystack)) || containsAny(haystack, strings.Fields(trimmed)) || overlapsCJKPhrase(trimmed, haystack) {
			selected = append(selected, tool)
		}
	}
	if len(selected) == 0 {
		selected = readonlyTools(tools)
	}
	if len(selected) == 0 {
		selected = tools
	}
	return RouteResult{
		Candidates: tools,
		Selected:   selected,
		Reason:     "keyword and permission based routing",
		RiskLevel:  routeRisk(selected),
	}
}

func overlapsCJKPhrase(query, haystack string) bool {
	runes := []rune(query)
	if len(runes) < 2 {
		return false
	}
	for index := 0; index < len(runes)-1; index++ {
		part := strings.TrimSpace(string(runes[index : index+2]))
		if len([]rune(part)) < 2 {
			continue
		}
		if strings.Contains(haystack, part) {
			return true
		}
	}
	return false
}

func containsAny(value string, terms []string) bool {
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if len([]rune(term)) < 2 {
			continue
		}
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func readonlyTools(tools []ToolDescriptor) []ToolDescriptor {
	result := make([]ToolDescriptor, 0)
	for _, tool := range tools {
		if tool.PermissionLevel == "readonly" {
			result = append(result, tool)
		}
	}
	return result
}

func routeRisk(tools []ToolDescriptor) string {
	risk := "low"
	for _, tool := range tools {
		switch tool.PermissionLevel {
		case "destructive":
			return "destructive"
		case "sensitive":
			risk = "sensitive"
		case "normal":
			if risk == "low" {
				risk = "normal"
			}
		}
	}
	return risk
}
