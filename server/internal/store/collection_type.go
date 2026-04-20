package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

func (s *PgStore) ListCollectionTypes(ctx context.Context, pg model.Pagination) ([]model.CollectionType, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM collection_types`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, attributes_schema, created_at, updated_at
		 FROM collection_types ORDER BY name
		 LIMIT $1 OFFSET $2`, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []model.CollectionType
	for rows.Next() {
		var ct model.CollectionType
		if err := rows.Scan(&ct.ID, &ct.Name, &ct.Description, &ct.AttributesSchema, &ct.CreatedAt, &ct.UpdatedAt); err != nil {
			return nil, 0, err
		}
		result = append(result, ct)
	}
	return result, total, rows.Err()
}

func (s *PgStore) GetCollectionType(ctx context.Context, id uuid.UUID) (*model.CollectionType, error) {
	var ct model.CollectionType
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, attributes_schema, created_at, updated_at
		 FROM collection_types WHERE id = $1`, id,
	).Scan(&ct.ID, &ct.Name, &ct.Description, &ct.AttributesSchema, &ct.CreatedAt, &ct.UpdatedAt)
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
	var ct model.CollectionType
	err := s.pool.QueryRow(ctx,
		`INSERT INTO collection_types (name, description, attributes_schema)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, description, attributes_schema, created_at, updated_at`,
		in.Name, in.Description, schema,
	).Scan(&ct.ID, &ct.Name, &ct.Description, &ct.AttributesSchema, &ct.CreatedAt, &ct.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &ct, nil
}

func (s *PgStore) UpdateCollectionType(ctx context.Context, id uuid.UUID, in model.CollectionTypeUpdate) (*model.CollectionType, error) {
	var ct model.CollectionType
	err := s.pool.QueryRow(ctx,
		`UPDATE collection_types SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			attributes_schema = COALESCE($4, attributes_schema)
		 WHERE id = $1
		 RETURNING id, name, description, attributes_schema, created_at, updated_at`,
		id, in.Name, in.Description, nullableJSON(in.AttributesSchema),
	).Scan(&ct.ID, &ct.Name, &ct.Description, &ct.AttributesSchema, &ct.CreatedAt, &ct.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &ct, nil
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
