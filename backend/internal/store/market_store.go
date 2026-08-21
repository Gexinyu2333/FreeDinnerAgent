package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MarketplaceItem struct {
	ID           string          `json:"id"`
	ItemType     string          `json:"item_type"`
	RefID        string          `json:"ref_id"`
	OwnerUserID  *string         `json:"owner_user_id"`
	Visibility   string          `json:"visibility"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Category     string          `json:"category"`
	Tags         []string        `json:"tags"`
	InstallCount int             `json:"install_count"`
	Rating       *float64        `json:"rating"`
	Status       string          `json:"status"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type MarketplaceReview struct {
	ID                string    `json:"id"`
	MarketplaceItemID string    `json:"marketplace_item_id"`
	UserID            string    `json:"user_id"`
	Rating            int       `json:"rating"`
	Comment           *string   `json:"comment"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CapabilityInstall struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	MarketplaceItemID *string   `json:"marketplace_item_id"`
	CapabilityType    string    `json:"capability_type"`
	CapabilityRefID   string    `json:"capability_ref_id"`
	IsEnabled         bool      `json:"is_enabled"`
	InstallSource     string    `json:"install_source"`
	InstalledAt       time.Time `json:"installed_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type AgentCapabilityBinding struct {
	ID              string    `json:"id"`
	AgentConfigID   string    `json:"agent_config_id"`
	UserID          string    `json:"user_id"`
	CapabilityType  string    `json:"capability_type"`
	CapabilityRefID string    `json:"capability_ref_id"`
	IsEnabled       bool      `json:"is_enabled"`
	LoadMode        string    `json:"load_mode"`
	Priority        int       `json:"priority"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SystemPromptTemplate struct {
	ID            string          `json:"id"`
	OwnerUserID   *string         `json:"owner_user_id"`
	Name          string          `json:"name"`
	DisplayName   string          `json:"display_name"`
	Description   string          `json:"description"`
	Category      string          `json:"category"`
	Tags          []string        `json:"tags"`
	Visibility    string          `json:"visibility"`
	Status        string          `json:"status"`
	LatestVersion int             `json:"latest_version"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type SystemPromptTemplateVersion struct {
	ID                      string          `json:"id"`
	TemplateID              string          `json:"template_id"`
	Version                 int             `json:"version"`
	Content                 string          `json:"content"`
	ChangeNote              *string         `json:"change_note"`
	RecommendedModelFamily  *string         `json:"recommended_model_family"`
	RecommendedCapabilities json.RawMessage `json:"recommended_capabilities"`
	SafetyPolicy            json.RawMessage `json:"safety_policy"`
	TokenEstimate           int             `json:"token_estimate"`
	Status                  string          `json:"status"`
	CreatedAt               time.Time       `json:"created_at"`
}

type SystemPromptVariable struct {
	ID                string          `json:"id"`
	TemplateVersionID string          `json:"template_version_id"`
	Name              string          `json:"name"`
	DisplayName       string          `json:"display_name"`
	Description       *string         `json:"description"`
	DefaultValue      *string         `json:"default_value"`
	Required          bool            `json:"required"`
	ValueType         string          `json:"value_type"`
	Metadata          json.RawMessage `json:"metadata"`
	CreatedAt         time.Time       `json:"created_at"`
}

type SystemPromptVariableInput struct {
	Name         string
	DisplayName  string
	Description  *string
	DefaultValue *string
	Required     bool
	ValueType    string
	Metadata     json.RawMessage
}

type CreateSystemPromptTemplateInput struct {
	UserID      string
	Name        string
	DisplayName string
	Description string
	Category    string
	Tags        []string
	Visibility  string
	Content     string
	ChangeNote  *string
	Variables   []SystemPromptVariableInput
}

type MarketStore struct {
	db *pgxpool.Pool
}

func NewMarketStore(db *pgxpool.Pool) *MarketStore {
	return &MarketStore{db: db}
}

func (s *MarketStore) ListMarketplaceItems(ctx context.Context, userID string, itemType *string, installedOnly bool, limit int) ([]MarketplaceItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	installedFilter := ""
	if installedOnly {
		installedFilter = "AND EXISTS (SELECT 1 FROM user_capability_installs uci WHERE uci.user_id = $1 AND uci.capability_type = mi.item_type AND uci.capability_ref_id = mi.ref_id AND uci.is_enabled = TRUE)"
	}
	rows, err := s.db.Query(ctx, `
		SELECT mi.id, mi.item_type, mi.ref_id, mi.owner_user_id, mi.visibility, mi.title, mi.description,
		       mi.category, mi.tags, mi.install_count, mi.rating, mi.status, mi.metadata, mi.created_at, mi.updated_at
		FROM marketplace_items mi
		WHERE mi.status = 'listed'
		  AND ($2::text IS NULL OR mi.item_type = $2)
		  AND (mi.visibility = 'public' OR mi.owner_user_id = $1)
		  `+installedFilter+`
		ORDER BY mi.install_count DESC, mi.updated_at DESC
		LIMIT $3
	`, userID, itemType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MarketplaceItem
	for rows.Next() {
		item, err := scanMarketplaceItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MarketStore) FindMarketplaceItem(ctx context.Context, userID, itemID string) (MarketplaceItem, error) {
	return scanMarketplaceItem(s.db.QueryRow(ctx, `
		SELECT id, item_type, ref_id, owner_user_id, visibility, title, description,
		       category, tags, install_count, rating, status, metadata, created_at, updated_at
		FROM marketplace_items
		WHERE id = $1 AND status = 'listed' AND (visibility = 'public' OR owner_user_id = $2)
	`, itemID, userID))
}

func (s *MarketStore) UpsertMarketplaceItem(ctx context.Context, item MarketplaceItem) (MarketplaceItem, error) {
	if len(item.Metadata) == 0 {
		item.Metadata = json.RawMessage(`{}`)
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.Visibility == "" {
		item.Visibility = "private"
	}
	if item.Category == "" {
		item.Category = "general"
	}
	return scanMarketplaceItem(s.db.QueryRow(ctx, `
		INSERT INTO marketplace_items (
			id, item_type, ref_id, owner_user_id, visibility, title, description, category, tags, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (item_type, ref_id) DO UPDATE SET
			visibility = EXCLUDED.visibility,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			status = 'listed',
			updated_at = NOW()
		RETURNING id, item_type, ref_id, owner_user_id, visibility, title, description,
		          category, tags, install_count, rating, status, metadata, created_at, updated_at
	`, item.ID, item.ItemType, item.RefID, item.OwnerUserID, item.Visibility, item.Title, item.Description,
		item.Category, item.Tags, item.Metadata))
}

func (s *MarketStore) InstallCapability(ctx context.Context, userID, itemID string) (CapabilityInstall, error) {
	item, err := s.FindMarketplaceItem(ctx, userID, itemID)
	if err != nil {
		return CapabilityInstall{}, err
	}
	install, err := scanCapabilityInstall(s.db.QueryRow(ctx, `
		INSERT INTO user_capability_installs (
			id, user_id, marketplace_item_id, capability_type, capability_ref_id, install_source
		)
		VALUES ($1, $2, $3, $4, $5, 'marketplace')
		ON CONFLICT (user_id, capability_type, capability_ref_id) DO UPDATE SET
			is_enabled = TRUE,
			marketplace_item_id = EXCLUDED.marketplace_item_id,
			updated_at = NOW()
		RETURNING id, user_id, marketplace_item_id, capability_type, capability_ref_id, is_enabled,
		          install_source, installed_at, updated_at
	`, uuid.NewString(), userID, item.ID, item.ItemType, item.RefID))
	if err != nil {
		return CapabilityInstall{}, err
	}
	_, _ = s.db.Exec(ctx, `
		UPDATE marketplace_items
		SET install_count = (
			SELECT COUNT(*)
			FROM user_capability_installs
			WHERE capability_type = $2 AND capability_ref_id = $3
		)
		WHERE id = $1
	`, item.ID, item.ItemType, item.RefID)
	return install, nil
}

func (s *MarketStore) RateMarketplaceItem(ctx context.Context, userID, itemID string, rating int, comment *string) (MarketplaceReview, MarketplaceItem, error) {
	item, err := s.FindMarketplaceItem(ctx, userID, itemID)
	if err != nil {
		return MarketplaceReview{}, MarketplaceItem{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return MarketplaceReview{}, MarketplaceItem{}, err
	}
	defer tx.Rollback(ctx)

	review, err := scanMarketplaceReview(tx.QueryRow(ctx, `
		INSERT INTO marketplace_item_reviews (
			id, marketplace_item_id, user_id, rating, comment
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (marketplace_item_id, user_id) DO UPDATE SET
			rating = EXCLUDED.rating,
			comment = EXCLUDED.comment,
			status = 'visible',
			updated_at = NOW()
		RETURNING id, marketplace_item_id, user_id, rating, comment, status, created_at, updated_at
	`, uuid.NewString(), item.ID, userID, rating, comment))
	if err != nil {
		return MarketplaceReview{}, MarketplaceItem{}, err
	}
	updated, err := scanMarketplaceItem(tx.QueryRow(ctx, `
		UPDATE marketplace_items
		SET rating = (
			SELECT ROUND(AVG(rating)::numeric, 2)
			FROM marketplace_item_reviews
			WHERE marketplace_item_id = $1 AND status = 'visible'
		),
		updated_at = NOW()
		WHERE id = $1
		RETURNING id, item_type, ref_id, owner_user_id, visibility, title, description,
		          category, tags, install_count, rating, status, metadata, created_at, updated_at
	`, item.ID))
	if err != nil {
		return MarketplaceReview{}, MarketplaceItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MarketplaceReview{}, MarketplaceItem{}, err
	}
	return review, updated, nil
}

func (s *MarketStore) SetInstallEnabled(ctx context.Context, userID, installID string, enabled bool) (CapabilityInstall, error) {
	return scanCapabilityInstall(s.db.QueryRow(ctx, `
		UPDATE user_capability_installs
		SET is_enabled = $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, marketplace_item_id, capability_type, capability_ref_id, is_enabled,
		          install_source, installed_at, updated_at
	`, installID, userID, enabled))
}

func (s *MarketStore) BindCapability(ctx context.Context, userID, agentConfigID, capabilityType, capabilityRefID, loadMode string, priority int) (AgentCapabilityBinding, error) {
	if loadMode == "" {
		loadMode = "auto"
	}
	return scanAgentCapabilityBinding(s.db.QueryRow(ctx, `
		INSERT INTO agent_capability_bindings (
			id, agent_config_id, user_id, capability_type, capability_ref_id, load_mode, priority
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (agent_config_id, capability_type, capability_ref_id) DO UPDATE SET
			is_enabled = TRUE,
			load_mode = EXCLUDED.load_mode,
			priority = EXCLUDED.priority,
			updated_at = NOW()
		RETURNING id, agent_config_id, user_id, capability_type, capability_ref_id, is_enabled,
		          load_mode, priority, created_at, updated_at
	`, uuid.NewString(), agentConfigID, userID, capabilityType, capabilityRefID, loadMode, priority))
}

func (s *MarketStore) SetBindingEnabled(ctx context.Context, userID, bindingID string, enabled bool) (AgentCapabilityBinding, error) {
	return scanAgentCapabilityBinding(s.db.QueryRow(ctx, `
		UPDATE agent_capability_bindings
		SET is_enabled = $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, agent_config_id, user_id, capability_type, capability_ref_id, is_enabled,
		          load_mode, priority, created_at, updated_at
	`, bindingID, userID, enabled))
}

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
	version, err := scanSystemPromptTemplateVersion(tx.QueryRow(ctx, `
		INSERT INTO system_prompt_template_versions (
			id, template_id, version, content, change_note, token_estimate
		)
		VALUES ($1, $2, 1, $3, $4, $5)
		RETURNING id, template_id, version, content, change_note, recommended_model_family,
		          recommended_capabilities, safety_policy, token_estimate, status, created_at
	`, uuid.NewString(), template.ID, input.Content, input.ChangeNote, estimateTokens(input.Content)))
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
	var variables []SystemPromptVariable
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

func scanMarketplaceItem(row pgx.Row) (MarketplaceItem, error) {
	var item MarketplaceItem
	if err := row.Scan(&item.ID, &item.ItemType, &item.RefID, &item.OwnerUserID, &item.Visibility,
		&item.Title, &item.Description, &item.Category, &item.Tags, &item.InstallCount, &item.Rating,
		&item.Status, &item.Metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MarketplaceItem{}, ErrNotFound
		}
		return MarketplaceItem{}, err
	}
	return item, nil
}

func scanMarketplaceReview(row pgx.Row) (MarketplaceReview, error) {
	var review MarketplaceReview
	if err := row.Scan(&review.ID, &review.MarketplaceItemID, &review.UserID, &review.Rating,
		&review.Comment, &review.Status, &review.CreatedAt, &review.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MarketplaceReview{}, ErrNotFound
		}
		return MarketplaceReview{}, err
	}
	return review, nil
}

func scanCapabilityInstall(row pgx.Row) (CapabilityInstall, error) {
	var install CapabilityInstall
	if err := row.Scan(&install.ID, &install.UserID, &install.MarketplaceItemID, &install.CapabilityType,
		&install.CapabilityRefID, &install.IsEnabled, &install.InstallSource, &install.InstalledAt,
		&install.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CapabilityInstall{}, ErrNotFound
		}
		return CapabilityInstall{}, err
	}
	return install, nil
}

func scanAgentCapabilityBinding(row pgx.Row) (AgentCapabilityBinding, error) {
	var binding AgentCapabilityBinding
	if err := row.Scan(&binding.ID, &binding.AgentConfigID, &binding.UserID, &binding.CapabilityType,
		&binding.CapabilityRefID, &binding.IsEnabled, &binding.LoadMode, &binding.Priority,
		&binding.CreatedAt, &binding.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentCapabilityBinding{}, ErrNotFound
		}
		return AgentCapabilityBinding{}, err
	}
	return binding, nil
}

func scanSystemPromptTemplate(row pgx.Row) (SystemPromptTemplate, error) {
	var template SystemPromptTemplate
	if err := row.Scan(&template.ID, &template.OwnerUserID, &template.Name, &template.DisplayName,
		&template.Description, &template.Category, &template.Tags, &template.Visibility,
		&template.Status, &template.LatestVersion, &template.Metadata, &template.CreatedAt,
		&template.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SystemPromptTemplate{}, ErrNotFound
		}
		return SystemPromptTemplate{}, err
	}
	return template, nil
}

func scanSystemPromptTemplateVersion(row pgx.Row) (SystemPromptTemplateVersion, error) {
	var version SystemPromptTemplateVersion
	if err := row.Scan(&version.ID, &version.TemplateID, &version.Version, &version.Content,
		&version.ChangeNote, &version.RecommendedModelFamily, &version.RecommendedCapabilities,
		&version.SafetyPolicy, &version.TokenEstimate, &version.Status, &version.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SystemPromptTemplateVersion{}, ErrNotFound
		}
		return SystemPromptTemplateVersion{}, err
	}
	return version, nil
}

func normalizeVisibility(value string) string {
	if strings.TrimSpace(value) == "public" {
		return "public"
	}
	return "private"
}

func estimateTokens(content string) int {
	runes := len([]rune(content))
	if runes == 0 {
		return 0
	}
	return runes/3 + 1
}
