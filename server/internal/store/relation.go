package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

func (s *PgStore) ListItemRelations(ctx context.Context, itemID uuid.UUID, relationTypeID *uuid.UUID) ([]model.ItemRelation, error) {
	query := `
		SELECT ir.id, ir.relation_type_id, ir.source_item_id, ir.target_item_id,
		       ir.attributes, ir.created_at, ir.updated_at,
		       rt.id, rt.name, rt.description,
		       rt.attributes_schema, rt.created_at, rt.updated_at,
		       si.id, si.item_type_id, si.attributes, si.created_at, si.updated_at,
		       sit.id, sit.name, sit.description, sit.attributes_schema, sit.display_order, sit.created_at, sit.updated_at,
		       ti.id, ti.item_type_id, ti.attributes, ti.created_at, ti.updated_at,
		       tit.id, tit.name, tit.description, tit.attributes_schema, tit.display_order, tit.created_at, tit.updated_at
		FROM item_relations ir
		JOIN relation_types rt ON rt.id = ir.relation_type_id
		JOIN items si ON si.id = ir.source_item_id
		JOIN item_types sit ON sit.id = si.item_type_id
		JOIN items ti ON ti.id = ir.target_item_id
		JOIN item_types tit ON tit.id = ti.item_type_id
		WHERE (ir.source_item_id = $1 OR ir.target_item_id = $1)`
	args := []any{itemID}

	if relationTypeID != nil {
		query += ` AND ir.relation_type_id = $2`
		args = append(args, *relationTypeID)
	}
	query += ` ORDER BY ir.created_at`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.ItemRelation
	for rows.Next() {
		var ir model.ItemRelation
		var rt model.RelationType
		var si, ti model.Item
		var sit, tit model.ItemType
		if err := rows.Scan(
			&ir.ID, &ir.RelationTypeID, &ir.SourceItemID, &ir.TargetItemID,
			&ir.Attributes, &ir.CreatedAt, &ir.UpdatedAt,
			&rt.ID, &rt.Name, &rt.Description,
			&rt.AttributesSchema, &rt.CreatedAt, &rt.UpdatedAt,
			&si.ID, &si.ItemTypeID, &si.Attributes, &si.CreatedAt, &si.UpdatedAt,
			&sit.ID, &sit.Name, &sit.Description, &sit.AttributesSchema, &sit.DisplayOrder, &sit.CreatedAt, &sit.UpdatedAt,
			&ti.ID, &ti.ItemTypeID, &ti.Attributes, &ti.CreatedAt, &ti.UpdatedAt,
			&tit.ID, &tit.Name, &tit.Description, &tit.AttributesSchema, &tit.DisplayOrder, &tit.CreatedAt, &tit.UpdatedAt,
		); err != nil {
			return nil, err
		}
		ir.RelationType = &rt
		si.ItemType = &sit
		ti.ItemType = &tit
		ir.SourceItem = &si
		ir.TargetItem = &ti
		result = append(result, ir)
	}
	return result, rows.Err()
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

	var id uuid.UUID
	err := s.pool.QueryRow(ctx,
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
	return s.GetRelation(ctx, id)
}

func (s *PgStore) UpdateRelation(ctx context.Context, id uuid.UUID, in model.ItemRelationUpdate) (*model.ItemRelation, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE item_relations SET attributes = $2 WHERE id = $1`,
		id, in.Attributes,
	)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetRelation(ctx, id)
}

func (s *PgStore) DeleteRelation(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM item_relations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
