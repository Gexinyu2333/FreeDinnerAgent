package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"freedinner/backend/internal/store"
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

type SyncResult struct {
	ServerID string   `json:"server_id"`
	UserID   string   `json:"user_id"`
	Tools    []string `json:"tools"`
}

func (r *Runtime) SyncConfiguredTools(ctx context.Context, mcpStore *store.MCPStore, toolStore *store.ToolStore, limit int) ([]SyncResult, error) {
	servers, err := mcpStore.ListEnabledServers(ctx, limit)
	if err != nil {
		return nil, err
	}
	results := make([]SyncResult, 0, len(servers))
	for _, server := range servers {
		def := ServerDefinition{
			ID:          server.Definition.ID,
			Name:        server.Definition.Name,
			DisplayName: server.Definition.DisplayName,
			Description: server.Definition.Description,
			Command:     server.Definition.Command,
			BaseURL:     server.Definition.Endpoint,
			Metadata:    server.Definition.Metadata,
		}
		setting := UserSetting{
			UserID:     server.Setting.UserID,
			ServerID:   server.Setting.MCPServerID,
			IsEnabled:  server.Setting.IsEnabled,
			Env:        server.Setting.EncryptedEnv,
			ToolPolicy: json.RawMessage(`{}`),
		}
		specs := r.DiscoverConfiguredTools(def, setting)
		result := SyncResult{ServerID: server.Definition.ID, UserID: server.Setting.UserID}
		for _, spec := range specs {
			owner := &server.Setting.UserID
			if server.Definition.Visibility == "public" && server.Definition.UserID == nil {
				owner = nil
			}
			metadata, _ := json.Marshal(map[string]any{
				"mcp_server_id":   server.Definition.ID,
				"mcp_server_name": server.Definition.Name,
				"handler_ref":     spec.HandlerRef,
				"transport_type":  server.Definition.TransportType,
				"endpoint":        server.Definition.Endpoint,
				"command":         server.Definition.Command,
			})
			name := "mcp_" + sanitizeName(server.Definition.Name) + "_" + sanitizeName(spec.Name)
			_, err := toolStore.UpsertMCPTool(ctx, store.MCPToolDefinition{
				OwnerUserID:      owner,
				Name:             name,
				Namespace:        "mcp." + sanitizeName(server.Definition.Name),
				DisplayName:      spec.DisplayName,
				Description:      spec.Description,
				HandlerRef:       spec.HandlerRef,
				PermissionLevel:  spec.PermissionLevel,
				RequiresApproval: spec.RequiresApproval,
				ParameterSchema:  spec.ParameterSchema,
				Metadata:         metadata,
			})
			if err != nil {
				return nil, err
			}
			result.Tools = append(result.Tools, name)
		}
		results = append(results, result)
	}
	return results, nil
}

func sanitizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "server"
	}
	return result
}
