package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *MarketStore) CreateSystemPromptTemplate(ctx context.Context, input CreateSystemPromptTemplateInput) (SystemPromptTemplate, SystemPromptTemplateVersion, MarketplaceItem, error) {
	if input.Category == "" {
		input.Category = "general"
	}
	visibility := normalizeVisibility(input.Visibility)
	status := "draft"
	if visibility == "public" {
		status = "published"
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return SystemPromptTemplate{}, SystemPromptTemplateVersion{}, MarketplaceItem{}, err
	}
	defer tx.Rollback(ctx)

	template, err := scanSystemPromptTemplate(tx.QueryRow(ctx, `
		INSERT INTO system_prompt_templates (
			id, owner_user_id, name, display_name, description, category, tags, visibility, status, latest_version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1)
		RETURNING id, owner_user_id, name, display_name, description, category, tags, visibility,
		          status, latest_version, metadata, created_at, updated_at
	`, uuid.NewString(), input.UserID, input.Name, input.DisplayName, input.Description, input.Category, input.Tags, visibility, status))
	if err != nil {
		return SystemPromptTemplate{}, SystemPromptTemplateVersion{}, MarketplaceItem{}, err
	}
	if len(input.SafetyPolicy) == 0 {
		input.SafetyPolicy = json.RawMessage(`{}`)
	}
	version, err := scanSystemPromptTemplateVersion(tx.QueryRow(ctx, `
		INSERT INTO system_prompt_template_versions (
			id, template_id, version, content, change_note, safety_policy, token_estimate
		)
		VALUES ($1, $2, 1, $3, $4, $5, $6)
		RETURNING id, template_id, version, content, change_note, recommended_model_family,
		          recommended_capabilities, safety_policy, token_estimate, status, created_at
	`, uuid.NewString(), template.ID, input.Content, input.ChangeNote, input.SafetyPolicy, estimateTokens(input.Content)))
	if err != nil {
		return SystemPromptTemplate{}, SystemPromptTemplateVersion{}, MarketplaceItem{}, err
	}
	for _, variable := range input.Variables {
		if len(variable.Metadata) == 0 {
			variable.Metadata = json.RawMessage(`{}`)
		}
		if variable.ValueType == "" {
			variable.ValueType = "string"
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO system_prompt_template_variables (
				id, template_version_id, name, display_name, description, default_value, required, value_type, metadata
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (template_version_id, name) DO UPDATE SET
				display_name = EXCLUDED.display_name,
				description = EXCLUDED.description,
				default_value = EXCLUDED.default_value,
				required = EXCLUDED.required,
				value_type = EXCLUDED.value_type,
				metadata = EXCLUDED.metadata
		`, uuid.NewString(), version.ID, variable.Name, variable.DisplayName, variable.Description,
			variable.DefaultValue, variable.Required, variable.ValueType, variable.Metadata)
		if err != nil {
			return SystemPromptTemplate{}, SystemPromptTemplateVersion{}, MarketplaceItem{}, err
		}
	}
	itemID := uuid.NewString()
	item, err := scanMarketplaceItem(tx.QueryRow(ctx, `
		INSERT INTO marketplace_items (
			id, item_type, ref_id, owner_user_id, visibility, title, description, category, tags
		)
		VALUES ($1, 'system_prompt_template', $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, item_type, ref_id, owner_user_id, visibility, title, description,
		          category, tags, install_count, rating, status, metadata, created_at, updated_at
	`, itemID, template.ID, input.UserID, visibility, input.DisplayName, input.Description, input.Category, input.Tags))
	if err != nil {
		return SystemPromptTemplate{}, SystemPromptTemplateVersion{}, MarketplaceItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SystemPromptTemplate{}, SystemPromptTemplateVersion{}, MarketplaceItem{}, err
	}
	return template, version, item, nil
}

func (s *MarketStore) ListSystemPromptVariables(ctx context.Context, versionID string) ([]SystemPromptVariable, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, template_version_id, name, display_name, description, default_value,
		       required, value_type, metadata, created_at
		FROM system_prompt_template_variables
		WHERE template_version_id = $1
		ORDER BY created_at ASC, name ASC
	`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	variables := make([]SystemPromptVariable, 0)
	for rows.Next() {
		var variable SystemPromptVariable
		if err := rows.Scan(&variable.ID, &variable.TemplateVersionID, &variable.Name, &variable.DisplayName,
			&variable.Description, &variable.DefaultValue, &variable.Required, &variable.ValueType,
			&variable.Metadata, &variable.CreatedAt); err != nil {
			return nil, err
		}
		variables = append(variables, variable)
	}
	return variables, rows.Err()
}

func (s *MarketStore) FindSystemPromptVersion(ctx context.Context, userID, versionID string) (SystemPromptTemplate, SystemPromptTemplateVersion, error) {
	row := s.db.QueryRow(ctx, `
		SELECT t.id, t.owner_user_id, t.name, t.display_name, t.description, t.category, t.tags, t.visibility,
		       t.status, t.latest_version, t.metadata, t.created_at, t.updated_at,
		       v.id, v.template_id, v.version, v.content, v.change_note, v.recommended_model_family,
		       v.recommended_capabilities, v.safety_policy, v.token_estimate, v.status, v.created_at
		FROM system_prompt_template_versions v
		JOIN system_prompt_templates t ON t.id = v.template_id
		WHERE v.id = $1
		  AND t.status <> 'deleted'
		  AND v.status = 'published'
		  AND (t.visibility = 'public' OR t.owner_user_id = $2)
	`, versionID, userID)
	var template SystemPromptTemplate
	var version SystemPromptTemplateVersion
	if err := row.Scan(&template.ID, &template.OwnerUserID, &template.Name, &template.DisplayName,
		&template.Description, &template.Category, &template.Tags, &template.Visibility,
		&template.Status, &template.LatestVersion, &template.Metadata, &template.CreatedAt,
		&template.UpdatedAt, &version.ID, &version.TemplateID, &version.Version, &version.Content,
		&version.ChangeNote, &version.RecommendedModelFamily, &version.RecommendedCapabilities,
		&version.SafetyPolicy, &version.TokenEstimate, &version.Status, &version.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SystemPromptTemplate{}, SystemPromptTemplateVersion{}, ErrNotFound
		}
		return SystemPromptTemplate{}, SystemPromptTemplateVersion{}, err
	}
	return template, version, nil
}

func (s *MarketStore) ResolveAgentSystemPrompt(ctx context.Context, userID, agentConfigID string) (*SystemPromptTemplateVersion, error) {
	version, err := scanSystemPromptTemplateVersion(s.db.QueryRow(ctx, `
		SELECT v.id, v.template_id, v.version, v.content, v.change_note, v.recommended_model_family,
		       v.recommended_capabilities, v.safety_policy, v.token_estimate, v.status, v.created_at
		FROM agent_capability_bindings b
		JOIN system_prompt_template_versions v ON v.id = b.capability_ref_id
		JOIN system_prompt_templates t ON t.id = v.template_id
		WHERE b.user_id = $1
		  AND b.agent_config_id = $2
		  AND b.capability_type = 'system_prompt_template'
		  AND b.is_enabled = TRUE
		  AND v.status = 'published'
		  AND t.status <> 'deleted'
		  AND (t.visibility = 'public' OR t.owner_user_id = $1)
		ORDER BY b.priority DESC, b.updated_at DESC
		LIMIT 1
	`, userID, agentConfigID))
	if err != nil {
		return nil, err
	}
	return &version, nil
}
