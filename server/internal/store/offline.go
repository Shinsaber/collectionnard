package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
)

// everything lists a whole table through the paginated store methods (meta-types, collections).
var everything = model.Pagination{Page: 1, PerPage: 1 << 30}

// GetOfflineSnapshot gathers the collections of userID, their items (once each)
// and entries, and every meta-type, for the offline copy of the mobile app.
func (s *PgStore) GetOfflineSnapshot(ctx context.Context, userID uuid.UUID) (*model.OfflineSnapshot, error) {
	u, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	snap := &model.OfflineSnapshot{GeneratedAt: time.Now().UTC(), User: *u}
	if snap.ItemTypes, _, err = s.ListItemTypes(ctx, everything); err != nil {
		return nil, err
	}
	if snap.CollectionTypes, _, err = s.ListCollectionTypes(ctx, everything); err != nil {
		return nil, err
	}
	if snap.Collections, _, err = s.ListCollections(ctx, userID, everything, nil); err != nil {
		return nil, err
	}
	// Tableaux toujours présents dans le JSON (jamais null), même vides.
	snap.ItemTypes = nonNilSlice(snap.ItemTypes)
	snap.CollectionTypes = nonNilSlice(snap.CollectionTypes)
	snap.Collections = nonNilSlice(snap.Collections)
	for i := range snap.Collections {
		snap.Collections[i].CollectionType = nil // déjà dans collection_types
	}

	rows, err := s.pool.Query(ctx, `
		SELECT ci.collection_id, ci.item_id, ci.attributes, ci.created_at
		FROM collection_items ci
		JOIN collections c ON c.id = ci.collection_id
		WHERE c.user_id = $1
		ORDER BY ci.collection_id, ci.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	snap.Entries = []model.OfflineEntry{}
	for rows.Next() {
		var e model.OfflineEntry
		if err := rows.Scan(&e.CollectionID, &e.ItemID, &e.Attributes, &e.AddedAt); err != nil {
			return nil, err
		}
		snap.Entries = append(snap.Entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items, err := s.pool.Query(ctx, `
		SELECT i.id, i.item_type_id, i.attributes, i.created_at, i.updated_at
		FROM items i
		WHERE i.id IN (SELECT ci.item_id FROM collection_items ci
		               JOIN collections c ON c.id = ci.collection_id WHERE c.user_id = $1)
		ORDER BY i.id`, userID)
	if err != nil {
		return nil, err
	}
	defer items.Close()
	snap.Items = []model.Item{}
	for items.Next() {
		var it model.Item
		if err := items.Scan(&it.ID, &it.ItemTypeID, &it.Attributes, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		snap.Items = append(snap.Items, it)
	}
	return snap, items.Err()
}

func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
