package store

import "context"

func (s *ToolStore) ListTools(ctx context.Context, userID string) ([]ToolDefinition, error) {
	rows, err := s.db.Query(ctx, `
		SELECT td.id, td.owner_user_id, td.name, td.namespace, td.display_name, td.description,
		       td.category, td.handler_type, td.handler_ref, td.visibility, td.permission_level,
		       td.requires_approval, td.timeout_ms, td.max_retries, td.retry_backoff_ms,
		       td.is_enabled, td.metadata, td.created_at, td.updated_at,
		       tv.version, tv.parameter_schema, tv.result_schema
		FROM tool_definitions td
		JOIN LATERAL (
			SELECT version, parameter_schema, result_schema
			FROM tool_versions
			WHERE tool_id = td.id AND status = 'active'
			ORDER BY version DESC
			LIMIT 1
		) tv ON TRUE
		WHERE td.is_enabled = TRUE
		  AND (td.visibility = 'public' OR td.owner_user_id = $1)
		ORDER BY td.namespace ASC, td.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tools := make([]ToolDefinition, 0)
	for rows.Next() {
		tool, err := scanToolDefinition(rows)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, rows.Err()
}

func (s *ToolStore) ListAgentBoundTools(ctx context.Context, userID, agentConfigID string) ([]ToolDefinition, error) {
	rows, err := s.db.Query(ctx, `
		SELECT td.id, td.owner_user_id, td.name, td.namespace, td.display_name, td.description,
		       td.category, td.handler_type, td.handler_ref, td.visibility, td.permission_level,
		       td.requires_approval, td.timeout_ms, td.max_retries, td.retry_backoff_ms,
		       td.is_enabled, td.metadata, td.created_at, td.updated_at,
		       tv.version, tv.parameter_schema, tv.result_schema
		FROM agent_capability_bindings b
		JOIN tool_definitions td ON td.id = b.capability_ref_id
		JOIN LATERAL (
			SELECT version, parameter_schema, result_schema
			FROM tool_versions
			WHERE tool_id = td.id AND status = 'active'
			ORDER BY version DESC
			LIMIT 1
		) tv ON TRUE
		WHERE b.user_id = $1
		  AND b.agent_config_id = $2
		  AND b.capability_type = 'tool'
		  AND b.is_enabled = TRUE
		  AND td.is_enabled = TRUE
		  AND (td.visibility = 'public' OR td.owner_user_id = $1)
		ORDER BY b.priority DESC, td.namespace ASC, td.name ASC
	`, userID, agentConfigID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tools := make([]ToolDefinition, 0)
	for rows.Next() {
		tool, err := scanToolDefinition(rows)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, rows.Err()
}

func (s *ToolStore) FindTool(ctx context.Context, userID, name string) (ToolDefinition, error) {
	query := `
		SELECT td.id, td.owner_user_id, td.name, td.namespace, td.display_name, td.description,
		       td.category, td.handler_type, td.handler_ref, td.visibility, td.permission_level,
		       td.requires_approval, td.timeout_ms, td.max_retries, td.retry_backoff_ms,
		       td.is_enabled, td.metadata, td.created_at, td.updated_at,
		       tv.version, tv.parameter_schema, tv.result_schema
		FROM tool_definitions td
		JOIN LATERAL (
			SELECT version, parameter_schema, result_schema
			FROM tool_versions
			WHERE tool_id = td.id AND status = 'active'
			ORDER BY version DESC
			LIMIT 1
		) tv ON TRUE
		WHERE td.name = $1 AND td.is_enabled = TRUE
		  AND (td.visibility = 'public' OR td.owner_user_id = $2)
	`
	return scanToolDefinition(s.db.QueryRow(ctx, query, name, userID))
}
