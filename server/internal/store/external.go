package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// FindItemsByAttribute returns the items of itemTypeID whose attribute at path
// (JSON path as keys, e.g. ["external_ids", "wikidata"]) equals one of values,
// oldest first, with their tags. fold compares case- and accent-insensitively.
// The path is a bind parameter: it never reaches the SQL text.
func (s *PgStore) FindItemsByAttribute(ctx context.Context, itemTypeID uuid.UUID, path, values []string, fold bool) ([]model.Item, error) {
	if len(values) == 0 || len(path) == 0 {
		return nil, nil
	}
	cond := `i.attributes #>> $2 = ANY($3)`
	if fold {
		cond = `lower(f_unaccent(i.attributes #>> $2)) = ANY(SELECT lower(f_unaccent(v)) FROM unnest($3::text[]) v)`
	}
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.item_type_id, i.attributes, i.created_at, i.updated_at
		FROM items i
		WHERE i.item_type_id = $1 AND `+cond+`
		ORDER BY i.created_at, i.id`, itemTypeID, path, values)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Item, error) {
		var it model.Item
		err := row.Scan(&it.ID, &it.ItemTypeID, &it.Attributes, &it.CreatedAt, &it.UpdatedAt)
		return it, err
	})
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	tags, err := s.getItemsTags(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Tags = tags[items[i].ID]
	}
	return items, nil
}

// ListRelationsBetween returns the relations whose source and target are both
// in ids (without the embedded items and types).
func (s *PgStore) ListRelationsBetween(ctx context.Context, ids []uuid.UUID) ([]model.ItemRelation, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, relation_type_id, source_item_id, target_item_id, attributes, created_at, updated_at
		FROM item_relations
		WHERE source_item_id = ANY($1) AND target_item_id = ANY($1)
		ORDER BY created_at, id`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.ItemRelation, error) {
		var r model.ItemRelation
		err := row.Scan(&r.ID, &r.RelationTypeID, &r.SourceItemID, &r.TargetItemID,
			&r.Attributes, &r.CreatedAt, &r.UpdatedAt)
		return r, err
	})
}
