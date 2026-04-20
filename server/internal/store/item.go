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
	where := []string{"1=1"}
	args := []interface{}{}
	argN := 1

	if f.ItemTypeID != nil {
		where = append(where, fmt.Sprintf("i.item_type_id = $%d", argN))
		args = append(args, *f.ItemTypeID)
		argN++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("i.attributes::text ILIKE $%d", argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.Tag != "" {
		where = append(where, fmt.Sprintf(
			`EXISTS (SELECT 1 FROM item_tags it2 JOIN tags t ON t.id = it2.tag_id WHERE it2.item_id = i.id AND lower(t.name) = lower($%d))`, argN))
		args = append(args, f.Tag)
		argN++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT count(*) FROM items i WHERE %s`, whereClause),
		countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	sortClause := buildSortClause(f.Sort, "i.created_at DESC", map[string]string{
		"created_at": "i.created_at",
		"updated_at": "i.updated_at",
	})

	args = append(args, pg.PerPage, pg.Offset())

	query := fmt.Sprintf(`
		SELECT i.id, i.item_type_id, i.attributes, i.created_at, i.updated_at,
		       it.id, it.name, it.description, it.attributes_schema, it.created_at, it.updated_at
		FROM items i
		JOIN item_types it ON it.id = i.item_type_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, whereClause, sortClause, argN, argN+1)

	rows, err := s.pool.Query(ctx, query, args...)
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
			&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.CreatedAt, &it.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		item.ItemType = &it
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Load tags for each item
	for i := range items {
		tags, err := s.getItemTags(ctx, items[i].ID)
		if err != nil {
			return nil, 0, err
		}
		items[i].Tags = tags
	}

	return items, total, nil
}

func (s *PgStore) GetItem(ctx context.Context, id uuid.UUID) (*model.Item, error) {
	var item model.Item
	var it model.ItemType
	err := s.pool.QueryRow(ctx,
		`SELECT i.id, i.item_type_id, i.attributes, i.created_at, i.updated_at,
		        it.id, it.name, it.description, it.attributes_schema, it.created_at, it.updated_at
		 FROM items i
		 JOIN item_types it ON it.id = i.item_type_id
		 WHERE i.id = $1`, id,
	).Scan(
		&item.ID, &item.ItemTypeID, &item.Attributes, &item.CreatedAt, &item.UpdatedAt,
		&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.CreatedAt, &it.UpdatedAt,
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
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

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
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

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
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

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
