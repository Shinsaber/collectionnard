package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/validator"
)

// ── Export ──────────────────────────────────────────────────────────

// ExportCatalog reads the whole catalogue in one repeatable-read snapshot,
// every list sorted by id (stable diffs between two exports).
func (s *PgStore) ExportCatalog(ctx context.Context) (*model.CatalogExport, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC().Truncate(time.Second)
	out := &model.CatalogExport{Format: model.CatalogFormat, Version: model.ExportVersion, ExportedAt: &now}

	metas, err := exportMetaTypes(ctx, tx, "item_types")
	if err != nil {
		return nil, err
	}
	orders, err := exportDisplayOrders(ctx, tx)
	if err != nil {
		return nil, err
	}
	out.ItemTypes = make([]model.ExportItemType, len(metas))
	for i, m := range metas {
		o := orders[m.ID]
		out.ItemTypes[i] = model.ExportItemType{ExportMetaType: m, DisplayOrder: &o}
	}
	if out.CollectionTypes, err = exportMetaTypes(ctx, tx, "collection_types"); err != nil {
		return nil, err
	}
	rts, err := exportMetaTypes(ctx, tx, "relation_types")
	if err != nil {
		return nil, err
	}
	sources, err := exportJunction(ctx, tx, "relation_type_sources")
	if err != nil {
		return nil, err
	}
	targets, err := exportJunction(ctx, tx, "relation_type_targets")
	if err != nil {
		return nil, err
	}
	out.RelationTypes = make([]model.ExportRelationType, len(rts))
	for i, rt := range rts {
		out.RelationTypes[i] = model.ExportRelationType{
			ExportMetaType:    rt,
			SourceItemTypeIDs: nonNil(sources[rt.ID]),
			TargetItemTypeIDs: nonNil(targets[rt.ID]),
		}
	}

	tags, err := exportTags(ctx, tx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id, item_type_id, attributes FROM items ORDER BY id`)
	if err != nil {
		return nil, err
	}
	out.Items, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (model.ExportItem, error) {
		var it model.ExportItem
		err := r.Scan(&it.ID, &it.ItemTypeID, &it.Attributes)
		it.Tags = tags[it.ID]
		return it, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = tx.Query(ctx, `
		SELECT id, relation_type_id, source_item_id, target_item_id, attributes
		FROM item_relations ORDER BY id`)
	if err != nil {
		return nil, err
	}
	out.Relations, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (model.ExportRelation, error) {
		var rel model.ExportRelation
		err := r.Scan(&rel.ID, &rel.RelationTypeID, &rel.SourceItemID, &rel.TargetItemID, &rel.Attributes)
		return rel, err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// metaTables is the whitelist of meta-type tables (their name is part of the SQL).
var metaTables = map[string]bool{"item_types": true, "collection_types": true, "relation_types": true}

func exportMetaTypes(ctx context.Context, tx pgx.Tx, table string) ([]model.ExportMetaType, error) {
	if !metaTables[table] {
		return nil, fmt.Errorf("unknown meta-type table %q", table)
	}
	rows, err := tx.Query(ctx, `SELECT id, name, description, attributes_schema FROM `+table+` ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (model.ExportMetaType, error) {
		var m model.ExportMetaType
		err := r.Scan(&m.ID, &m.Name, &m.Description, &m.AttributesSchema)
		return m, err
	})
}

// exportDisplayOrders reads item_types.display_order by type.
func exportDisplayOrders(ctx context.Context, tx pgx.Tx) (map[uuid.UUID]int, error) {
	rows, err := tx.Query(ctx, `SELECT id, display_order FROM item_types`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]int{}
	for rows.Next() {
		var id uuid.UUID
		var o int
		if err := rows.Scan(&id, &o); err != nil {
			return nil, err
		}
		out[id] = o
	}
	return out, rows.Err()
}

// exportJunction reads relation_type_sources / relation_type_targets.
func exportJunction(ctx context.Context, tx pgx.Tx, table string) (map[uuid.UUID][]uuid.UUID, error) {
	if table != "relation_type_sources" && table != "relation_type_targets" {
		return nil, fmt.Errorf("unknown junction table %q", table)
	}
	rows, err := tx.Query(ctx, `SELECT relation_type_id, item_type_id FROM `+table+` ORDER BY relation_type_id, item_type_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID][]uuid.UUID{}
	for rows.Next() {
		var rt, it uuid.UUID
		if err := rows.Scan(&rt, &it); err != nil {
			return nil, err
		}
		out[rt] = append(out[rt], it)
	}
	return out, rows.Err()
}

// exportTags returns the sorted tag names of every item.
func exportTags(ctx context.Context, tx pgx.Tx) (map[uuid.UUID][]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT it.item_id, t.name FROM item_tags it JOIN tags t ON t.id = it.tag_id
		ORDER BY it.item_id, t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID][]string{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = append(out[id], name)
	}
	return out, rows.Err()
}

func nonNil(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

// ── Import ──────────────────────────────────────────────────────────

// outcome of one upsert.
type outcome int

const (
	unchanged outcome = iota
	created
	updated
)

func (o outcome) count(c *model.ImportCounts) {
	switch o {
	case created:
		c.Created++
	case updated:
		c.Updated++
	default:
		c.Unchanged++
	}
}

// importer runs one import inside a transaction. Each entity is written in
// its own savepoint: a failing row is reported and rolled back alone, so all
// the errors of a file are listed in one pass.
type importer struct {
	s   *PgStore
	tx  pgx.Tx
	rep *model.ImportReport
}

func (s *PgStore) ImportCatalog(ctx context.Context, cat *model.CatalogExport, dryRun bool) (*model.ImportReport, error) {
	rep := model.NewImportReport(dryRun)
	if cat.Format != model.CatalogFormat || cat.Version != model.ExportVersion {
		rep.Fail(model.EntityFile, -1, nil, fmt.Sprintf(
			"unsupported file: format %q version %d (expected %q version %d)",
			cat.Format, cat.Version, model.CatalogFormat, model.ExportVersion))
		return rep, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	im := &importer{s: s, tx: tx, rep: rep}
	steps := []func(context.Context) error{
		func(ctx context.Context) error {
			return im.itemTypes(ctx, cat.ItemTypes)
		},
		func(ctx context.Context) error {
			return im.metaTypes(ctx, model.EntityCollectionTypes, "collection_types", cat.CollectionTypes)
		},
		func(ctx context.Context) error { return im.relationTypes(ctx, cat.RelationTypes) },
		func(ctx context.Context) error { return im.items(ctx, cat.Items) },
		func(ctx context.Context) error { return im.relations(ctx, cat.Relations) },
	}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return nil, err
		}
	}

	if len(rep.Errors) > 0 || dryRun {
		return rep, nil // deferred rollback
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	rep.Applied = true
	return rep, nil
}

// row runs fn in a savepoint. A database error of the row (constraint…) is
// reported and returns ok=false; a failure of the savepoint itself is fatal.
func (im *importer) row(ctx context.Context, entity string, index int, id uuid.UUID,
	fn func(tx pgx.Tx) (outcome, error)) (outcome, bool, error) {
	sp, err := im.tx.Begin(ctx)
	if err != nil {
		return 0, false, err
	}
	o, err := fn(sp)
	if err != nil {
		if rbErr := sp.Rollback(ctx); rbErr != nil {
			return 0, false, rbErr
		}
		im.rep.Fail(entity, index, &id, describeDBError(err))
		return 0, false, nil
	}
	if err := sp.Commit(ctx); err != nil {
		return 0, false, err
	}
	return o, true, nil
}

// rowError is a validation failure detected before writing.
type rowError string

func (e rowError) Error() string { return string(e) }

// describeDBError turns a constraint violation into a short English message.
func describeDBError(err error) string {
	var re rowError
	if errors.As(err, &re) {
		return string(re)
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			if strings.Contains(pg.ConstraintName, "name") {
				return "name already used by another entity with a different id"
			}
			return "an equivalent entity already exists with a different id (" + pg.ConstraintName + ")"
		case "23503":
			return "references an unknown entity (" + pg.ConstraintName + ")"
		case "23502", "22P02", "23514":
			return "invalid value: " + pg.Message
		}
		return pg.Message
	}
	return err.Error()
}

// upsert executes an INSERT … ON CONFLICT (id) DO UPDATE … WHERE <changed>
// RETURNING (xmax = 0): no row means the stored entity is already identical.
func upsert(ctx context.Context, tx pgx.Tx, sql string, args ...any) (outcome, error) {
	var inserted bool
	err := tx.QueryRow(ctx, sql, args...).Scan(&inserted)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return unchanged, nil
	case err != nil:
		return 0, err
	case inserted:
		return created, nil
	}
	return updated, nil
}

func emptyObject(raw json.RawMessage) json.RawMessage {
	if len(strings.TrimSpace(string(raw))) == 0 || string(raw) == "null" {
		return json.RawMessage(`{}`)
	}
	return raw
}

func (im *importer) metaTypes(ctx context.Context, entity, table string, list []model.ExportMetaType) error {
	if !metaTables[table] {
		return fmt.Errorf("unknown meta-type table %q", table)
	}
	sql := fmt.Sprintf(`
		INSERT INTO %[1]s (id, name, description, attributes_schema) VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, description = EXCLUDED.description,
			attributes_schema = EXCLUDED.attributes_schema
		WHERE (%[1]s.name, %[1]s.description, %[1]s.attributes_schema)
			IS DISTINCT FROM (EXCLUDED.name, EXCLUDED.description, EXCLUDED.attributes_schema)
		RETURNING (xmax = 0)`, table)
	for i, m := range list {
		o, ok, err := im.row(ctx, entity, i, m.ID, func(tx pgx.Tx) (outcome, error) {
			if err := checkMeta(m); err != nil {
				return 0, err
			}
			return upsert(ctx, tx, sql, m.ID, m.Name, m.Description, emptyObject(m.AttributesSchema))
		})
		if err != nil {
			return err
		}
		if ok {
			o.count(im.rep.Counts[entity])
		}
	}
	return nil
}

// itemTypes upserts the item types; an absent display_order keeps the stored one (100 if new).
func (im *importer) itemTypes(ctx context.Context, list []model.ExportItemType) error {
	const sql = `
		INSERT INTO item_types (id, name, description, attributes_schema, display_order)
		VALUES ($1, $2, $3, $4, COALESCE($5, 100))
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, description = EXCLUDED.description,
			attributes_schema = EXCLUDED.attributes_schema,
			display_order = COALESCE($5, item_types.display_order)
		WHERE (item_types.name, item_types.description, item_types.attributes_schema, item_types.display_order)
			IS DISTINCT FROM (EXCLUDED.name, EXCLUDED.description, EXCLUDED.attributes_schema,
				COALESCE($5, item_types.display_order))
		RETURNING (xmax = 0)`
	for i, it := range list {
		o, ok, err := im.row(ctx, model.EntityItemTypes, i, it.ID, func(tx pgx.Tx) (outcome, error) {
			if err := checkMeta(it.ExportMetaType); err != nil {
				return 0, err
			}
			return upsert(ctx, tx, sql, it.ID, it.Name, it.Description, emptyObject(it.AttributesSchema), it.DisplayOrder)
		})
		if err != nil {
			return err
		}
		if ok {
			o.count(im.rep.Counts[model.EntityItemTypes])
		}
	}
	return nil
}

func checkMeta(m model.ExportMetaType) error {
	if m.ID == uuid.Nil {
		return rowError("id is required")
	}
	if strings.TrimSpace(m.Name) == "" {
		return rowError("name is required")
	}
	if err := validator.CheckSchema(emptyObject(m.AttributesSchema)); err != nil {
		return rowError("attributes_schema: " + err.Error())
	}
	return nil
}

func (im *importer) relationTypes(ctx context.Context, list []model.ExportRelationType) error {
	const sql = `
		INSERT INTO relation_types (id, name, description, attributes_schema) VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, description = EXCLUDED.description,
			attributes_schema = EXCLUDED.attributes_schema
		WHERE (relation_types.name, relation_types.description, relation_types.attributes_schema)
			IS DISTINCT FROM (EXCLUDED.name, EXCLUDED.description, EXCLUDED.attributes_schema)
		RETURNING (xmax = 0)`
	for i, rt := range list {
		o, ok, err := im.row(ctx, model.EntityRelationTypes, i, rt.ID, func(tx pgx.Tx) (outcome, error) {
			if err := checkMeta(rt.ExportMetaType); err != nil {
				return 0, err
			}
			if len(rt.SourceItemTypeIDs) == 0 || len(rt.TargetItemTypeIDs) == 0 {
				return 0, rowError("source_item_type_ids and target_item_type_ids must not be empty")
			}
			o, err := upsert(ctx, tx, sql, rt.ID, rt.Name, rt.Description, emptyObject(rt.AttributesSchema))
			if err != nil {
				return 0, err
			}
			for _, j := range []struct {
				table string
				ids   []uuid.UUID
			}{{"relation_type_sources", rt.SourceItemTypeIDs}, {"relation_type_targets", rt.TargetItemTypeIDs}} {
				changed, err := replaceJunction(ctx, tx, j.table, rt.ID, j.ids)
				if err != nil {
					return 0, err
				}
				if changed && o == unchanged {
					o = updated
				}
			}
			return o, nil
		})
		if err != nil {
			return err
		}
		if ok {
			o.count(im.rep.Counts[model.EntityRelationTypes])
		}
	}
	return nil
}

// replaceJunction sets the item types of a relation type side, reporting
// whether the stored set differed.
func replaceJunction(ctx context.Context, tx pgx.Tx, table string, rtID uuid.UUID, ids []uuid.UUID) (bool, error) {
	current, err := uuidColumn(ctx, tx, `SELECT item_type_id FROM `+table+` WHERE relation_type_id = $1`, rtID)
	if err != nil {
		return false, err
	}
	if sameUUIDs(current, ids) {
		return false, nil
	}
	ids = slices.Compact(slices.SortedFunc(slices.Values(ids), func(a, b uuid.UUID) int {
		return strings.Compare(a.String(), b.String())
	}))
	if _, err := tx.Exec(ctx, `DELETE FROM `+table+` WHERE relation_type_id = $1`, rtID); err != nil {
		return false, err
	}
	return true, insertJunctions(ctx, tx, table, rtID, ids)
}

func uuidColumn(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}

func sameUUIDs(a, b []uuid.UUID) bool {
	key := func(ids []uuid.UUID) []string {
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			out = append(out, id.String())
		}
		sort.Strings(out)
		return slices.Compact(out)
	}
	return slices.Equal(key(a), key(b))
}

// schemas reads every schema of a meta-type table as written so far in the transaction.
func (im *importer) schemas(ctx context.Context, table string) (map[uuid.UUID]json.RawMessage, error) {
	if !metaTables[table] {
		return nil, fmt.Errorf("unknown meta-type table %q", table)
	}
	rows, err := im.tx.Query(ctx, `SELECT id, attributes_schema FROM `+table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]json.RawMessage{}
	for rows.Next() {
		var id uuid.UUID
		var schema json.RawMessage
		if err := rows.Scan(&id, &schema); err != nil {
			return nil, err
		}
		out[id] = schema
	}
	return out, rows.Err()
}

// validateAttrs checks attributes against a schema and renders the first
// violations as "attributes: /path: message".
func validateAttrs(schema, attrs json.RawMessage) error {
	err := validator.ValidateAttributes(schema, attrs)
	if err == nil {
		return nil
	}
	fields := validator.FormatValidationError(err)
	parts := make([]string, 0, len(fields))
	for path, msg := range fields {
		parts = append(parts, path+": "+msg)
	}
	sort.Strings(parts)
	return rowError("attributes: " + strings.Join(parts, "; "))
}

func (im *importer) items(ctx context.Context, list []model.ExportItem) error {
	schemas, err := im.schemas(ctx, "item_types")
	if err != nil {
		return err
	}
	const sql = `
		INSERT INTO items (id, item_type_id, attributes) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET item_type_id = EXCLUDED.item_type_id, attributes = EXCLUDED.attributes
		WHERE (items.item_type_id, items.attributes) IS DISTINCT FROM (EXCLUDED.item_type_id, EXCLUDED.attributes)
		RETURNING (xmax = 0)`
	for i, it := range list {
		o, ok, err := im.row(ctx, model.EntityItems, i, it.ID, func(tx pgx.Tx) (outcome, error) {
			if it.ID == uuid.Nil {
				return 0, rowError("id is required")
			}
			schema, found := schemas[it.ItemTypeID]
			if !found {
				return 0, rowError("unknown item_type_id " + it.ItemTypeID.String())
			}
			attrs := emptyObject(it.Attributes)
			if err := validateAttrs(schema, attrs); err != nil {
				return 0, err
			}
			o, err := upsert(ctx, tx, sql, it.ID, it.ItemTypeID, attrs)
			if err != nil {
				return 0, err
			}
			changed, err := im.s.replaceItemTags(ctx, tx, it.ID, it.Tags)
			if err != nil {
				return 0, err
			}
			if changed && o == unchanged {
				o = updated
			}
			return o, nil
		})
		if err != nil {
			return err
		}
		if ok {
			o.count(im.rep.Counts[model.EntityItems])
		}
	}
	return nil
}

// replaceItemTags makes the tag set of an item exactly names (normalised like
// upsertItemTags), reporting whether it changed.
func (s *PgStore) replaceItemTags(ctx context.Context, tx pgx.Tx, itemID uuid.UUID, names []string) (bool, error) {
	want := make([]string, 0, len(names))
	for _, n := range names {
		if n = strings.ToLower(strings.TrimSpace(n)); n != "" {
			want = append(want, n)
		}
	}
	sort.Strings(want)
	want = slices.Compact(want)

	rows, err := tx.Query(ctx, `
		SELECT t.name FROM item_tags it JOIN tags t ON t.id = it.tag_id
		WHERE it.item_id = $1 ORDER BY t.name`, itemID)
	if err != nil {
		return false, err
	}
	current, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return false, err
	}
	if slices.Equal(current, want) {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM item_tags WHERE item_id = $1`, itemID); err != nil {
		return false, err
	}
	return true, s.upsertItemTags(ctx, tx, itemID, want)
}

// relationRules are the allowed source / target item types of each relation type.
type relationRules struct {
	schema           json.RawMessage
	sources, targets map[uuid.UUID]bool
}

func (im *importer) relationRules(ctx context.Context) (map[uuid.UUID]*relationRules, error) {
	schemas, err := im.schemas(ctx, "relation_types")
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]*relationRules, len(schemas))
	for id, schema := range schemas {
		out[id] = &relationRules{schema: schema, sources: map[uuid.UUID]bool{}, targets: map[uuid.UUID]bool{}}
	}
	for _, side := range []string{"relation_type_sources", "relation_type_targets"} {
		rows, err := im.tx.Query(ctx, `SELECT relation_type_id, item_type_id FROM `+side)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var rt, it uuid.UUID
			if err := rows.Scan(&rt, &it); err != nil {
				rows.Close()
				return nil, err
			}
			if r := out[rt]; r != nil {
				if side == "relation_type_sources" {
					r.sources[it] = true
				} else {
					r.targets[it] = true
				}
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (im *importer) relations(ctx context.Context, list []model.ExportRelation) error {
	if len(list) == 0 {
		return nil
	}
	rules, err := im.relationRules(ctx)
	if err != nil {
		return err
	}
	itemTypes := map[uuid.UUID]uuid.UUID{}
	rows, err := im.tx.Query(ctx, `SELECT id, item_type_id FROM items`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, t uuid.UUID
		if err := rows.Scan(&id, &t); err != nil {
			rows.Close()
			return err
		}
		itemTypes[id] = t
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	const sql = `
		INSERT INTO item_relations (id, relation_type_id, source_item_id, target_item_id, attributes)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			relation_type_id = EXCLUDED.relation_type_id, source_item_id = EXCLUDED.source_item_id,
			target_item_id = EXCLUDED.target_item_id, attributes = EXCLUDED.attributes
		WHERE (item_relations.relation_type_id, item_relations.source_item_id,
		       item_relations.target_item_id, item_relations.attributes)
			IS DISTINCT FROM (EXCLUDED.relation_type_id, EXCLUDED.source_item_id,
			                  EXCLUDED.target_item_id, EXCLUDED.attributes)
		RETURNING (xmax = 0)`
	for i, rel := range list {
		o, ok, err := im.row(ctx, model.EntityRelations, i, rel.ID, func(tx pgx.Tx) (outcome, error) {
			if rel.ID == uuid.Nil {
				return 0, rowError("id is required")
			}
			r := rules[rel.RelationTypeID]
			if r == nil {
				return 0, rowError("unknown relation_type_id " + rel.RelationTypeID.String())
			}
			srcType, ok := itemTypes[rel.SourceItemID]
			if !ok {
				return 0, rowError("unknown source_item_id " + rel.SourceItemID.String())
			}
			dstType, ok := itemTypes[rel.TargetItemID]
			if !ok {
				return 0, rowError("unknown target_item_id " + rel.TargetItemID.String())
			}
			if !r.sources[srcType] {
				return 0, rowError("source item type is not allowed by this relation type")
			}
			if !r.targets[dstType] {
				return 0, rowError("target item type is not allowed by this relation type")
			}
			attrs := emptyObject(rel.Attributes)
			if err := validateAttrs(r.schema, attrs); err != nil {
				return 0, err
			}
			return upsert(ctx, tx, sql, rel.ID, rel.RelationTypeID, rel.SourceItemID, rel.TargetItemID, attrs)
		})
		if err != nil {
			return err
		}
		if ok {
			o.count(im.rep.Counts[model.EntityRelations])
		}
	}
	return nil
}
