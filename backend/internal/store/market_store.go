package store

import (
	"encoding/json"
	"time"

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

type MarketplaceItemView struct {
	MarketplaceItem
	ViewerInstall               *CapabilityInstall `json:"viewer_install,omitempty"`
	SystemPromptLatestVersionID *string            `json:"system_prompt_latest_version_id,omitempty"`
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
	UserID       string
	Name         string
	DisplayName  string
	Description  string
	Category     string
	Tags         []string
	Visibility   string
	Content      string
	ChangeNote   *string
	SafetyPolicy json.RawMessage
	Variables    []SystemPromptVariableInput
}

type MarketStore struct {
	db *pgxpool.Pool
}

func NewMarketStore(db *pgxpool.Pool) *MarketStore {
	return &MarketStore{db: db}
}
