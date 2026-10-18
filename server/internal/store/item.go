package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

func (s *PgStore) ListItems(ctx context.Context, pg model.Pagination, f ItemFilters) ([]model.Item, int, error) {
	q, err := buildItemQuery(f)
	if err != nil {
		return nil, 0, err
	}
	whereClause := q.whereClause()

	var total int
	err = s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT count(*) FROM items i WHERE %s`, whereClause),
		q.args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	sortClause := q.orderBy(f) // may add the sort key to q.args
	limit, offset := q.arg(pg.PerPage), q.arg(pg.Offset())

	query := fmt.Sprintf(`
		SELECT i.id, i.item_type_id, i.attributes, i.created_at, i.updated_at,
		       it.id, it.name, it.description, it.attributes_schema, it.display_order, it.created_at, it.updated_at
		FROM items i
		JOIN item_types it ON it.id = i.item_type_id
		WHERE %s
		ORDER BY %s
		LIMIT %s OFFSET %s`, whereClause, sortClause, limit, offset)

	rows, err := s.pool.Query(ctx, query, q.args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []model.Item
	for rows.Next() {
		var item model.Item
		var it model.ItemType
		if err := rows.Scan(
			&item.ID, &item.ItemTypeID, &item.Attributes, &item.CreatedAt, &item.UpdatedAt,
			&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.DisplayOrder, &it.CreatedAt, &it.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		item.ItemType = &it
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Tags of the whole page in one query.
	ids := make([]uuid.UUID, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	tagsByItem, err := s.getItemsTags(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].Tags = tagsByItem[items[i].ID]
	}

	return items, total, nil
}

func (s *PgStore) GetItem(ctx context.Context, id uuid.UUID) (*model.Item, error) {
	var item model.Item
	var it model.ItemType
	err := s.pool.QueryRow(ctx,
		`SELECT i.id, i.item_type_id, i.attributes, i.created_at, i.updated_at,
		        it.id, it.name, it.description, it.attributes_schema, it.display_order, it.created_at, it.updated_at
		 FROM items i
		 JOIN item_types it ON it.id = i.item_type_id
		 WHERE i.id = $1`, id,
	).Scan(
		&item.ID, &item.ItemTypeID, &item.Attributes, &item.CreatedAt, &item.UpdatedAt,
		&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.DisplayOrder, &it.CreatedAt, &it.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	item.ItemType = &it

	tags, err := s.getItemTags(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	item.Tags = tags

	return &item, nil
}

func (s *PgStore) CreateItem(ctx context.Context, in model.ItemCreate) (*model.Item, error) {
	tx, err := s.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // sans effet après Commit

	attrs := in.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}

	var item model.Item
	err = tx.QueryRow(ctx,
		`INSERT INTO items (item_type_id, attributes)
		 VALUES ($1, $2)
		 RETURNING id, item_type_id, attributes, created_at, updated_at`,
		in.ItemTypeID, attrs,
	).Scan(&item.ID, &item.ItemTypeID, &item.Attributes, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Handle tags
	if len(in.Tags) > 0 {
		if err := s.upsertItemTags(ctx, tx, item.ID, in.Tags); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetItem(ctx, item.ID)
}

func (s *PgStore) UpdateItem(ctx context.Context, id uuid.UUID, in model.ItemUpdate) (*model.Item, error) {
	tx, err := s.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // sans effet après Commit

	if len(in.Attributes) > 0 {
		ct, err := tx.Exec(ctx,
			`UPDATE items SET attributes = $2 WHERE id = $1`,
			id, in.Attributes,
		)
		if err != nil {
			return nil, err
		}
		if ct.RowsAffected() == 0 {
			return nil, ErrNotFound
		}
	}

	if in.Tags != nil {
		// Delete existing tags
		_, err := tx.Exec(ctx, `DELETE FROM item_tags WHERE item_id = $1`, id)
		if err != nil {
			return nil, err
		}
		if len(*in.Tags) > 0 {
			if err := s.upsertItemTags(ctx, tx, id, *in.Tags); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetItem(ctx, id)
}

func (s *PgStore) DeleteItem(ctx context.Context, id uuid.UUID) error {
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // sans effet après Commit

	// Delete relations
	_, err = tx.Exec(ctx, `DELETE FROM item_relations WHERE source_item_id = $1 OR target_item_id = $1`, id)
	if err != nil {
		return err
	}

	// Delete collection items
	_, err = tx.Exec(ctx, `DELETE FROM collection_items WHERE item_id = $1`, id)
	if err != nil {
		return err
	}

	// Delete item tags
	_, err = tx.Exec(ctx, `DELETE FROM item_tags WHERE item_id = $1`, id)
	if err != nil {
		return err
	}

	ct, err := tx.Exec(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	return tx.Commit(ctx)
}

func (s *PgStore) getItemTags(ctx context.Context, itemID uuid.UUID) ([]model.Tag, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT t.id, t.name FROM tags t
		 JOIN item_tags it ON it.tag_id = t.id
		 WHERE it.item_id = $1
		 ORDER BY t.name`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *PgStore) upsertItemTags(ctx context.Context, tx pgx.Tx, itemID uuid.UUID, tagNames []string) error {
	for _, name := range tagNames {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		var tagID uuid.UUID
		err := tx.QueryRow(ctx,
			`INSERT INTO tags (name) VALUES ($1)
			 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			 RETURNING id`, name,
		).Scan(&tagID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO item_tags (item_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			itemID, tagID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// getItemsTags loads the tags of several items in one query, keyed by item id.
// Items without tags are absent from the map (nil slice → omitted in JSON).
func (s *PgStore) getItemsTags(ctx context.Context, itemIDs []uuid.UUID) (map[uuid.UUID][]model.Tag, error) {
	result := make(map[uuid.UUID][]model.Tag, len(itemIDs))
	if len(itemIDs) == 0 {
		return result, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT it.item_id, t.id, t.name
		 FROM item_tags it
		 JOIN tags t ON t.id = it.tag_id
		 WHERE it.item_id = ANY($1)
		 ORDER BY it.item_id, t.name`, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var itemID uuid.UUID
		var t model.Tag
		if err := rows.Scan(&itemID, &t.ID, &t.Name); err != nil {
			return nil, err
		}
		result[itemID] = append(result[itemID], t)
	}
	return result, rows.Err()
}
