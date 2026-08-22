package store

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

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

func scanMarketplaceItemView(row pgx.Row) (MarketplaceItemView, error) {
	var view MarketplaceItemView
	var install CapabilityInstall
	var installID *string
	var installUserID *string
	var marketplaceItemID *string
	var capabilityType *string
	var capabilityRefID *string
	var isEnabled *bool
	var installSource *string
	var installedAt *time.Time
	var updatedAt *time.Time
	var systemPromptLatestVersionID *string
	if err := row.Scan(&view.ID, &view.ItemType, &view.RefID, &view.OwnerUserID, &view.Visibility,
		&view.Title, &view.Description, &view.Category, &view.Tags, &view.InstallCount, &view.Rating,
		&view.Status, &view.Metadata, &view.CreatedAt, &view.UpdatedAt,
		&installID, &installUserID, &marketplaceItemID, &capabilityType, &capabilityRefID,
		&isEnabled, &installSource, &installedAt, &updatedAt, &systemPromptLatestVersionID); err != nil {
		return MarketplaceItemView{}, err
	}
	view.SystemPromptLatestVersionID = systemPromptLatestVersionID
	if installID != nil {
		install.ID = *installID
		if installUserID != nil {
			install.UserID = *installUserID
		}
		if marketplaceItemID != nil {
			install.MarketplaceItemID = marketplaceItemID
		}
		if capabilityType != nil {
			install.CapabilityType = *capabilityType
		}
		if capabilityRefID != nil {
			install.CapabilityRefID = *capabilityRefID
		}
		if isEnabled != nil {
			install.IsEnabled = *isEnabled
		}
		if installSource != nil {
			install.InstallSource = *installSource
		}
		if installedAt != nil {
			install.InstalledAt = *installedAt
		}
		if updatedAt != nil {
			install.UpdatedAt = *updatedAt
		}
		view.ViewerInstall = &install
	}
	return view, nil
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
