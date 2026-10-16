package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

func (s *PgStore) ListItemTypes(ctx context.Context, pg model.Pagination) ([]model.ItemType, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM item_types`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, attributes_schema, display_order, created_at, updated_at
		 FROM item_types ORDER BY display_order, name
		 LIMIT $1 OFFSET $2`, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []model.ItemType
	for rows.Next() {
		var it model.ItemType
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.DisplayOrder, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}

func (s *PgStore) GetItemType(ctx context.Context, id uuid.UUID) (*model.ItemType, error) {
	var it model.ItemType
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, attributes_schema, display_order, created_at, updated_at
		 FROM item_types WHERE id = $1`, id,
	).Scan(&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.DisplayOrder, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &it, nil
}

func (s *PgStore) CreateItemType(ctx context.Context, in model.ItemTypeCreate) (*model.ItemType, error) {
	var it model.ItemType
	err := s.pool.QueryRow(ctx,
		`INSERT INTO item_types (name, description, attributes_schema, display_order)
		 VALUES ($1, $2, $3, COALESCE($4, 100))
		 RETURNING id, name, description, attributes_schema, display_order, created_at, updated_at`,
		in.Name, in.Description, in.AttributesSchema, in.DisplayOrder,
	).Scan(&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.DisplayOrder, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &it, nil
}

func (s *PgStore) UpdateItemType(ctx context.Context, id uuid.UUID, in model.ItemTypeUpdate) (*model.ItemType, error) {
	var it model.ItemType
	err := s.pool.QueryRow(ctx,
		`UPDATE item_types SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			attributes_schema = COALESCE($4, attributes_schema),
			display_order = COALESCE($5, display_order)
		 WHERE id = $1
		 RETURNING id, name, description, attributes_schema, display_order, created_at, updated_at`,
		id, in.Name, in.Description, nullableJSON(in.AttributesSchema), in.DisplayOrder,
	).Scan(&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.DisplayOrder, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &it, nil
}

func (s *PgStore) DeleteItemType(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM item_types WHERE id = $1`, id)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrConflict
		}
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
