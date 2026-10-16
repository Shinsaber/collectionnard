package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// ctItemTypeIDs is the SQL expression of the item types allowed by the collection
// type aliased `alias` (migration 006; empty array = every type is allowed).
func ctItemTypeIDs(alias string) string {
	return `COALESCE((SELECT array_agg(j.item_type_id ORDER BY j.item_type_id)
		FROM collection_type_item_types j WHERE j.collection_type_id = ` + alias + `.id), '{}')`
}

func (s *PgStore) ListCollectionTypes(ctx context.Context, pg model.Pagination) ([]model.CollectionType, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM collection_types`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT ct.id, ct.name, ct.description, ct.attributes_schema, `+ctItemTypeIDs("ct")+`, ct.created_at, ct.updated_at
		 FROM collection_types ct ORDER BY ct.name
		 LIMIT $1 OFFSET $2`, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []model.CollectionType
	for rows.Next() {
		var ct model.CollectionType
		if err := rows.Scan(&ct.ID, &ct.Name, &ct.Description, &ct.AttributesSchema, &ct.ItemTypeIDs, &ct.CreatedAt, &ct.UpdatedAt); err != nil {
			return nil, 0, err
		}
		result = append(result, ct)
	}
	return result, total, rows.Err()
}

func (s *PgStore) GetCollectionType(ctx context.Context, id uuid.UUID) (*model.CollectionType, error) {
	var ct model.CollectionType
	err := s.pool.QueryRow(ctx,
		`SELECT ct.id, ct.name, ct.description, ct.attributes_schema, `+ctItemTypeIDs("ct")+`, ct.created_at, ct.updated_at
		 FROM collection_types ct WHERE ct.id = $1`, id,
	).Scan(&ct.ID, &ct.Name, &ct.Description, &ct.AttributesSchema, &ct.ItemTypeIDs, &ct.CreatedAt, &ct.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ct, nil
}

func (s *PgStore) CreateCollectionType(ctx context.Context, in model.CollectionTypeCreate) (*model.CollectionType, error) {
	schema := in.AttributesSchema
	if len(schema) == 0 {
		schema = []byte(`{}`)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // sans effet après Commit

	var id uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO collection_types (name, description, attributes_schema)
		 VALUES ($1, $2, $3) RETURNING id`,
		in.Name, in.Description, schema,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	if err := setCollectionTypeItemTypes(ctx, tx, id, in.ItemTypeIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetCollectionType(ctx, id)
}

func (s *PgStore) UpdateCollectionType(ctx context.Context, id uuid.UUID, in model.CollectionTypeUpdate) (*model.CollectionType, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // sans effet après Commit

	ct, err := tx.Exec(ctx,
		`UPDATE collection_types SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			attributes_schema = COALESCE($4, attributes_schema)
		 WHERE id = $1`,
		id, in.Name, in.Description, nullableJSON(in.AttributesSchema),
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
	if in.ItemTypeIDs != nil {
		if err := setCollectionTypeItemTypes(ctx, tx, id, *in.ItemTypeIDs); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetCollectionType(ctx, id)
}

// setCollectionTypeItemTypes replaces the allowed item types of a collection
// type (empty = every type). An unknown item type gives ErrNotFound.
func setCollectionTypeItemTypes(ctx context.Context, tx pgx.Tx, ctID uuid.UUID, ids []uuid.UUID) error {
	if _, err := tx.Exec(ctx, `DELETE FROM collection_type_item_types WHERE collection_type_id = $1`, ctID); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx,
		`INSERT INTO collection_type_item_types (collection_type_id, item_type_id)
		 SELECT $1, t FROM unnest($2::uuid[]) AS t ON CONFLICT DO NOTHING`, ctID, ids)
	if isForeignKeyViolation(err) {
		return ErrNotFound
	}
	return err
}

func (s *PgStore) DeleteCollectionType(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM collection_types WHERE id = $1`, id)
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
