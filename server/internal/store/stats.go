package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shinsaber/colec/internal/collstats"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
)

func (s *PgStore) GetStats(ctx context.Context) (*model.Stats, error) {
	var stats model.Stats

	// Items by type
	rows, err := s.pool.Query(ctx,
		`SELECT it.id, it.name, count(i.id)
		 FROM item_types it
		 LEFT JOIN items i ON i.item_type_id = it.id
		 GROUP BY it.id
		 ORDER BY it.display_order, it.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ibt model.ItemsByType
		if err := rows.Scan(&ibt.TypeID, &ibt.TypeName, &ibt.Count); err != nil {
			return nil, err
		}
		stats.ItemsByType = append(stats.ItemsByType, ibt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM items`).Scan(&stats.TotalItems)
	if err != nil {
		return nil, err
	}

	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM collections`).Scan(&stats.TotalCollections)
	if err != nil {
		return nil, err
	}

	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM item_relations`).Scan(&stats.TotalRelations)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetCollectionStats computes the statistics of a collection (P4-1): attribute
// aggregates from the CollectionType schema (collstats), items per type and
// completeness through the relations marked `x-completeness`. The hidden fields
// (P4-2, reader other than the owner) are removed from the schema first, so
// that no aggregate is computed on them.
func (s *PgStore) GetCollectionStats(ctx context.Context, collectionID uuid.UUID, hidden []string) (*model.CollectionStats, error) {
	var schema json.RawMessage
	err := s.pool.QueryRow(ctx,
		`SELECT ct.attributes_schema FROM collections c
		 JOIN collection_types ct ON ct.id = c.collection_type_id WHERE c.id = $1`, collectionID).Scan(&schema)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT attributes, created_at FROM collection_items WHERE collection_id = $1`, collectionID)
	if err != nil {
		return nil, err
	}
	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (collstats.Entry, error) {
		var e collstats.Entry
		err := row.Scan(&e.Attributes, &e.AddedAt)
		return e, err
	})
	if err != nil {
		return nil, err
	}
	agg := collstats.Aggregate(privacy.StripSchema(schema, hidden), entries)
	st := &model.CollectionStats{
		CollectionID:  collectionID,
		TotalItems:    len(entries),
		Numbers:       agg.Numbers,
		Distributions: agg.Distributions,
		Timelines:     agg.Timelines,
		Added:         agg.Added,
	}

	if st.ItemsByType, err = s.itemsByType(ctx,
		`SELECT it.id, it.name, count(*) FROM collection_items ci
		 JOIN items i ON i.id = ci.item_id JOIN item_types it ON it.id = i.item_type_id
		 WHERE ci.collection_id = $1
		 GROUP BY it.id ORDER BY count(*) DESC, it.display_order, it.name`, collectionID); err != nil {
		return nil, err
	}

	// Containers with at least one child in the collection; incomplete ones
	// first, closest to completion at the top.
	rows, err = s.pool.Query(ctx, `
		WITH mine AS (SELECT item_id FROM collection_items WHERE collection_id = $1),
		comp AS (SELECT id FROM relation_types WHERE attributes_schema -> 'x-completeness' = 'true'::jsonb),
		parents AS (
			SELECT DISTINCT ir.source_item_id AS id FROM item_relations ir
			JOIN mine m ON m.item_id = ir.target_item_id
			WHERE ir.relation_type_id IN (SELECT id FROM comp)),
		counts AS (
			SELECT p.id, count(DISTINCT ir.target_item_id) AS total,
			       count(DISTINCT ir.target_item_id) FILTER (WHERE ir.target_item_id IN (SELECT item_id FROM mine)) AS owned
			FROM parents p
			JOIN item_relations ir ON ir.source_item_id = p.id AND ir.relation_type_id IN (SELECT id FROM comp)
			GROUP BY p.id)
		SELECT i.id, i.item_type_id, i.attributes, c.owned, c.total
		FROM counts c JOIN items i ON i.id = c.id
		ORDER BY c.owned = c.total, c.owned::float / c.total DESC, c.total DESC, i.id
		LIMIT $2`, collectionID, model.MaxCompleteness+1)
	if err != nil {
		return nil, err
	}
	st.Completeness, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Completeness, error) {
		var c model.Completeness
		err := row.Scan(&c.Item.ID, &c.Item.ItemTypeID, &c.Item.Attributes, &c.Owned, &c.Total)
		return c, err
	})
	if err != nil {
		return nil, err
	}
	if st.Completeness == nil {
		st.Completeness = []model.Completeness{}
	}
	if len(st.Completeness) > model.MaxCompleteness {
		st.Completeness = st.Completeness[:model.MaxCompleteness]
		st.CompletenessTruncated = true
	}

	// ItemTypes cited, for the labels of a reader who cannot list /item-types.
	var typeIDs []uuid.UUID
	for _, b := range st.ItemsByType {
		typeIDs = append(typeIDs, b.TypeID)
	}
	for _, c := range st.Completeness {
		typeIDs = append(typeIDs, c.Item.ItemTypeID)
	}
	rows, err = s.pool.Query(ctx,
		`SELECT id, name, description, attributes_schema, display_order, created_at, updated_at
		 FROM item_types WHERE id = ANY($1) ORDER BY display_order, name`, typeIDs)
	if err != nil {
		return nil, err
	}
	st.ItemTypes, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.ItemType, error) {
		var it model.ItemType
		err := row.Scan(&it.ID, &it.Name, &it.Description, &it.AttributesSchema, &it.DisplayOrder, &it.CreatedAt, &it.UpdatedAt)
		return it, err
	})
	if err != nil {
		return nil, err
	}
	if st.ItemTypes == nil {
		st.ItemTypes = []model.ItemType{}
	}
	return st, nil
}

// GetUserStats summarizes all the collections of a user (P4-1).
func (s *PgStore) GetUserStats(ctx context.Context, userID uuid.UUID) (*model.MeStats, error) {
	st := &model.MeStats{}
	rows, err := s.pool.Query(ctx,
		`SELECT c.id, c.name, c.collection_type_id, count(ci.id)
		 FROM collections c LEFT JOIN collection_items ci ON ci.collection_id = c.id
		 WHERE c.user_id = $1
		 GROUP BY c.id ORDER BY count(ci.id) DESC, c.name`, userID)
	if err != nil {
		return nil, err
	}
	st.Collections, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.CollectionSummary, error) {
		var c model.CollectionSummary
		err := row.Scan(&c.ID, &c.Name, &c.CollectionTypeID, &c.ItemCount)
		return c, err
	})
	if err != nil {
		return nil, err
	}
	if st.Collections == nil {
		st.Collections = []model.CollectionSummary{}
	}
	st.TotalCollections = len(st.Collections)
	for _, c := range st.Collections {
		st.TotalEntries += c.ItemCount
	}

	if err := s.pool.QueryRow(ctx,
		`SELECT count(DISTINCT ci.item_id) FROM collection_items ci
		 JOIN collections c ON c.id = ci.collection_id WHERE c.user_id = $1`, userID).Scan(&st.DistinctItems); err != nil {
		return nil, err
	}
	if st.ItemsByType, err = s.itemsByType(ctx,
		`SELECT it.id, it.name, count(DISTINCT i.id) FROM collection_items ci
		 JOIN collections c ON c.id = ci.collection_id
		 JOIN items i ON i.id = ci.item_id JOIN item_types it ON it.id = i.item_type_id
		 WHERE c.user_id = $1
		 GROUP BY it.id ORDER BY count(DISTINCT i.id) DESC, it.display_order, it.name`, userID); err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx,
		`SELECT to_char(ci.created_at AT TIME ZONE 'UTC', 'YYYY-MM'), count(*) FROM collection_items ci
		 JOIN collections c ON c.id = ci.collection_id WHERE c.user_id = $1 GROUP BY 1`, userID)
	if err != nil {
		return nil, err
	}
	added := map[string]int{}
	for rows.Next() {
		var month string
		var n int
		if err := rows.Scan(&month, &n); err != nil {
			rows.Close()
			return nil, err
		}
		added[month] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	st.Added = collstats.Months(added)
	return st, nil
}

// itemsByType runs a (type id, type name, count) query; never nil.
func (s *PgStore) itemsByType(ctx context.Context, query string, arg any) ([]model.ItemsByType, error) {
	rows, err := s.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.ItemsByType, error) {
		var t model.ItemsByType
		err := row.Scan(&t.TypeID, &t.TypeName, &t.Count)
		return t, err
	})
	if out == nil {
		out = []model.ItemsByType{}
	}
	return out, err
}
