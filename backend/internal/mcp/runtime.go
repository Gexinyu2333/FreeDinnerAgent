package mcp

import (
	"encoding/json"
	"strings"
)

type ServerDefinition struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	Description string          `json:"description"`
	Command     *string         `json:"command,omitempty"`
	BaseURL     *string         `json:"base_url,omitempty"`
	Metadata    json.RawMessage `json:"metadata"`
}

type UserSetting struct {
	UserID     string          `json:"user_id"`
	ServerID   string          `json:"server_id"`
	IsEnabled  bool            `json:"is_enabled"`
	Env        json.RawMessage `json:"-"`
	ToolPolicy json.RawMessage `json:"tool_policy"`
}

type ToolSpec struct {
	Name             string          `json:"name"`
	DisplayName      string          `json:"display_name"`
	Description      string          `json:"description"`
	HandlerRef       string          `json:"handler_ref"`
	PermissionLevel  string          `json:"permission_level"`
	RequiresApproval bool            `json:"requires_approval"`
	ParameterSchema  json.RawMessage `json:"parameter_schema"`
}

type Runtime struct{}

func NewRuntime() *Runtime {
	return &Runtime{}
}

func (r *Runtime) DiscoverConfiguredTools(def ServerDefinition, setting UserSetting) []ToolSpec {
	if !setting.IsEnabled {
		return nil
	}
	var metadata struct {
		Tools []ToolSpec `json:"tools"`
	}
	_ = json.Unmarshal(def.Metadata, &metadata)
	result := make([]ToolSpec, 0, len(metadata.Tools))
	for _, tool := range metadata.Tools {
		if strings.TrimSpace(tool.Name) == "" {
			continue
		}
		if strings.TrimSpace(tool.DisplayName) == "" {
			tool.DisplayName = tool.Name
		}
		if strings.TrimSpace(tool.HandlerRef) == "" {
			tool.HandlerRef = def.Name + "." + tool.Name
		}
		if strings.TrimSpace(tool.PermissionLevel) == "" {
			tool.PermissionLevel = "normal"
		}
		if len(tool.ParameterSchema) == 0 {
			tool.ParameterSchema = json.RawMessage(`{"type":"object"}`)
		}
		result = append(result, tool)
	}
	return result
}
