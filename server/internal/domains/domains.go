// Package domains holds the installable domain packages (meta-types of a kind of
// collection: TV series, books…) and plans their installation on an instance:
// a type the instance changed is merged (Merge3), never overwritten.
package domains

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
)

//go:embed packages/*.json
var files embed.FS

// All returns the packages built into the server, sorted by id.
func All() ([]*model.DomainPackage, error) {
	entries, err := fs.ReadDir(files, "packages")
	if err != nil {
		return nil, err
	}
	var out []*model.DomainPackage
	for _, e := range entries {
		raw, err := files.ReadFile("packages/" + e.Name())
		if err != nil {
			return nil, err
		}
		p, err := Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Get returns the package with this id, or nil.
func Get(id string) (*model.DomainPackage, error) {
	all, err := All()
	if err != nil {
		return nil, err
	}
	for _, p := range all {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}

// Parse decodes and checks a package file.
func Parse(raw []byte) (*model.DomainPackage, error) {
	var p model.DomainPackage
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	switch {
	case p.Format != model.DomainFormat || p.Version != 1:
		return nil, fmt.Errorf("unsupported package: format %q version %d", p.Format, p.Version)
	case p.ID == "" || p.PackageVersion < 1:
		return nil, fmt.Errorf("package id and package_version are required")
	}
	return &p, nil
}

// State is what the planner needs to know about the instance.
type State struct {
	// Local: current meta-types of the package ids present on the instance.
	Local map[uuid.UUID]Snapshot
	// Base: package version of each type at its last install (origins).
	Base map[uuid.UUID]Snapshot
	// ItemTypes: ids of every item type on the instance.
	ItemTypes map[uuid.UUID]bool
}

// Plan is the result of planning an install: the meta-types to write (in the
// import format), the per-type outcomes and the new reference copies.
type Plan struct {
	Catalog *model.CatalogExport
	Entries []model.DomainInstallEntry
	// Bases: package version to record for each type (types skipped excluded).
	Bases map[uuid.UUID]Snapshot
	// Kinds: entity of each type of Bases (item_types, collection_types, relation_types).
	Kinds map[uuid.UUID]string
}

type typeRef struct {
	entity string
	id     uuid.UUID
	name   string
	value  any
}

// PlanInstall merges pkg into the instance described by st.
func PlanInstall(pkg *model.DomainPackage, st State) (*Plan, error) {
	plan := &Plan{
		Catalog: &model.CatalogExport{Format: model.CatalogFormat, Version: model.ExportVersion},
		Entries: []model.DomainInstallEntry{},
		Bases:   map[uuid.UUID]Snapshot{},
		Kinds:   map[uuid.UUID]string{},
	}
	var refs []typeRef
	for _, t := range pkg.ItemTypes {
		refs = append(refs, typeRef{model.EntityItemTypes, t.ID, t.Name, t})
	}
	for _, t := range pkg.CollectionTypes {
		refs = append(refs, typeRef{model.EntityCollectionTypes, t.ID, t.Name, t})
	}
	for _, t := range pkg.RelationTypes {
		refs = append(refs, typeRef{model.EntityRelationTypes, t.ID, t.Name, t})
	}

	// Item types present after the install: those of the instance, plus those of
	// the package that are not skipped (deleted locally since their install).
	known := map[string]bool{}
	for id := range st.ItemTypes {
		known[id.String()] = true
	}
	for _, t := range pkg.ItemTypes {
		_, local := st.Local[t.ID]
		_, installed := st.Base[t.ID]
		if local || !installed {
			known[t.ID.String()] = true
		}
	}

	for _, r := range refs {
		next, err := SnapshotOf(r.value)
		if err != nil {
			return nil, err
		}
		// Version du paquet telle qu'elle s'applique ici (types absents retirés) : c'est elle qui sert
		// de référence au prochain passage, pour qu'un type installé plus tard (ex. Series) soit ajouté.
		applied := clone(next)
		keepKnown(applied, known)
		local, hasLocal := st.Local[r.id]
		base, hasBase := st.Base[r.id]
		entry := model.DomainInstallEntry{Entity: r.entity, ID: r.id, Name: r.name}

		var result Snapshot
		switch {
		case !hasLocal && hasBase:
			entry.Outcome = model.DomainSkipped
			plan.Entries = append(plan.Entries, entry)
			continue
		case !hasLocal:
			entry.Outcome = model.DomainCreated
			result = clone(next)
		case hasBase && equal(local, base):
			entry.Outcome = model.DomainUpdated
			result = clone(next)
		default:
			var b any
			if hasBase {
				b = base
			}
			merged, conflicts := Merge3(b, local, next)
			result, _ = merged.(Snapshot)
			entry.Conflicts = conflicts
			entry.Outcome = model.DomainMerged
		}
		for _, id := range keepKnown(result, known) {
			if u, err := uuid.Parse(id); err == nil {
				entry.MissingTypes = append(entry.MissingTypes, u)
			}
		}
		// Relation sans source ou sans cible présente (domaine lié pas encore installé) : pas
		// installée, ni mémorisée ; elle le sera quand ses types existeront.
		if r.entity == model.EntityRelationTypes && (isEmpty(result["source_item_type_ids"]) || isEmpty(result["target_item_type_ids"])) {
			entry.Outcome = model.DomainSkipped
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		if hasLocal && equal(result, local) {
			entry.Outcome = model.DomainUnchanged
		}
		if name, _ := result["name"].(string); name != "" {
			entry.Name = name
		}
		plan.Entries = append(plan.Entries, entry)
		plan.Bases[r.id] = applied
		plan.Kinds[r.id] = r.entity
		if entry.Outcome == model.DomainUnchanged {
			continue
		}
		if err := addToCatalog(plan.Catalog, r.entity, r.id, result); err != nil {
			return nil, err
		}
	}
	return plan, nil
}

func isEmpty(v any) bool {
	a, _ := v.([]any)
	return len(a) == 0
}

func clone(s Snapshot) Snapshot {
	raw, _ := json.Marshal(s)
	var c Snapshot
	_ = json.Unmarshal(raw, &c)
	return c
}

func addToCatalog(cat *model.CatalogExport, entity string, id uuid.UUID, s Snapshot) error {
	s = clone(s)
	s["id"] = id.String()
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	switch entity {
	case model.EntityItemTypes:
		var t model.ExportItemType
		err = json.Unmarshal(raw, &t)
		cat.ItemTypes = append(cat.ItemTypes, t)
	case model.EntityCollectionTypes:
		var t model.ExportCollectionType
		err = json.Unmarshal(raw, &t)
		cat.CollectionTypes = append(cat.CollectionTypes, t)
	case model.EntityRelationTypes:
		var t model.ExportRelationType
		err = json.Unmarshal(raw, &t)
		cat.RelationTypes = append(cat.RelationTypes, t)
	default:
		err = fmt.Errorf("unknown entity %q", entity)
	}
	return err
}

// Title returns the package title in lang, falling back to French then the id.
func Title(p *model.DomainPackage, lang string) string {
	for _, l := range []string{lang, "fr", "en"} {
		if t := strings.TrimSpace(p.Title[l]); t != "" {
			return t
		}
	}
	return p.ID
}
