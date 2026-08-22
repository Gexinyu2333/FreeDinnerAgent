package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *ToolStore) EnsureBuiltinTools(ctx context.Context, tools []BuiltinToolDefinition) error {
	for _, tool := range tools {
		toolID := uuid.NewString()
		row := s.db.QueryRow(ctx, `
			INSERT INTO tool_definitions (
				id, name, namespace, display_name, description, category, handler_type,
				handler_ref, visibility, permission_level, requires_approval, metadata
			)
			VALUES ($1, $2, $3, $4, $5, $6, 'builtin', $7, 'public', $8, $9, '{}'::jsonb)
			ON CONFLICT (name) DO UPDATE
			SET namespace = EXCLUDED.namespace,
			    display_name = EXCLUDED.display_name,
			    description = EXCLUDED.description,
			    category = EXCLUDED.category,
			    handler_type = EXCLUDED.handler_type,
			    handler_ref = EXCLUDED.handler_ref,
			    permission_level = EXCLUDED.permission_level,
			    requires_approval = EXCLUDED.requires_approval,
			    is_enabled = TRUE,
			    updated_at = NOW()
			RETURNING id
		`, toolID, tool.Name, tool.Namespace, tool.DisplayName, tool.Description, tool.Category,
			tool.HandlerRef, tool.PermissionLevel, tool.RequiresApproval)
		if err := row.Scan(&toolID); err != nil {
			return err
		}

		_, err := s.db.Exec(ctx, `
			INSERT INTO tool_versions (id, tool_id, version, parameter_schema, result_schema, change_note)
			VALUES ($1, $2, 1, $3, $4, 'builtin mvp')
			ON CONFLICT (tool_id, version) DO UPDATE
			SET parameter_schema = EXCLUDED.parameter_schema,
			    result_schema = EXCLUDED.result_schema,
			    status = 'active'
		`, uuid.NewString(), toolID, tool.ParameterSchema, tool.ResultSchema)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ToolStore) UpsertMCPTool(ctx context.Context, tool MCPToolDefinition) (ToolDefinition, error) {
	if len(tool.ParameterSchema) == 0 {
		tool.ParameterSchema = json.RawMessage(`{"type":"object"}`)
	}
	if len(tool.Metadata) == 0 {
		tool.Metadata = json.RawMessage(`{}`)
	}
	if tool.PermissionLevel == "" {
		tool.PermissionLevel = "normal"
	}
	visibility := "private"
	if tool.OwnerUserID == nil {
		visibility = "public"
	}
	toolID := uuid.NewString()
	row := s.db.QueryRow(ctx, `
		INSERT INTO tool_definitions (
			id, owner_user_id, name, namespace, display_name, description, category,
			handler_type, handler_ref, visibility, permission_level, requires_approval, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'mcp', 'mcp', $7, $8, $9, $10, $11)
		ON CONFLICT (name) DO UPDATE SET
			owner_user_id = EXCLUDED.owner_user_id,
			namespace = EXCLUDED.namespace,
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			handler_type = EXCLUDED.handler_type,
			handler_ref = EXCLUDED.handler_ref,
			visibility = EXCLUDED.visibility,
			permission_level = EXCLUDED.permission_level,
			requires_approval = EXCLUDED.requires_approval,
			metadata = EXCLUDED.metadata,
			is_enabled = TRUE,
			updated_at = NOW()
		RETURNING id
	`, toolID, tool.OwnerUserID, tool.Name, tool.Namespace, tool.DisplayName, tool.Description,
		tool.HandlerRef, visibility, tool.PermissionLevel, tool.RequiresApproval, tool.Metadata)
	if err := row.Scan(&toolID); err != nil {
		return ToolDefinition{}, err
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO tool_versions (id, tool_id, version, parameter_schema, result_schema, change_note)
		VALUES ($1, $2, 1, $3, '{"type":"object"}'::jsonb, 'mcp metadata sync')
		ON CONFLICT (tool_id, version) DO UPDATE SET
			parameter_schema = EXCLUDED.parameter_schema,
			status = 'active'
	`, uuid.NewString(), toolID, tool.ParameterSchema)
	if err != nil {
		return ToolDefinition{}, err
	}
	return s.FindTool(ctx, derefOwner(tool.OwnerUserID), tool.Name)
}
