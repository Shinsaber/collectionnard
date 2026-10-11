package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// ImportCollection restores a collection export into the account of userID.
// The target is the collection of the file when it belongs to the user
// (updated, and made exactly like the file: missing items are removed),
// otherwise a new collection (keeping the file id when it is free).
func (s *PgStore) ImportCollection(ctx context.Context, userID uuid.UUID, exp *model.CollectionExport, dryRun bool) (*model.ImportReport, error) {
	rep := model.NewCollectionImportReport(dryRun)
	if exp.Format != model.CollectionFormat || exp.Version != model.ExportVersion {
		rep.Fail(model.EntityFile, -1, nil, fmt.Sprintf(
			"unsupported file: format %q version %d (expected %q version %d)",
			exp.Format, exp.Version, model.CollectionFormat, model.ExportVersion))
		return rep, nil
	}
	head := exp.Collection
	if head.Name == "" {
		rep.Fail(model.EntityCollection, -1, nil, "collection name is required")
		return rep, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Collection type: by id, else by name.
	var ctID uuid.UUID
	var ctSchema json.RawMessage
	err = tx.QueryRow(ctx, `SELECT id, attributes_schema FROM collection_types WHERE id = $1`,
		head.CollectionType.ID).Scan(&ctID, &ctSchema)
	if errors.Is(err, pgx.ErrNoRows) && head.CollectionType.Name != "" {
		err = tx.QueryRow(ctx, `SELECT id, attributes_schema FROM collection_types WHERE name = $1`,
			head.CollectionType.Name).Scan(&ctID, &ctSchema)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		rep.Fail(model.EntityCollection, -1, nil, fmt.Sprintf("unknown collection type %q (%s)",
			head.CollectionType.Name, head.CollectionType.ID))
		return rep, nil
	}
	if err != nil {
		return nil, err
	}

	// Target collection.
	target := head.ID
	var owner uuid.UUID
	err = tx.QueryRow(ctx, `SELECT user_id FROM collections WHERE id = $1`, target).Scan(&owner)
	exists := err == nil
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return nil, err
	}
	if target == uuid.Nil || (exists && owner != userID) {
		target, exists = uuid.New(), false
	}
	name, err := freeCollectionName(ctx, tx, userID, target, head.Name)
	if err != nil {
		return nil, err
	}

	colCounts := rep.Counts[model.EntityCollection]
	if exists {
		o, err := upsert(ctx, tx, `
			UPDATE collections SET name = $2, description = $3, collection_type_id = $4
			WHERE id = $1 AND (name, description, collection_type_id) IS DISTINCT FROM ($2, $3, $4)
			RETURNING false`, target, name, head.Description, ctID)
		if err != nil {
			return nil, err
		}
		o.count(colCounts)
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO collections (id, user_id, collection_type_id, name, description)
			VALUES ($1, $2, $3, $4, $5)`, target, userID, ctID, name, head.Description); err != nil {
			return nil, err
		}
		colCounts.Created++
	}
	rep.CollectionID = &target

	if err := s.importCollectionItems(ctx, tx, rep, target, ctSchema, exp.Items); err != nil {
		return nil, err
	}

	if len(rep.Errors) > 0 || dryRun {
		return rep, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	rep.Applied = true
	return rep, nil
}

// freeCollectionName returns name, or "name (2)", "name (3)"… when another
// collection of the user (not target) already uses it.
func freeCollectionName(ctx context.Context, tx pgx.Tx, userID, target uuid.UUID, name string) (string, error) {
	candidate := name
	for n := 2; ; n++ {
		var taken bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM collections WHERE user_id = $1 AND name = $2 AND id <> $3)`,
			userID, candidate, target).Scan(&taken); err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		if n > 1000 {
			return "", errors.New("no free collection name")
		}
		candidate = fmt.Sprintf("%s (%d)", name, n)
	}
}

func (s *PgStore) importCollectionItems(ctx context.Context, tx pgx.Tx, rep *model.ImportReport,
	collectionID uuid.UUID, schema json.RawMessage, entries []model.CollectionExportItem) error {
	im := &importer{s: s, tx: tx, rep: rep}
	counts := rep.Counts[model.EntityItems]

	ids := make([]uuid.UUID, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ItemID)
	}
	known, err := uuidColumn(ctx, tx, `SELECT id FROM items WHERE id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	existing := make(map[uuid.UUID]bool, len(known))
	for _, id := range known {
		existing[id] = true
	}

	const sql = `
		INSERT INTO collection_items (collection_id, item_id, attributes, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (collection_id, item_id) DO UPDATE SET attributes = EXCLUDED.attributes
		WHERE collection_items.attributes IS DISTINCT FROM EXCLUDED.attributes
		RETURNING (xmax = 0)`
	seen := make(map[uuid.UUID]bool, len(entries))
	for i, e := range entries {
		o, ok, err := im.row(ctx, model.EntityItems, i, e.ItemID, func(sp pgx.Tx) (outcome, error) {
			switch {
			case e.ItemID == uuid.Nil:
				return 0, rowError("item_id is required")
			case seen[e.ItemID]:
				return 0, rowError("item listed twice in the file")
			case !existing[e.ItemID]:
				return 0, rowError("unknown item_id (not in the catalogue)")
			}
			seen[e.ItemID] = true
			attrs := emptyObject(e.Attributes)
			if err := validateAttrs(schema, attrs); err != nil {
				return 0, err
			}
			added := e.AddedAt
			if added.IsZero() {
				added = time.Now()
			}
			return upsert(ctx, sp, sql, collectionID, e.ItemID, attrs, added)
		})
		if err != nil {
			return err
		}
		if ok {
			o.count(counts)
		}
	}

	// The collection becomes exactly the file: drop the items it no longer lists.
	tag, err := tx.Exec(ctx,
		`DELETE FROM collection_items WHERE collection_id = $1 AND NOT (item_id = ANY($2))`,
		collectionID, ids)
	if err != nil {
		return err
	}
	counts.Removed = int(tag.RowsAffected())
	return nil
}
