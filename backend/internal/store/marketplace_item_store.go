package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

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
	items := make([]MarketplaceItem, 0)
	for rows.Next() {
		item, err := scanMarketplaceItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MarketStore) ListMarketplaceItemViews(ctx context.Context, userID string, itemType *string, installedOnly bool, limit int) ([]MarketplaceItemView, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	installedFilter := ""
	if installedOnly {
		installedFilter = "AND uci.id IS NOT NULL AND uci.is_enabled = TRUE"
	}
	rows, err := s.db.Query(ctx, `
		SELECT mi.id, mi.item_type, mi.ref_id, mi.owner_user_id, mi.visibility, mi.title, mi.description,
		       mi.category, mi.tags, mi.install_count, mi.rating, mi.status, mi.metadata, mi.created_at, mi.updated_at,
		       uci.id, uci.user_id, uci.marketplace_item_id, uci.capability_type, uci.capability_ref_id,
		       uci.is_enabled, uci.install_source, uci.installed_at, uci.updated_at,
		       spv.id
		FROM marketplace_items mi
		LEFT JOIN user_capability_installs uci
		  ON uci.user_id = $1
		 AND uci.capability_type = mi.item_type
		 AND uci.capability_ref_id = mi.ref_id
		LEFT JOIN system_prompt_templates spt
		  ON mi.item_type = 'system_prompt_template'
		 AND spt.id = mi.ref_id
		LEFT JOIN system_prompt_template_versions spv
		  ON spv.template_id = spt.id
		 AND spv.version = spt.latest_version
		 AND spv.status = 'published'
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
	items := make([]MarketplaceItemView, 0)
	for rows.Next() {
		item, err := scanMarketplaceItemView(rows)
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
