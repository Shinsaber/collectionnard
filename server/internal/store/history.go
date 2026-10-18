package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shinsaber/colec/internal/model"
)

// GetItemHistory lists the writes of an item and of the relations it is the
// source or target of (written by the triggers of migration 010), newest
// first, plus its creation. Authors are left as UserID: the handler resolves
// them for the reader.
func (s *PgStore) GetItemHistory(ctx context.Context, itemID uuid.UUID) (*model.ItemHistory, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT 'item', item_id, version, origin, origin_ref, user_id, before, after,
		       NULL::uuid, NULL::uuid, NULL::uuid, created_at, id
		FROM item_history WHERE item_id = $1
		UNION ALL
		SELECT 'relation', relation_id, version, origin, origin_ref, user_id, before, after,
		       relation_type_id, source_item_id, target_item_id, created_at, id
		FROM relation_history WHERE source_item_id = $1 OR target_item_id = $1
		ORDER BY 12 DESC, 13 DESC
		LIMIT $2`, itemID, model.MaxHistory+1)
	if err != nil {
		return nil, err
	}
	entries, err := pgx.CollectRows(rows, scanHistory)
	if err != nil {
		return nil, err
	}
	h := &model.ItemHistory{Entries: entries}
	if h.Entries == nil {
		h.Entries = []model.HistoryEntry{}
	}
	if len(h.Entries) > model.MaxHistory {
		h.Entries, h.Truncated = h.Entries[:model.MaxHistory], true
	}
	rows, err = s.pool.Query(ctx, `
		SELECT 'item', item_id, version, origin, origin_ref, user_id, before, after,
		       NULL::uuid, NULL::uuid, NULL::uuid, created_at, id
		FROM item_history WHERE item_id = $1 AND version = 1`, itemID)
	if err != nil {
		return nil, err
	}
	created, err := pgx.CollectRows(rows, scanHistory)
	if err != nil {
		return nil, err
	}
	if len(created) == 1 {
		h.Created = &created[0]
	}
	return h, nil
}

func scanHistory(row pgx.CollectableRow) (model.HistoryEntry, error) {
	var e model.HistoryEntry
	var relType, source, target *uuid.UUID
	var id int64
	err := row.Scan(&e.Kind, &e.EntityID, &e.Version, &e.Origin, &e.OriginRef, &e.UserID, &e.Before, &e.After,
		&relType, &source, &target, &e.CreatedAt, &id)
	if relType != nil && source != nil && target != nil {
		e.Relation = &model.HistoryRelation{RelationTypeID: *relType, SourceItemID: *source, TargetItemID: *target}
	}
	if e.Before == nil {
		e.Before = json.RawMessage("null")
	}
	if e.After == nil {
		e.After = json.RawMessage("null")
	}
	return e, err
}

// ErrDeletedVersion: the version records a deletion; nothing to restore.
var ErrDeletedVersion = errors.New("version is a deletion")

// GetItemVersion returns the attributes of an item after a given version.
func (s *PgStore) GetItemVersion(ctx context.Context, itemID uuid.UUID, version int) (json.RawMessage, error) {
	var after json.RawMessage
	err := s.pool.QueryRow(ctx,
		`SELECT after FROM item_history WHERE item_id = $1 AND version = $2`, itemID, version).Scan(&after)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if after == nil {
		return nil, ErrDeletedVersion
	}
	return after, nil
}
