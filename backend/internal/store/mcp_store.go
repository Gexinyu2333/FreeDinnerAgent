package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MCPServerDefinition struct {
	ID              string          `json:"id"`
	UserID          *string         `json:"user_id"`
	Name            string          `json:"name"`
	DisplayName     string          `json:"display_name"`
	Description     string          `json:"description"`
	TransportType   string          `json:"transport_type"`
	Endpoint        *string         `json:"endpoint"`
	Command         *string         `json:"command"`
	Visibility      string          `json:"visibility"`
	PermissionLevel string          `json:"permission_level"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type MCPUserSetting struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	MCPServerID    string          `json:"mcp_server_id"`
	IsEnabled      bool            `json:"is_enabled"`
	EncryptedEnv   json.RawMessage `json:"-"`
	ApprovalPolicy string          `json:"approval_policy"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type EnabledMCPServer struct {
	Definition MCPServerDefinition `json:"definition"`
	Setting    MCPUserSetting      `json:"setting"`
}

type MCPStore struct {
	db *pgxpool.Pool
}

func NewMCPStore(db *pgxpool.Pool) *MCPStore {
	return &MCPStore{db: db}
}

func (s *MCPStore) ListEnabledServers(ctx context.Context, limit int) ([]EnabledMCPServer, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.Query(ctx, `
		SELECT d.id, d.user_id, d.name, d.display_name, d.description, d.transport_type,
		       d.endpoint, d.command, d.visibility, d.permission_level, d.status, d.metadata,
		       d.created_at, d.updated_at,
		       us.id, us.user_id, us.mcp_server_id, us.is_enabled, us.encrypted_env,
		       us.approval_policy, us.created_at, us.updated_at
		FROM user_mcp_server_settings us
		JOIN mcp_server_definitions d ON d.id = us.mcp_server_id
		WHERE us.is_enabled = TRUE
		  AND d.status = 'active'
		ORDER BY us.updated_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []EnabledMCPServer
	for rows.Next() {
		var item EnabledMCPServer
		err := rows.Scan(&item.Definition.ID, &item.Definition.UserID, &item.Definition.Name,
			&item.Definition.DisplayName, &item.Definition.Description, &item.Definition.TransportType,
			&item.Definition.Endpoint, &item.Definition.Command, &item.Definition.Visibility,
			&item.Definition.PermissionLevel, &item.Definition.Status, &item.Definition.Metadata,
			&item.Definition.CreatedAt, &item.Definition.UpdatedAt,
			&item.Setting.ID, &item.Setting.UserID, &item.Setting.MCPServerID, &item.Setting.IsEnabled,
			&item.Setting.EncryptedEnv, &item.Setting.ApprovalPolicy, &item.Setting.CreatedAt,
			&item.Setting.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
