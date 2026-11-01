package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// ListItemRelations returns every relation of an item (source or target), for the
// callers that need them all (connectors matching through a relation). The API uses
// ListItemRelationsPage.
func (s *PgStore) ListItemRelations(ctx context.Context, itemID uuid.UUID, relationTypeID *uuid.UUID) ([]model.ItemRelation, error) {
	rels, _, err := s.listItemRelations(ctx, itemID, relationTypeID, nil)
	return rels, err
}

// ListItemRelationsPage returns one page of the relations of an item and their total.
func (s *PgStore) ListItemRelationsPage(ctx context.Context, itemID uuid.UUID, relationTypeID *uuid.UUID, pg model.Pagination) ([]model.ItemRelation, int, error) {
	return s.listItemRelations(ctx, itemID, relationTypeID, &pg)
}

// listItemRelations lists the relations of an item, sorted by relation type name,
// then by the `order` attribute, then by creation. Both ends are embedded with their
// attributes; the RelationType and the ItemTypes are not (ids only): their schemas
// were repeated on every row and the clients cache the meta-types (audit Q-001).
func (s *PgStore) listItemRelations(ctx context.Context, itemID uuid.UUID, relationTypeID *uuid.UUID, pg *model.Pagination) ([]model.ItemRelation, int, error) {
	where := `(ir.source_item_id = $1 OR ir.target_item_id = $1)`
	args := []any{itemID}
	if relationTypeID != nil {
		where += ` AND ir.relation_type_id = $2`
		args = append(args, *relationTypeID)
	}

	total := 0
	if pg != nil {
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM item_relations ir WHERE `+where, args...).Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	query := `
		SELECT ir.id, ir.relation_type_id, ir.source_item_id, ir.target_item_id,
		       ir.attributes, ir.created_at, ir.updated_at,
		       si.id, si.item_type_id, si.attributes, si.created_at, si.updated_at,
		       ti.id, ti.item_type_id, ti.attributes, ti.created_at, ti.updated_at
		FROM item_relations ir
		JOIN relation_types rt ON rt.id = ir.relation_type_id
		JOIN items si ON si.id = ir.source_item_id
		JOIN items ti ON ti.id = ir.target_item_id
		WHERE ` + where + `
		ORDER BY rt.name,
		         CASE WHEN jsonb_typeof(ir.attributes -> 'order') = 'number' THEN (ir.attributes ->> 'order')::numeric END NULLS LAST,
		         ir.created_at, ir.id`
	if pg != nil {
		query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
		args = append(args, pg.PerPage, pg.Offset())
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := []model.ItemRelation{}
	for rows.Next() {
		var ir model.ItemRelation
		var si, ti model.Item
		if err := rows.Scan(
			&ir.ID, &ir.RelationTypeID, &ir.SourceItemID, &ir.TargetItemID,
			&ir.Attributes, &ir.CreatedAt, &ir.UpdatedAt,
			&si.ID, &si.ItemTypeID, &si.Attributes, &si.CreatedAt, &si.UpdatedAt,
			&ti.ID, &ti.ItemTypeID, &ti.Attributes, &ti.CreatedAt, &ti.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		ir.SourceItem = &si
		ir.TargetItem = &ti
		result = append(result, ir)
	}
	if pg == nil {
		total = len(result)
	}
	return result, total, rows.Err()
}

func (s *PgStore) GetRelation(ctx context.Context, id uuid.UUID) (*model.ItemRelation, error) {
	var ir model.ItemRelation
	var rt model.RelationType
	err := s.pool.QueryRow(ctx,
		`SELECT ir.id, ir.relation_type_id, ir.source_item_id, ir.target_item_id,
		        ir.attributes, ir.created_at, ir.updated_at,
		        rt.id, rt.name, rt.description,
		        rt.attributes_schema, rt.created_at, rt.updated_at
		 FROM item_relations ir
		 JOIN relation_types rt ON rt.id = ir.relation_type_id
		 WHERE ir.id = $1`, id,
	).Scan(
		&ir.ID, &ir.RelationTypeID, &ir.SourceItemID, &ir.TargetItemID,
		&ir.Attributes, &ir.CreatedAt, &ir.UpdatedAt,
		&rt.ID, &rt.Name, &rt.Description,
		&rt.AttributesSchema, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	ir.RelationType = &rt
	return &ir, nil
}

func (s *PgStore) CreateRelation(ctx context.Context, in model.ItemRelationCreate) (*model.ItemRelation, error) {
	attrs := in.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}

	tx, err := s.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		in.RelationTypeID, in.SourceItemID, in.TargetItemID, attrs,
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
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetRelation(ctx, id)
}

func (s *PgStore) UpdateRelation(ctx context.Context, id uuid.UUID, in model.ItemRelationUpdate) (*model.ItemRelation, error) {
	tx, err := s.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ct, err := tx.Exec(ctx,
		`UPDATE item_relations SET attributes = $2 WHERE id = $1`,
		id, in.Attributes,
	)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetRelation(ctx, id)
}

func (s *PgStore) DeleteRelation(ctx context.Context, id uuid.UUID) error {
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ct, err := tx.Exec(ctx, `DELETE FROM item_relations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}
