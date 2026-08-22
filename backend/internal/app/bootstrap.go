package app

import (
	"context"
	"encoding/json"

	"freedinner/backend/internal/store"
)

func syncToolMarketplaceItems(ctx context.Context, marketStore *store.MarketStore, toolStore *store.ToolStore) error {
	tools, err := toolStore.ListTools(ctx, "")
	if err != nil {
		return err
	}
	for _, tool := range tools {
		if tool.Visibility != "public" {
			continue
		}
		metadata, _ := json.Marshal(map[string]any{
			"handler_type":      tool.HandlerType,
			"permission_level":  tool.PermissionLevel,
			"requires_approval": tool.RequiresApproval,
		})
		_, err := marketStore.UpsertMarketplaceItem(ctx, store.MarketplaceItem{
			ItemType:    "tool",
			RefID:       tool.ID,
			Visibility:  "public",
			Title:       tool.DisplayName,
			Description: tool.Description,
			Category:    tool.Category,
			Tags:        []string{tool.Namespace, tool.PermissionLevel},
			Metadata:    metadata,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func syncChannelMarketplaceItems(ctx context.Context, marketStore *store.MarketStore, channelStore *store.ChannelStore) error {
	providers, err := channelStore.ListProviders(ctx)
	if err != nil {
		return err
	}
	for _, provider := range providers {
		metadata, _ := json.Marshal(map[string]any{
			"provider_type":  provider.ProviderType,
			"adapter_type":   provider.AdapterType,
			"inbound_modes":  provider.InboundModes,
			"outbound_modes": provider.OutboundModes,
		})
		_, err := marketStore.UpsertMarketplaceItem(ctx, store.MarketplaceItem{
			ItemType:    "channel_adapter",
			RefID:       provider.ID,
			Visibility:  "public",
			Title:       provider.DisplayName,
			Description: provider.Description,
			Category:    provider.ProviderType,
			Tags:        []string{provider.ProviderType, provider.AdapterType},
			Metadata:    metadata,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
