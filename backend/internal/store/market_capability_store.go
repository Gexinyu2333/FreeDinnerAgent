package store

import (
	"context"

	"github.com/google/uuid"
)

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
