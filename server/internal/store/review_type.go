package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

const reviewTypeColumns = `t.id, t.name, t.description, t.attributes_schema, t.context_label,
	COALESCE((SELECT array_agg(j.relation_type_id ORDER BY j.relation_type_id)
		FROM review_type_context_relations j WHERE j.review_type_id = t.id), '{}'),
	t.created_at, t.updated_at`

func scanReviewType(row pgx.Row) (*model.ReviewType, error) {
	var t model.ReviewType
	err := row.Scan(&t.ID, &t.Name, &t.Description, &t.AttributesSchema, &t.ContextLabel, &t.ContextRelationTypeIDs,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// contextLabel stores an empty label as NULL.
func contextLabel(label map[string]string) any {
	if len(label) == 0 {
		return nil
	}
	raw, _ := json.Marshal(label)
	return raw
}

func (s *PgStore) ListReviewTypes(ctx context.Context, pg model.Pagination) ([]model.ReviewType, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM review_types`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+reviewTypeColumns+` FROM review_types t ORDER BY t.name LIMIT $1 OFFSET $2`, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.ReviewType
	for rows.Next() {
		t, err := scanReviewType(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

func (s *PgStore) GetReviewType(ctx context.Context, id uuid.UUID) (*model.ReviewType, error) {
	t, err := scanReviewType(s.pool.QueryRow(ctx, `SELECT `+reviewTypeColumns+` FROM review_types t WHERE t.id = $1`, id))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *PgStore) CreateReviewType(ctx context.Context, in model.ReviewTypeCreate) (*model.ReviewType, error) {
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
		`INSERT INTO review_types (name, description, attributes_schema, context_label) VALUES ($1, $2, $3, $4)
		 RETURNING id`, in.Name, in.Description, schema, contextLabel(in.ContextLabel)).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	if _, err := setReviewTypeContexts(ctx, tx, id, in.ContextRelationTypeIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetReviewType(ctx, id)
}

func (s *PgStore) UpdateReviewType(ctx context.Context, id uuid.UUID, in model.ReviewTypeUpdate) (*model.ReviewType, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // sans effet après Commit

	var label any
	if in.ContextLabel != nil {
		label = contextLabel(*in.ContextLabel)
	}
	ct, err := tx.Exec(ctx,
		`UPDATE review_types SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			attributes_schema = COALESCE($4, attributes_schema),
			context_label = CASE WHEN $5 THEN $6::jsonb ELSE context_label END
		 WHERE id = $1`,
		id, in.Name, in.Description, nullableJSON(in.AttributesSchema), in.ContextLabel != nil, label)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if in.ContextRelationTypeIDs != nil {
		if _, err := setReviewTypeContexts(ctx, tx, id, *in.ContextRelationTypeIDs); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetReviewType(ctx, id)
}

// setReviewTypeContexts replaces the context relations of a review type; the
// relation types unknown here are left out (a domain whose relation is not
// installed yet). Reports whether the list changed.
func setReviewTypeContexts(ctx context.Context, tx pgx.Tx, id uuid.UUID, ids []uuid.UUID) (bool, error) {
	before, err := uuidColumn(ctx, tx, `SELECT relation_type_id FROM review_type_context_relations WHERE review_type_id = $1`, id)
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM review_type_context_relations WHERE review_type_id = $1`, id); err != nil {
		return false, err
	}
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO review_type_context_relations (review_type_id, relation_type_id)
			 SELECT $1, rt.id FROM relation_types rt WHERE rt.id = ANY($2) ON CONFLICT DO NOTHING`, id, ids); err != nil {
			return false, err
		}
	}
	after, err := uuidColumn(ctx, tx, `SELECT relation_type_id FROM review_type_context_relations WHERE review_type_id = $1`, id)
	if err != nil {
		return false, err
	}
	return !sameUUIDs(before, after), nil
}

// DeleteReviewType refuses (ErrConflict) a type used by an item type or a review.
func (s *PgStore) DeleteReviewType(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM review_types WHERE id = $1`, id)
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
