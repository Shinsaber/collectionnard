package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// collectionColumns / scanCollection: every collection read (with its CollectionType and item count).
var collectionColumns = `c.id, c.user_id, c.collection_type_id, c.name, c.description, c.visibility, c.hidden_fields,
		c.created_at, c.updated_at,
		ct.id, ct.name, ct.description, ct.attributes_schema, ` + ctItemTypeIDs("ct") + `, ct.created_at, ct.updated_at,
		(SELECT count(*) FROM collection_items ci WHERE ci.collection_id = c.id)`

const collectionFrom = ` FROM collections c JOIN collection_types ct ON ct.id = c.collection_type_id`

func scanCollection(row pgx.Row) (*model.Collection, error) {
	var col model.Collection
	var ct model.CollectionType
	if err := row.Scan(
		&col.ID, &col.UserID, &col.CollectionTypeID, &col.Name, &col.Description, &col.Visibility, &col.HiddenFields,
		&col.CreatedAt, &col.UpdatedAt,
		&ct.ID, &ct.Name, &ct.Description, &ct.AttributesSchema, &ct.ItemTypeIDs, &ct.CreatedAt, &ct.UpdatedAt,
		&col.ItemCount,
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

func (s *PgStore) ListCollectionItems(ctx context.Context, collectionID uuid.UUID, pg model.Pagination, itemTypeID *uuid.UUID, sort string) ([]model.CollectionItem, int, error) {
	where := "ci.collection_id = $1"
	args := []any{collectionID}
	argN := 2

	if itemTypeID != nil {
		where += fmt.Sprintf(" AND i.item_type_id = $%d", argN)
		args = append(args, *itemTypeID)
		argN++
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT count(*) FROM collection_items ci JOIN items i ON i.id = ci.item_id WHERE %s`, where),
		countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	sortClause := buildSortClause(sort, "ci.created_at DESC", map[string]string{
		"created_at": "ci.created_at",
		"updated_at": "ci.updated_at",
	})

	args = append(args, pg.PerPage, pg.Offset())
	query := fmt.Sprintf(`
		SELECT ci.id, ci.collection_id, ci.item_id, ci.attributes, ci.created_at, ci.updated_at,
		       i.id, i.item_type_id, i.attributes, i.created_at, i.updated_at,
		       it.id, it.name, it.description, it.attributes_schema, it.display_order, it.created_at, it.updated_at
		FROM collection_items ci
		JOIN items i ON i.id = ci.item_id
		JOIN item_types it ON it.id = i.item_type_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, where, sortClause, argN, argN+1)

	rows, err := s.pool.Query(ctx, query, args...)
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
			&ci.ID, &ci.CollectionID, &ci.ItemID, &ci.Attributes, &ci.CreatedAt, &ci.UpdatedAt,
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

func (s *PgStore) AddCollectionItem(ctx context.Context, collectionID uuid.UUID, in model.CollectionItemAdd) (*model.CollectionItem, error) {
	attrs := in.Attributes
	if len(attrs) == 0 {
		attrs = []byte(`{}`)
	}
	var ci model.CollectionItem
	err := s.pool.QueryRow(ctx,
		`INSERT INTO collection_items (collection_id, item_id, attributes)
		 VALUES ($1, $2, $3)
		 RETURNING id, collection_id, item_id, attributes, created_at, updated_at`,
		collectionID, in.ItemID, attrs,
	).Scan(&ci.ID, &ci.CollectionID, &ci.ItemID, &ci.Attributes, &ci.CreatedAt, &ci.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ci, nil
}

func (s *PgStore) UpdateCollectionItem(ctx context.Context, collectionID, itemID uuid.UUID, in model.CollectionItemUpdate) (*model.CollectionItem, error) {
	var ci model.CollectionItem
	err := s.pool.QueryRow(ctx,
		`UPDATE collection_items SET attributes = $3
		 WHERE collection_id = $1 AND item_id = $2
		 RETURNING id, collection_id, item_id, attributes, created_at, updated_at`,
		collectionID, itemID, in.Attributes,
	).Scan(&ci.ID, &ci.CollectionID, &ci.ItemID, &ci.Attributes, &ci.CreatedAt, &ci.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ci, nil
}

func (s *PgStore) RemoveCollectionItem(ctx context.Context, collectionID, itemID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM collection_items WHERE collection_id = $1 AND item_id = $2`,
		collectionID, itemID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PgStore) ListItemCollections(ctx context.Context, userID, itemID uuid.UUID) ([]model.ItemCollectionMembership, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.user_id, c.collection_type_id, c.name, c.description,
		       c.created_at, c.updated_at,
		       ct.id, ct.name, ct.description, ct.attributes_schema, `+ctItemTypeIDs("ct")+`, ct.created_at, ct.updated_at,
		       (SELECT count(*) FROM collection_items ci2 WHERE ci2.collection_id = c.id) AS item_count,
		       ci.attributes
		FROM collection_items ci
		JOIN collections c ON c.id = ci.collection_id
		JOIN collection_types ct ON ct.id = c.collection_type_id
		WHERE ci.item_id = $1 AND c.user_id = $2
		ORDER BY c.name`, itemID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.ItemCollectionMembership
	for rows.Next() {
		var col model.Collection
		var colType model.CollectionType
		var attrs []byte
		if err := rows.Scan(
			&col.ID, &col.UserID, &col.CollectionTypeID, &col.Name, &col.Description,
			&col.CreatedAt, &col.UpdatedAt,
			&colType.ID, &colType.Name, &colType.Description, &colType.AttributesSchema, &colType.ItemTypeIDs, &colType.CreatedAt, &colType.UpdatedAt,
			&col.ItemCount, &attrs,
		); err != nil {
			return nil, err
		}
		col.CollectionType = &colType
		result = append(result, model.ItemCollectionMembership{
			Collection: col,
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
		SELECT ci.item_id, c.id, c.name, c.collection_type_id, ci.attributes
		FROM linked l
		JOIN collection_items ci ON ci.item_id = l.item_id
		JOIN collections c ON c.id = ci.collection_id AND c.user_id = $2
		ORDER BY ci.item_id, c.name`, itemID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.RelatedItemCollections{}
	for rows.Next() {
		var itemID uuid.UUID
		var col model.RelatedCollection
		if err := rows.Scan(&itemID, &col.ID, &col.Name, &col.CollectionTypeID, &col.Attributes); err != nil {
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
