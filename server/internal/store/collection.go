package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// collectionColumns / scanCollection: every collection read (with its CollectionType and item count).
var collectionColumns = `c.id, c.user_id, c.collection_type_id, c.name, c.description, c.visibility, c.hidden_fields,
		c.created_at, c.updated_at,
		ct.id, ct.name, ct.description, ct.attributes_schema, ` + ctItemTypeIDs("ct") + `, ct.created_at, ct.updated_at,
		(SELECT count(DISTINCT ci.item_id) FROM collection_items ci WHERE ci.collection_id = c.id),
		(SELECT COALESCE(sum(ci.quantity), 0) FROM collection_items ci WHERE ci.collection_id = c.id)`

const collectionFrom = ` FROM collections c JOIN collection_types ct ON ct.id = c.collection_type_id`

func scanCollection(row pgx.Row) (*model.Collection, error) {
	var col model.Collection
	var ct model.CollectionType
	if err := row.Scan(
		&col.ID, &col.UserID, &col.CollectionTypeID, &col.Name, &col.Description, &col.Visibility, &col.HiddenFields,
		&col.CreatedAt, &col.UpdatedAt,
		&ct.ID, &ct.Name, &ct.Description, &ct.AttributesSchema, &ct.ItemTypeIDs, &ct.CreatedAt, &ct.UpdatedAt,
		&col.ItemCount, &col.CopyCount,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	col.CollectionType = &ct
	return &col, nil
}

func (s *PgStore) ListCollections(ctx context.Context, userID uuid.UUID, pg model.Pagination, collectionTypeID *uuid.UUID) ([]model.Collection, int, error) {
	where := "c.user_id = $1"
	args := []any{userID}
	argN := 2

	if collectionTypeID != nil {
		where += fmt.Sprintf(" AND c.collection_type_id = $%d", argN)
		args = append(args, *collectionTypeID)
		argN++
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT count(*) FROM collections c WHERE %s`, where),
		countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, pg.PerPage, pg.Offset())
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`SELECT %s%s WHERE %s ORDER BY c.name LIMIT $%d OFFSET $%d`,
		collectionColumns, collectionFrom, where, argN, argN+1), args...)
	if err != nil {
		return nil, 0, err
	}
	result, err := collectCollections(rows)
	return result, total, err
}

func collectCollections(rows pgx.Rows) ([]model.Collection, error) {
	defer rows.Close()
	result := []model.Collection{}
	for rows.Next() {
		col, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *col)
	}
	return result, rows.Err()
}

func (s *PgStore) GetCollection(ctx context.Context, id uuid.UUID) (*model.Collection, error) {
	return scanCollection(s.pool.QueryRow(ctx, `SELECT `+collectionColumns+collectionFrom+` WHERE c.id = $1`, id))
}

// ListVisibleCollections lists the collections of owner whose visibility is in
// levels (P4-2), the largest first.
func (s *PgStore) ListVisibleCollections(ctx context.Context, ownerID uuid.UUID, levels []string) ([]model.Collection, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+collectionColumns+collectionFrom+`
		WHERE c.user_id = $1 AND c.visibility = ANY($2)
		ORDER BY (SELECT count(*) FROM collection_items ci WHERE ci.collection_id = c.id) DESC, c.name
		LIMIT 500`, ownerID, levels)
	if err != nil {
		return nil, err
	}
	return collectCollections(rows)
}

func (s *PgStore) CreateCollection(ctx context.Context, userID uuid.UUID, in model.CollectionCreate) (*model.Collection, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx,
		`INSERT INTO collections (user_id, collection_type_id, name, description)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, in.CollectionTypeID, in.Name, in.Description,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.GetCollection(ctx, id)
}

func (s *PgStore) UpdateCollection(ctx context.Context, id uuid.UUID, in model.CollectionUpdate) (*model.Collection, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE collections SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			visibility = COALESCE($4, visibility),
			hidden_fields = CASE WHEN $5 THEN $6::text[] ELSE hidden_fields END,
			updated_at = now()
		 WHERE id = $1`,
		id, in.Name, in.Description, in.Visibility, in.HiddenFields.Set, in.HiddenFields.Value,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetCollection(ctx, id)
}

func (s *PgStore) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM collections WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- CollectionItem ---

func (s *PgStore) ListCollectionItems(ctx context.Context, collectionID uuid.UUID, pg model.Pagination, f CollectionItemFilters) ([]model.CollectionItem, int, error) {
	q, err := buildItemQuery(f.ItemFilters)
	if err != nil {
		return nil, 0, err
	}
	q.add("ci.collection_id = " + q.arg(collectionID))
	if f.Multiple {
		q.add("ci.quantity > 1")
	}
	where := q.whereClause()

	var total int
	err = s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT count(*) FROM collection_items ci JOIN items i ON i.id = ci.item_id WHERE %s`, where),
		q.args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	sortClause := q.lotOrderBy(f.ItemFilters) // may add the sort key to q.args
	limit, offset := q.arg(pg.PerPage), q.arg(pg.Offset())
	query := fmt.Sprintf(`
		SELECT ci.id, ci.collection_id, ci.item_id, ci.quantity, ci.attributes, ci.created_at, ci.updated_at,
		       i.id, i.item_type_id, i.attributes, i.created_at, i.updated_at,
		       it.id, it.name, it.description, it.attributes_schema, it.display_order, it.created_at, it.updated_at
		FROM collection_items ci
		JOIN items i ON i.id = ci.item_id
		JOIN item_types it ON it.id = i.item_type_id
		WHERE %s
		ORDER BY %s
		LIMIT %s OFFSET %s`, where, sortClause, limit, offset)

	rows, err := s.pool.Query(ctx, query, q.args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []model.CollectionItem
	for rows.Next() {
		var ci model.CollectionItem
		var item model.Item
		var it model.ItemType
		if err := rows.Scan(
			&ci.ID, &ci.CollectionID, &ci.ItemID, &ci.Quantity, &ci.Attributes, &ci.CreatedAt, &ci.UpdatedAt,
			&item.ID, &item.ItemTypeID, &item.Attributes, &item.CreatedAt, &item.UpdatedAt,
			&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.DisplayOrder, &it.CreatedAt, &it.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		item.ItemType = &it
		ci.Item = &item
		result = append(result, ci)
	}
	return result, total, rows.Err()
}

// Lots (P7-1): an item may form several lots of a collection; every route
// addresses a lot by its id, always within its collection.
const lotColumns = `id, collection_id, item_id, quantity, attributes, created_at, updated_at`

var (
	// ErrInvalidQuantity: a split leaving no copy behind, or a merge beyond model.MaxQuantity.
	ErrInvalidQuantity = errors.New("invalid quantity")
	// ErrNotMergeable: merging a lot into itself or into a lot of another item.
	ErrNotMergeable = errors.New("lots not mergeable")
)

func scanLot(row pgx.Row) (*model.CollectionItem, error) {
	var ci model.CollectionItem
	err := row.Scan(&ci.ID, &ci.CollectionID, &ci.ItemID, &ci.Quantity, &ci.Attributes, &ci.CreatedAt, &ci.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ci, nil
}

// AddCollectionItem always creates a new lot, even when the item is already in the collection.
func (s *PgStore) AddCollectionItem(ctx context.Context, collectionID uuid.UUID, in model.CollectionItemAdd) (*model.CollectionItem, error) {
	attrs := in.Attributes
	if len(attrs) == 0 {
		attrs = []byte(`{}`)
	}
	quantity := 1
	if in.Quantity != nil {
		quantity = *in.Quantity
	}
	ci, err := scanLot(s.pool.QueryRow(ctx,
		`INSERT INTO collection_items (collection_id, item_id, attributes, quantity)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+lotColumns,
		collectionID, in.ItemID, attrs, quantity))
	if isForeignKeyViolation(err) {
		return nil, ErrNotFound
	}
	return ci, err
}

// UpdateCollectionItem replaces the attributes of a lot; its quantity changes only when given.
func (s *PgStore) UpdateCollectionItem(ctx context.Context, collectionID, lotID uuid.UUID, in model.CollectionItemUpdate) (*model.CollectionItem, error) {
	return scanLot(s.pool.QueryRow(ctx,
		`UPDATE collection_items SET attributes = $3, quantity = COALESCE($4, quantity)
		 WHERE collection_id = $1 AND id = $2
		 RETURNING `+lotColumns,
		collectionID, lotID, in.Attributes, in.Quantity))
}

func (s *PgStore) RemoveCollectionItem(ctx context.Context, collectionID, lotID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM collection_items WHERE collection_id = $1 AND id = $2`,
		collectionID, lotID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SplitCollectionLot moves quantity copies of a lot to a new lot of the same
// item, with attrs (nil: a copy of the lot's). The new lot keeps the date the
// copies were added. At least one copy stays in the original lot.
func (s *PgStore) SplitCollectionLot(ctx context.Context, collectionID, lotID uuid.UUID, quantity int, attrs json.RawMessage) (*model.CollectionItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	lot, err := scanLot(tx.QueryRow(ctx, `SELECT `+lotColumns+` FROM collection_items
		WHERE collection_id = $1 AND id = $2 FOR UPDATE`, collectionID, lotID))
	if err != nil {
		return nil, err
	}
	if quantity < 1 || quantity >= lot.Quantity {
		return nil, ErrInvalidQuantity
	}
	if len(attrs) == 0 {
		attrs = lot.Attributes
	}
	if _, err := tx.Exec(ctx, `UPDATE collection_items SET quantity = quantity - $2 WHERE id = $1`, lotID, quantity); err != nil {
		return nil, err
	}
	created, err := scanLot(tx.QueryRow(ctx,
		`INSERT INTO collection_items (collection_id, item_id, attributes, quantity, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+lotColumns,
		collectionID, lot.ItemID, attrs, quantity, lot.CreatedAt))
	if err != nil {
		return nil, err
	}
	return created, tx.Commit(ctx)
}

// MergeCollectionLots adds the copies of lotID to into (same item, same
// collection), then removes lotID. into keeps its attributes and receives the
// photos of lotID's galleries (the image-gallery properties of the schema).
func (s *PgStore) MergeCollectionLots(ctx context.Context, collectionID, lotID, into uuid.UUID, galleries []string) (*model.CollectionItem, error) {
	if lotID == into {
		return nil, ErrNotMergeable
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Locked in id order: two opposite merges cannot deadlock.
	rows, err := tx.Query(ctx, `SELECT `+lotColumns+` FROM collection_items
		WHERE collection_id = $1 AND id = ANY($2) ORDER BY id FOR UPDATE`, collectionID, []uuid.UUID{lotID, into})
	if err != nil {
		return nil, err
	}
	lots, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*model.CollectionItem, error) { return scanLot(row) })
	if err != nil {
		return nil, err
	}
	if len(lots) != 2 {
		return nil, ErrNotFound
	}
	if lots[0].ItemID != lots[1].ItemID {
		return nil, ErrNotMergeable
	}
	if lots[0].Quantity+lots[1].Quantity > model.MaxQuantity {
		return nil, ErrInvalidQuantity
	}
	if _, err := tx.Exec(ctx, `DELETE FROM collection_items WHERE id = $1`, lotID); err != nil {
		return nil, err
	}
	target, from := lots[0], lots[1]
	if target.ID != into {
		target, from = from, target
	}
	attrs, err := mergeGalleries(target.Attributes, from.Attributes, galleries)
	if err != nil {
		return nil, err
	}
	merged, err := scanLot(tx.QueryRow(ctx,
		`UPDATE collection_items SET quantity = quantity + $2, attributes = $3 WHERE id = $1 RETURNING `+lotColumns,
		into, from.Quantity, attrs))
	if err != nil {
		return nil, err
	}
	return merged, tx.Commit(ctx)
}
func (s *PgStore) ListItemCollections(ctx context.Context, userID, itemID uuid.UUID) ([]model.ItemCollectionMembership, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.user_id, c.collection_type_id, c.name, c.description,
		       c.created_at, c.updated_at,
		       ct.id, ct.name, ct.description, ct.attributes_schema, `+ctItemTypeIDs("ct")+`, ct.created_at, ct.updated_at,
		       (SELECT count(DISTINCT ci2.item_id) FROM collection_items ci2 WHERE ci2.collection_id = c.id) AS item_count,
		       (SELECT COALESCE(sum(ci2.quantity), 0) FROM collection_items ci2 WHERE ci2.collection_id = c.id) AS copy_count,
		       ci.id, ci.quantity, ci.attributes
		FROM collection_items ci
		JOIN collections c ON c.id = ci.collection_id
		JOIN collection_types ct ON ct.id = c.collection_type_id
		WHERE ci.item_id = $1 AND c.user_id = $2
		ORDER BY c.name, ci.created_at, ci.id`, itemID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.ItemCollectionMembership
	for rows.Next() {
		var col model.Collection
		var colType model.CollectionType
		var attrs []byte
		var lotID uuid.UUID
		var quantity int
		if err := rows.Scan(
			&col.ID, &col.UserID, &col.CollectionTypeID, &col.Name, &col.Description,
			&col.CreatedAt, &col.UpdatedAt,
			&colType.ID, &colType.Name, &colType.Description, &colType.AttributesSchema, &colType.ItemTypeIDs, &colType.CreatedAt, &colType.UpdatedAt,
			&col.ItemCount, &col.CopyCount, &lotID, &quantity, &attrs,
		); err != nil {
			return nil, err
		}
		col.CollectionType = &colType
		result = append(result, model.ItemCollectionMembership{
			Collection: col,
			LotID:      lotID,
			Quantity:   quantity,
			Attributes: attrs,
		})
	}
	return result, rows.Err()
}

func (s *PgStore) ListRelatedItemCollections(ctx context.Context, userID, itemID uuid.UUID) ([]model.RelatedItemCollections, error) {
	// One indexed lookup per linked item (idx_relations_source / _target, idx_collection_items_item).
	rows, err := s.pool.Query(ctx, `
		WITH linked AS (
			SELECT target_item_id AS item_id FROM item_relations WHERE source_item_id = $1
			UNION
			SELECT source_item_id FROM item_relations WHERE target_item_id = $1
		)
		SELECT ci.item_id, c.id, c.name, c.collection_type_id, ci.id, ci.quantity, ci.attributes
		FROM linked l
		JOIN collection_items ci ON ci.item_id = l.item_id
		JOIN collections c ON c.id = ci.collection_id AND c.user_id = $2
		ORDER BY ci.item_id, c.name, ci.created_at, ci.id`, itemID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.RelatedItemCollections{}
	for rows.Next() {
		var itemID uuid.UUID
		var col model.RelatedCollection
		if err := rows.Scan(&itemID, &col.ID, &col.Name, &col.CollectionTypeID, &col.LotID, &col.Quantity, &col.Attributes); err != nil {
			return nil, err
		}
		if n := len(result); n > 0 && result[n-1].ItemID == itemID {
			result[n-1].Collections = append(result[n-1].Collections, col)
		} else {
			result = append(result, model.RelatedItemCollections{ItemID: itemID, Collections: []model.RelatedCollection{col}})
		}
	}
	return result, rows.Err()
}

// mergeGalleries adds the photos of the galleries keys of from to those of
// into, category by category, without duplicates; into keeps its default image
// (from's when it has none). Values that are not photo URLs are ignored.
func mergeGalleries(into, from json.RawMessage, keys []string) (json.RawMessage, error) {
	var dst, src map[string]any
	if err := json.Unmarshal(into, &dst); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(from, &src); err != nil {
		return nil, err
	}
	if dst == nil {
		dst = map[string]any{}
	}
	changed := false
	for _, key := range keys {
		in, _ := src[key].(map[string]any)
		if len(in) == 0 {
			continue
		}
		out, _ := dst[key].(map[string]any)
		if out == nil {
			out = map[string]any{}
		}
		for cat, v := range in {
			if cat == "default_image" {
				if img, ok := v.(string); ok && img != "" && out[cat] == nil {
					out[cat], changed = img, true
				}
				continue
			}
			urls, ok := v.([]any)
			if !ok {
				continue
			}
			have, _ := out[cat].([]any)
			for _, u := range urls {
				if url, ok := u.(string); ok && !slices.Contains(have, any(url)) {
					have, changed = append(have, url), true
				}
			}
			out[cat] = have
		}
		dst[key] = out
	}
	if !changed {
		return into, nil
	}
	return json.Marshal(dst)
}
