package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

func (s *PgStore) ListRelationTypes(ctx context.Context, pg model.Pagination, sourceTypeID, targetTypeID *uuid.UUID) ([]model.RelationType, int, error) {
	where := "1=1"
	args := []any{}
	argN := 1

	if sourceTypeID != nil {
		where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM relation_type_sources rts WHERE rts.relation_type_id = rt.id AND rts.item_type_id = $%d)", argN)
		args = append(args, *sourceTypeID)
		argN++
	}
	if targetTypeID != nil {
		where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM relation_type_targets rtt WHERE rtt.relation_type_id = rt.id AND rtt.item_type_id = $%d)", argN)
		args = append(args, *targetTypeID)
		argN++
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT count(*) FROM relation_types rt WHERE %s`, where),
		countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, pg.PerPage, pg.Offset())
	query := fmt.Sprintf(`
		SELECT rt.id, rt.name, rt.description,
		       rt.attributes_schema, rt.created_at, rt.updated_at
		FROM relation_types rt
		WHERE %s
		ORDER BY rt.name
		LIMIT $%d OFFSET $%d`, where, argN, argN+1)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []model.RelationType
	for rows.Next() {
		var rt model.RelationType
		if err := rows.Scan(
			&rt.ID, &rt.Name, &rt.Description,
			&rt.AttributesSchema, &rt.CreatedAt, &rt.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, rt)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Load junction data for all results
	for i := range result {
		if err := s.loadRelationTypeJunctions(ctx, &result[i]); err != nil {
			return nil, 0, err
		}
	}

	return result, total, nil
}

func (s *PgStore) GetRelationType(ctx context.Context, id uuid.UUID) (*model.RelationType, error) {
	var rt model.RelationType
	err := s.pool.QueryRow(ctx,
		`SELECT rt.id, rt.name, rt.description,
		        rt.attributes_schema, rt.created_at, rt.updated_at
		 FROM relation_types rt
		 WHERE rt.id = $1`, id,
	).Scan(
		&rt.ID, &rt.Name, &rt.Description,
		&rt.AttributesSchema, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.loadRelationTypeJunctions(ctx, &rt); err != nil {
		return nil, err
	}
	return &rt, nil
}

func (s *PgStore) CreateRelationType(ctx context.Context, in model.RelationTypeCreate) (*model.RelationType, error) {
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
		`INSERT INTO relation_types (name, description, attributes_schema)
		 VALUES ($1, $2, $3) RETURNING id`,
		in.Name, in.Description, schema,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}

	if err := insertJunctions(ctx, tx, "relation_type_sources", id, in.SourceItemTypeIDs); err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := insertJunctions(ctx, tx, "relation_type_targets", id, in.TargetItemTypeIDs); err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetRelationType(ctx, id)
}

func (s *PgStore) UpdateRelationType(ctx context.Context, id uuid.UUID, in model.RelationTypeUpdate) (*model.RelationType, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // sans effet après Commit

	ct, err := tx.Exec(ctx,
		`UPDATE relation_types SET
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

	if in.SourceItemTypeIDs != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM relation_type_sources WHERE relation_type_id = $1`, id); err != nil {
			return nil, err
		}
		if err := insertJunctions(ctx, tx, "relation_type_sources", id, in.SourceItemTypeIDs); err != nil {
			if isForeignKeyViolation(err) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	}
	if in.TargetItemTypeIDs != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM relation_type_targets WHERE relation_type_id = $1`, id); err != nil {
			return nil, err
		}
		if err := insertJunctions(ctx, tx, "relation_type_targets", id, in.TargetItemTypeIDs); err != nil {
			if isForeignKeyViolation(err) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetRelationType(ctx, id)
}

func (s *PgStore) DeleteRelationType(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM relation_types WHERE id = $1`, id)
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

// loadRelationTypeJunctions populates SourceItemTypeIDs/TargetItemTypeIDs and
// SourceItemTypes/TargetItemTypes from junction tables.
func (s *PgStore) loadRelationTypeJunctions(ctx context.Context, rt *model.RelationType) error {
	// Sources
	rows, err := s.pool.Query(ctx,
		`SELECT it.id, it.name, it.description, it.attributes_schema, it.created_at, it.updated_at
		 FROM relation_type_sources rts
		 JOIN item_types it ON it.id = rts.item_type_id
		 WHERE rts.relation_type_id = $1
		 ORDER BY it.name`, rt.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	rt.SourceItemTypeIDs = []uuid.UUID{}
	rt.SourceItemTypes = []model.ItemType{}
	for rows.Next() {
		var it model.ItemType
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return err
		}
		rt.SourceItemTypeIDs = append(rt.SourceItemTypeIDs, it.ID)
		rt.SourceItemTypes = append(rt.SourceItemTypes, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Targets
	rows2, err := s.pool.Query(ctx,
		`SELECT it.id, it.name, it.description, it.attributes_schema, it.created_at, it.updated_at
		 FROM relation_type_targets rtt
		 JOIN item_types it ON it.id = rtt.item_type_id
		 WHERE rtt.relation_type_id = $1
		 ORDER BY it.name`, rt.ID)
	if err != nil {
		return err
	}
	defer rows2.Close()
	rt.TargetItemTypeIDs = []uuid.UUID{}
	rt.TargetItemTypes = []model.ItemType{}
	for rows2.Next() {
		var it model.ItemType
		if err := rows2.Scan(&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return err
		}
		rt.TargetItemTypeIDs = append(rt.TargetItemTypeIDs, it.ID)
		rt.TargetItemTypes = append(rt.TargetItemTypes, it)
	}
	return rows2.Err()
}

// insertJunctions inserts rows into a junction table (relation_type_sources or relation_type_targets).
func insertJunctions(ctx context.Context, tx pgx.Tx, table string, rtID uuid.UUID, typeIDs []uuid.UUID) error {
	for _, tid := range typeIDs {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf(`INSERT INTO %s (relation_type_id, item_type_id) VALUES ($1, $2)`, table),
			rtID, tid,
		); err != nil {
			return err
		}
	}
	return nil
}
