package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/domains"
	"github.com/shinsaber/colec/internal/model"
)

// snapshotSQL reads meta-types as domains.Snapshot (same keys as the export
// format, without id; empty junctions as [] so that they compare equal).
var snapshotSQL = map[string]string{
	model.EntityItemTypes: `SELECT t.id, jsonb_strip_nulls(jsonb_build_object(
		'name', t.name, 'description', t.description, 'attributes_schema', t.attributes_schema,
		'display_order', t.display_order))
		FROM item_types t WHERE t.id = ANY($1)`,
	model.EntityCollectionTypes: `SELECT t.id, jsonb_strip_nulls(jsonb_build_object(
		'name', t.name, 'description', t.description, 'attributes_schema', t.attributes_schema,
		'item_type_ids', COALESCE((SELECT jsonb_agg(j.item_type_id ORDER BY j.item_type_id)
			FROM collection_type_item_types j WHERE j.collection_type_id = t.id), '[]')))
		FROM collection_types t WHERE t.id = ANY($1)`,
	model.EntityRelationTypes: `SELECT t.id, jsonb_strip_nulls(jsonb_build_object(
		'name', t.name, 'description', t.description, 'attributes_schema', t.attributes_schema,
		'source_item_type_ids', COALESCE((SELECT jsonb_agg(j.item_type_id ORDER BY j.item_type_id)
			FROM relation_type_sources j WHERE j.relation_type_id = t.id), '[]'),
		'target_item_type_ids', COALESCE((SELECT jsonb_agg(j.item_type_id ORDER BY j.item_type_id)
			FROM relation_type_targets j WHERE j.relation_type_id = t.id), '[]')))
		FROM relation_types t WHERE t.id = ANY($1)`,
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// localSnapshots reads the current version of the given meta-types.
func localSnapshots(ctx context.Context, q querier, ids []uuid.UUID) (map[uuid.UUID]domains.Snapshot, error) {
	out := map[uuid.UUID]domains.Snapshot{}
	for _, sql := range snapshotSQL {
		rows, err := q.Query(ctx, sql, ids)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id uuid.UUID
			var raw json.RawMessage
			if err := rows.Scan(&id, &raw); err != nil {
				rows.Close()
				return nil, err
			}
			s, err := domains.SnapshotOf(raw)
			if err != nil {
				rows.Close()
				return nil, err
			}
			out[id] = s
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

type origin struct {
	kind    string
	domain  string
	version int
	base    domains.Snapshot
}

// loadOrigins reads the origins of the given meta-types (every origin when ids is nil).
func loadOrigins(ctx context.Context, q querier, ids []uuid.UUID) (map[uuid.UUID]origin, error) {
	rows, err := q.Query(ctx, `SELECT meta_type_id, kind, domain, package_version, base
		FROM meta_type_origins WHERE $1::uuid[] IS NULL OR meta_type_id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]origin{}
	for rows.Next() {
		var id uuid.UUID
		var o origin
		var raw json.RawMessage
		if err := rows.Scan(&id, &o.kind, &o.domain, &o.version, &raw); err != nil {
			return nil, err
		}
		if o.base, err = domains.SnapshotOf(raw); err != nil {
			return nil, err
		}
		out[id] = o
	}
	return out, rows.Err()
}

func packageIDs(pkg *model.DomainPackage) []uuid.UUID {
	var ids []uuid.UUID
	for _, t := range pkg.ItemTypes {
		ids = append(ids, t.ID)
	}
	for _, t := range pkg.CollectionTypes {
		ids = append(ids, t.ID)
	}
	for _, t := range pkg.RelationTypes {
		ids = append(ids, t.ID)
	}
	return ids
}

// InstallDomain installs or updates a domain package in one transaction: the
// types the instance changed are merged (domains.PlanInstall), then written
// through the catalogue import; the reference copies are recorded. Nothing is
// written on dryRun or when the import reports an error.
func (s *PgStore) InstallDomain(ctx context.Context, pkg *model.DomainPackage, dryRun bool) (*model.DomainInstallReport, error) {
	rep := &model.DomainInstallReport{DryRun: dryRun, Domain: pkg.ID, ToVersion: pkg.PackageVersion,
		Entries: []model.DomainInstallEntry{}, Errors: []model.ImportError{}}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ids := packageIDs(pkg)
	local, err := localSnapshots(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	origins, err := loadOrigins(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	st := domains.State{Local: local, Base: map[uuid.UUID]domains.Snapshot{}, ItemTypes: map[uuid.UUID]bool{}}
	for id, o := range origins {
		st.Base[id] = o.base
		if o.domain == pkg.ID && (rep.FromVersion == nil || o.version > *rep.FromVersion) {
			v := o.version
			rep.FromVersion = &v
		}
	}
	itemTypeIDs, err := uuidColumn(ctx, tx, `SELECT id FROM item_types`)
	if err != nil {
		return nil, err
	}
	for _, id := range itemTypeIDs {
		st.ItemTypes[id] = true
	}

	plan, err := domains.PlanInstall(pkg, st)
	if err != nil {
		return nil, err
	}
	rep.Entries = plan.Entries

	imp := model.NewImportReport(dryRun)
	im := &importer{s: s, tx: tx, rep: imp}
	if err := im.itemTypes(ctx, plan.Catalog.ItemTypes); err != nil {
		return nil, err
	}
	if err := im.collectionTypes(ctx, plan.Catalog.CollectionTypes); err != nil {
		return nil, err
	}
	if err := im.relationTypes(ctx, plan.Catalog.RelationTypes); err != nil {
		return nil, err
	}
	rep.Errors = imp.Errors
	if len(rep.Errors) > 0 || dryRun {
		return rep, nil
	}

	for id, base := range plan.Bases {
		raw, err := json.Marshal(base)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO meta_type_origins (kind, meta_type_id, domain, package_version, base)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (kind, meta_type_id) DO UPDATE SET
				domain = EXCLUDED.domain, package_version = EXCLUDED.package_version,
				base = EXCLUDED.base, installed_at = now()`,
			plan.Kinds[id], id, pkg.ID, pkg.PackageVersion, raw); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	rep.Applied = true
	return rep, nil
}

// ListDomains returns the packages built into the server with their state on the instance.
func (s *PgStore) ListDomains(ctx context.Context) ([]model.DomainInfo, error) {
	pkgs, err := domains.All()
	if err != nil {
		return nil, err
	}
	origins, err := loadOrigins(ctx, s.pool, nil)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(origins))
	for id := range origins {
		ids = append(ids, id)
	}
	local, err := localSnapshots(ctx, s.pool, ids)
	if err != nil {
		return nil, err
	}
	out := make([]model.DomainInfo, 0, len(pkgs))
	for _, p := range pkgs {
		info := model.DomainInfo{ID: p.ID, Title: p.Title, Description: p.Description,
			PackageVersion: p.PackageVersion, Status: model.DomainNotInstalled,
			ItemTypes: len(p.ItemTypes), RelationTypes: len(p.RelationTypes), CollectionTypes: len(p.CollectionTypes)}
		for id, o := range origins {
			if o.domain != p.ID {
				continue
			}
			if info.InstalledVersion == nil || o.version > *info.InstalledVersion {
				v := o.version
				info.InstalledVersion = &v
			}
			if l, ok := local[id]; !ok || !snapshotsEqual(l, o.base) {
				info.ModifiedLocally = true
			}
		}
		if info.InstalledVersion != nil {
			info.Status = model.DomainUpToDate
			if *info.InstalledVersion < p.PackageVersion {
				info.Status = model.DomainUpdateAvailable
			}
		}
		out = append(out, info)
	}
	return out, nil
}

func snapshotsEqual(a, b domains.Snapshot) bool {
	ra, _ := json.Marshal(a)
	rb, _ := json.Marshal(b)
	return string(ra) == string(rb)
}

// InstallDomains installs the given packages (all of them when none is given),
// stopping at the first one whose import reports an error. Used before the
// demonstration seeds and by `colec-server install-domain`.
func (s *PgStore) InstallDomains(ctx context.Context, ids ...string) ([]*model.DomainInstallReport, error) {
	pkgs, err := domains.All()
	if err != nil {
		return nil, err
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	var reports []*model.DomainInstallReport
	for _, p := range pkgs {
		if len(ids) > 0 && !wanted[p.ID] {
			continue
		}
		delete(wanted, p.ID)
		rep, err := s.InstallDomain(ctx, p, false)
		if err != nil {
			return reports, err
		}
		reports = append(reports, rep)
		if len(rep.Errors) > 0 {
			msgs := make([]string, len(rep.Errors))
			for i, e := range rep.Errors {
				msgs[i] = e.Message
			}
			return reports, fmt.Errorf("domain %s: %s", p.ID, strings.Join(msgs, "; "))
		}
	}
	for id := range wanted {
		return reports, errors.New("unknown domain " + id)
	}
	return reports, s.RefreshInstalledDomains(ctx, "")
}

// RefreshInstalledDomains merges again every installed package but `except`:
// after an install, the relations of another domain that were left out (ends
// not installed yet, e.g. adapted_as → Series) are added.
func (s *PgStore) RefreshInstalledDomains(ctx context.Context, except string) error {
	list, err := s.ListDomains(ctx)
	if err != nil {
		return err
	}
	for _, d := range list {
		if d.ID == except || d.InstalledVersion == nil {
			continue
		}
		p, err := domains.Get(d.ID)
		if err != nil || p == nil {
			return err
		}
		if _, err := s.InstallDomain(ctx, p, false); err != nil {
			return err
		}
	}
	return nil
}
