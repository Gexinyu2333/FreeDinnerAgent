package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *ChannelStore) UpsertPolicy(ctx context.Context, input ChannelPolicyUpsert) (ChannelPolicy, error) {
	if len(input.QuietHours) == 0 {
		input.QuietHours = json.RawMessage(`{}`)
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	return scanChannelPolicy(s.db.QueryRow(ctx, `
		INSERT INTO channel_policies (
			id, user_id, channel_connection_id, scope_type, external_scope_id, mode, trigger_keywords,
			allow_memory_write, allow_tool_use, require_approval_for_outbound, rate_limit_per_minute,
			quiet_hours, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (channel_connection_id, scope_type, external_scope_id) DO UPDATE SET
			mode = EXCLUDED.mode,
			trigger_keywords = EXCLUDED.trigger_keywords,
			allow_memory_write = EXCLUDED.allow_memory_write,
			allow_tool_use = EXCLUDED.allow_tool_use,
			require_approval_for_outbound = EXCLUDED.require_approval_for_outbound,
			rate_limit_per_minute = EXCLUDED.rate_limit_per_minute,
			quiet_hours = EXCLUDED.quiet_hours,
			metadata = EXCLUDED.metadata,
			status = 'active',
			updated_at = NOW()
		RETURNING id, user_id, channel_connection_id, scope_type, external_scope_id, mode,
			trigger_keywords, allow_memory_write, allow_tool_use, require_approval_for_outbound,
			rate_limit_per_minute, quiet_hours, status, metadata, created_at, updated_at
	`, uuid.NewString(), input.UserID, input.ChannelConnectionID, input.ScopeType, input.ExternalScopeID,
		input.Mode, input.TriggerKeywords, input.AllowMemoryWrite, input.AllowToolUse,
		input.RequireApprovalForOutbound, input.RateLimitPerMinute, input.QuietHours, input.Metadata))
}

func (s *ChannelStore) FindPolicy(ctx context.Context, connectionID, scopeType string, externalScopeID *string) (ChannelPolicy, error) {
	return scanChannelPolicy(s.db.QueryRow(ctx, `
		SELECT id, user_id, channel_connection_id, scope_type, external_scope_id, mode,
			trigger_keywords, allow_memory_write, allow_tool_use, require_approval_for_outbound,
			rate_limit_per_minute, quiet_hours, status, metadata, created_at, updated_at
		FROM channel_policies
		WHERE channel_connection_id = $1 AND scope_type = $2
			AND (($3::text IS NULL AND external_scope_id IS NULL) OR external_scope_id = $3)
			AND status = 'active'
	`, connectionID, scopeType, externalScopeID))
}

func (s *ChannelStore) ListPolicies(ctx context.Context, userID, connectionID string) ([]ChannelPolicy, error) {
	if _, err := s.FindUserConnectionByID(ctx, userID, connectionID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, channel_connection_id, scope_type, external_scope_id, mode,
			trigger_keywords, allow_memory_write, allow_tool_use, require_approval_for_outbound,
			rate_limit_per_minute, quiet_hours, status, metadata, created_at, updated_at
		FROM channel_policies
		WHERE user_id = $1 AND channel_connection_id = $2 AND status = 'active'
		ORDER BY scope_type ASC, external_scope_id ASC NULLS FIRST, created_at ASC
	`, userID, connectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policies := make([]ChannelPolicy, 0)
	for rows.Next() {
		policy, err := scanChannelPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}
