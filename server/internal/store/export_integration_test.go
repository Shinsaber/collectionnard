//go:build integration

package store_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/dbmigrate"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

// exportNow exports the catalogue without its timestamp (comparable exports).
func exportNow(t *testing.T, st *store.PgStore) *model.CatalogExport {
	t.Helper()
	cat, err := st.ExportCatalog(ctx)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	cat.ExportedAt = nil
	return cat
}

// normalise round-trips through JSON so that RawMessage formatting (jsonb
// re-serialisation, key order) does not matter in comparisons.
func normalise(t *testing.T, cat *model.CatalogExport) map[string]any {
	t.Helper()
	b, err := json.Marshal(cat)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func counts(rep *model.ImportReport, entity string) model.ImportCounts {
	return *rep.Counts[entity]
}

func TestExportCatalog_Seed(t *testing.T) {
	cat := exportNow(t, s)
	if cat.Format != model.CatalogFormat || len(cat.ItemTypes) != 5 || len(cat.RelationTypes) != 5 ||
		len(cat.CollectionTypes) != 3 || len(cat.Items) != 8 || len(cat.Relations) != 10 {
		t.Fatalf("unexpected sizes: %d types, %d relation types, %d collection types, %d items, %d relations",
			len(cat.ItemTypes), len(cat.RelationTypes), len(cat.CollectionTypes), len(cat.Items), len(cat.Relations))
	}
	for i := 1; i < len(cat.Items); i++ {
		if cat.Items[i-1].ID.String() >= cat.Items[i].ID.String() {
			t.Fatal("items must be sorted by id")
		}
	}
	if cat.Items[0].ID != itemSerie || !reflect.DeepEqual(cat.Items[0].Tags, []string{"crime", "drama", "thriller"}) {
		t.Errorf("series tags: %v", cat.Items[0].Tags)
	}
	for _, rt := range cat.RelationTypes {
		if rt.ID == rtEditeEn && (len(rt.SourceItemTypeIDs) != 1 || rt.SourceItemTypeIDs[0] != typeSerie) {
			t.Errorf("released_as sources: %v", rt.SourceItemTypeIDs)
		}
	}
}

// Re-importing an export changes nothing and reports every entity unchanged.
func TestImportCatalog_Idempotent(t *testing.T) {
	before := exportNow(t, s)
	rep, err := s.ImportCatalog(ctx, before, false)
	if err != nil || len(rep.Errors) != 0 || !rep.Applied {
		t.Fatalf("import: %v %+v", err, rep)
	}
	for entity, c := range rep.Counts {
		if c.Created != 0 || c.Updated != 0 {
			t.Errorf("%s: %+v, want everything unchanged", entity, *c)
		}
	}
	if counts(rep, model.EntityItems).Unchanged != 8 {
		t.Errorf("items: %+v", counts(rep, model.EntityItems))
	}
	if !reflect.DeepEqual(normalise(t, before), normalise(t, exportNow(t, s))) {
		t.Error("export after re-import differs")
	}
}

// Lot 12 acceptance: export, empty database, import reproduces the catalogue.
func TestImportCatalog_RestoresEmptyDatabase(t *testing.T) {
	source := exportNow(t, s)

	const dbName = "colec_test_import"
	admin, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+dbName)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+dbName); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+dbName+" WITH (FORCE)") }()
	u, _ := url.Parse(baseURL)
	u.Path = "/" + dbName
	if err := dbmigrate.Migrate(ctx, u.String(), migrationsDir); err != nil {
		t.Fatal(err)
	}
	empty, err := store.NewPgStore(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer empty.Close()

	rep, err := empty.ImportCatalog(ctx, source, true)
	if err != nil || len(rep.Errors) != 0 || rep.Applied || counts(rep, model.EntityItems).Created != 8 {
		t.Fatalf("dry run: %v %+v", err, rep)
	}
	if cat := exportNow(t, empty); len(cat.Items) != 0 {
		t.Fatal("dry run must not write")
	}

	rep, err = empty.ImportCatalog(ctx, source, false)
	if err != nil || len(rep.Errors) != 0 || !rep.Applied {
		t.Fatalf("import: %v %+v", err, rep)
	}
	if c := counts(rep, model.EntityRelations); c.Created != 10 {
		t.Errorf("relations: %+v", c)
	}
	if !reflect.DeepEqual(normalise(t, source), normalise(t, exportNow(t, empty))) {
		t.Error("restored catalogue differs from the source")
	}
	// The restored catalogue is usable: full-text search sees imported items.
	if _, total, err := empty.ListItems(ctx, model.Pagination{Page: 1, PerPage: 20}, store.ItemFilters{Search: "breaking"}); err != nil || total != 2 {
		t.Errorf("search after import: %d %v", total, err)
	}
}

// Any invalid entity rejects the whole file, with one error per entity.
func TestImportCatalog_ValidationErrors(t *testing.T) {
	cat := exportNow(t, s)
	newItem := uuid.New()
	cat.Items = append(cat.Items,
		// valid new item: must not be written since the file has errors
		model.ExportItem{ID: newItem, ItemTypeID: typeSerie, Attributes: json.RawMessage(`{"title": "Better Call Saul"}`)},
		// Series without its required title
		model.ExportItem{ID: uuid.New(), ItemTypeID: typeSerie, Attributes: json.RawMessage(`{"start_year": 2015}`)},
		// unknown item type
		model.ExportItem{ID: uuid.New(), ItemTypeID: uuid.New(), Attributes: json.RawMessage(`{}`)},
	)
	// a new item type reusing an existing name
	cat.ItemTypes = append(cat.ItemTypes, model.ExportItemType{ExportMetaType: model.ExportMetaType{ID: uuid.New(), Name: "Series", AttributesSchema: json.RawMessage(`{"type":"object"}`)}})
	// released_as (Series → DVD) used from an episode
	cat.Relations = append(cat.Relations, model.ExportRelation{
		ID: uuid.New(), RelationTypeID: rtEditeEn, SourceItemID: itemPilot, TargetItemID: itemDVD,
	})
	// tags changed on an existing item
	cat.Items[0].Tags = []string{"drama"}

	rep, err := s.ImportCatalog(ctx, cat, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Applied || len(rep.Errors) != 4 {
		t.Fatalf("expected 4 errors and nothing applied, got %+v", rep)
	}
	want := map[string]string{
		model.EntityItemTypes: "name already used",
		model.EntityItems:     "", // two item errors, checked below
		model.EntityRelations: "source item type is not allowed",
	}
	itemErrors := 0
	for _, e := range rep.Errors {
		switch e.Entity {
		case model.EntityItems:
			itemErrors++
		default:
			if w, ok := want[e.Entity]; !ok || !strings.Contains(e.Message, w) {
				t.Errorf("unexpected error %+v", e)
			}
		}
	}
	if itemErrors != 2 {
		t.Errorf("expected 2 item errors: %+v", rep.Errors)
	}
	if c := counts(rep, model.EntityItems); c.Created != 1 || c.Updated != 1 {
		t.Errorf("items: %+v (1 created, the series' tags updated)", c)
	}
	// Nothing was written.
	if _, err := s.GetItem(ctx, newItem); err == nil {
		t.Error("a rejected import must not create items")
	}
	if it, err := s.GetItem(ctx, itemSerie); err != nil || len(it.Tags) != 3 {
		t.Errorf("a rejected import must not touch tags: %v", err)
	}

	cat.Format = "something"
	if rep, err := s.ImportCatalog(ctx, cat, true); err != nil || len(rep.Errors) != 1 || rep.Errors[0].Entity != model.EntityFile {
		t.Errorf("bad format: %v %+v", err, rep)
	}
}

// --- Collection import ---

// collectionFile builds the export of a collection as GET /collections/{id}/export does.
func collectionFile(t *testing.T, id uuid.UUID) *model.CollectionExport {
	t.Helper()
	col, err := s.GetCollection(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	entries, _, err := s.ListCollectionItems(ctx, id, model.Pagination{Page: 1, PerPage: 100}, nil, "created_at")
	if err != nil {
		t.Fatal(err)
	}
	exp := &model.CollectionExport{Format: model.CollectionFormat, Version: model.ExportVersion,
		Collection: model.CollectionExportHeader{ID: col.ID, Name: col.Name, Description: col.Description,
			CollectionType: model.ExportMetaType{ID: col.CollectionTypeID, Name: col.CollectionType.Name}}}
	for _, e := range entries {
		exp.Items = append(exp.Items, model.CollectionExportItem{ItemID: e.ItemID, Attributes: e.Attributes, AddedAt: e.CreatedAt})
	}
	return exp
}

func TestImportCollection_RestoreOwnCollection(t *testing.T) {
	file := collectionFile(t, colMesDVDs)

	rep, err := s.ImportCollection(ctx, shinsaberUser, file, false)
	if err != nil || !rep.Applied || len(rep.Errors) != 0 || *rep.CollectionID != colMesDVDs {
		t.Fatalf("re-import: %v %+v", err, rep)
	}
	if c := counts(rep, model.EntityItems); c.Unchanged != 1 || c.Created+c.Updated+c.Removed != 0 {
		t.Errorf("items: %+v, want 1 unchanged", c)
	}
	if c := counts(rep, model.EntityCollection); c.Unchanged != 1 {
		t.Errorf("collection: %+v", c)
	}

	// The collection becomes exactly the file: new item added, the DVD's attributes
	// changed, and (dry run) a file without the DVD removes it.
	changed := collectionFile(t, colMesDVDs)
	changed.Items[0].Attributes = json.RawMessage(`{"condition": "good"}`)
	changed.Items = append(changed.Items, model.CollectionExportItem{ItemID: itemSerie, Attributes: json.RawMessage(`{}`)})
	rep, err = s.ImportCollection(ctx, shinsaberUser, changed, true)
	if err != nil || len(rep.Errors) != 0 || rep.Applied {
		t.Fatalf("dry run: %v %+v", err, rep)
	}
	if c := counts(rep, model.EntityItems); c.Created != 1 || c.Updated != 1 {
		t.Errorf("items: %+v, want 1 created, 1 updated", c)
	}
	changed.Items = changed.Items[1:]
	rep, _ = s.ImportCollection(ctx, shinsaberUser, changed, true)
	if c := counts(rep, model.EntityItems); c.Removed != 1 {
		t.Errorf("items: %+v, want 1 removed", c)
	}
	if _, total, _ := s.ListCollectionItems(ctx, colMesDVDs, model.Pagination{Page: 1, PerPage: 10}, nil, ""); total != 1 {
		t.Errorf("dry runs must not write: %d items", total)
	}
}

func TestImportCollection_CopyForAnotherUser(t *testing.T) {
	file := collectionFile(t, colMesDVDs)
	// Another user: never writes into shinsaber's collection, creates its own.
	rep, err := s.ImportCollection(ctx, adminUser, file, false)
	if err != nil || !rep.Applied || *rep.CollectionID == colMesDVDs {
		t.Fatalf("import by admin: %v %+v", err, rep)
	}
	created, err := s.GetCollection(ctx, *rep.CollectionID)
	if err != nil || created.UserID != adminUser || created.Name != "Mes DVDs" {
		t.Fatalf("created: %v %+v", err, created)
	}
	defer func() { _ = s.DeleteCollection(ctx, created.ID) }()

	// Same name, new id, same owner: the name gets a suffix.
	file.Collection.ID = uuid.New()
	rep, err = s.ImportCollection(ctx, adminUser, file, false)
	if err != nil || !rep.Applied || *rep.CollectionID != file.Collection.ID {
		t.Fatalf("second import: %v %+v", err, rep)
	}
	defer func() { _ = s.DeleteCollection(ctx, file.Collection.ID) }()
	if c, _ := s.GetCollection(ctx, file.Collection.ID); c == nil || c.Name != "Mes DVDs (2)" {
		t.Errorf("expected suffixed name, got %+v", c)
	}
}

func TestImportCollection_Errors(t *testing.T) {
	file := collectionFile(t, colMesDVDs)
	file.Items = append(file.Items,
		model.CollectionExportItem{ItemID: uuid.New()},                                                 // not in the catalogue
		model.CollectionExportItem{ItemID: itemDVD},                                                    // listed twice
		model.CollectionExportItem{ItemID: itemPilot, Attributes: json.RawMessage(`{"condition": 3}`)}, // invalid attributes
	)
	rep, err := s.ImportCollection(ctx, shinsaberUser, file, false)
	if err != nil || rep.Applied || len(rep.Errors) != 3 {
		t.Fatalf("expected 3 errors: %v %+v", err, rep)
	}

	file = collectionFile(t, colMesDVDs)
	file.Collection.CollectionType = model.ExportMetaType{ID: uuid.New(), Name: "Nope"}
	rep, _ = s.ImportCollection(ctx, shinsaberUser, file, false)
	if len(rep.Errors) != 1 || rep.Errors[0].Entity != model.EntityCollection {
		t.Errorf("unknown type: %+v", rep)
	}
	// Type resolved by name when its id differs (other installation).
	file.Collection.CollectionType = model.ExportMetaType{ID: uuid.New(), Name: "Ownership"}
	if rep, _ = s.ImportCollection(ctx, shinsaberUser, file, true); len(rep.Errors) != 0 {
		t.Errorf("type by name: %+v", rep)
	}
}

// Migration 006: a collection type limited to some item types refuses new items
// of other types, keeps those already in the collection, and round-trips its list.
func TestCollectionType_AllowedItemTypes(t *testing.T) {
	col, err := s.GetCollection(ctx, colMesDVDs) // Ownership, holds the DVD only
	if err != nil {
		t.Fatal(err)
	}
	ctID := col.CollectionTypeID
	only := []uuid.UUID{typeSerie}
	ct, err := s.UpdateCollectionType(ctx, ctID, model.CollectionTypeUpdate{ItemTypeIDs: &only})
	if err != nil || !slices.Equal(ct.ItemTypeIDs, only) {
		t.Fatalf("update: %+v, %v", ct, err)
	}
	defer func() {
		none := []uuid.UUID{}
		if ct, err := s.UpdateCollectionType(ctx, ctID, model.CollectionTypeUpdate{ItemTypeIDs: &none}); err != nil || len(ct.ItemTypeIDs) != 0 {
			t.Errorf("reset: %+v, %v", ct, err)
		}
	}()
	if got, _ := s.GetCollection(ctx, colMesDVDs); !slices.Equal(got.CollectionType.ItemTypeIDs, only) {
		t.Errorf("embedded collection type: %+v", got.CollectionType)
	}
	unknown := []uuid.UUID{uuid.New()}
	if _, err := s.UpdateCollectionType(ctx, ctID, model.CollectionTypeUpdate{ItemTypeIDs: &unknown}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown item type: %v", err)
	}

	// The DVD already there is kept; the episode is a new entry of a disallowed type.
	file := collectionFile(t, colMesDVDs)
	if rep, _ := s.ImportCollection(ctx, shinsaberUser, file, true); len(rep.Errors) != 0 {
		t.Errorf("existing DVD must be kept: %+v", rep.Errors)
	}
	file.Items = append(file.Items, model.CollectionExportItem{ItemID: itemPilot})
	rep, _ := s.ImportCollection(ctx, shinsaberUser, file, true)
	if len(rep.Errors) != 1 || !strings.Contains(rep.Errors[0].Message, "not allowed") {
		t.Errorf("episode must be refused: %+v", rep.Errors)
	}

	// Catalogue export → import: the list round-trips unchanged.
	cat, err := s.ExportCatalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cat.CollectionTypes {
		if c.ID == ctID && !slices.Equal(c.ItemTypeIDs, only) {
			t.Errorf("export: %+v", c)
		}
	}
	rep, err = s.ImportCatalog(ctx, cat, true)
	if err != nil || len(rep.Errors) != 0 || rep.Counts[model.EntityCollectionTypes].Updated != 0 {
		t.Errorf("re-import: %v %+v", err, rep)
	}
}

// db/seed_bd_manga.sql (lot 13) loads on top of seed.sql and every entity it
// inserts directly in SQL satisfies its JSON Schema (checked by an import dry run).
func TestSeedBDManga_Valid(t *testing.T) {
	const dbName = "colec_test_bd"
	admin, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+dbName)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+dbName); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+dbName+" WITH (FORCE)") }()
	u, _ := url.Parse(baseURL)
	u.Path = "/" + dbName
	if err := dbmigrate.Migrate(ctx, u.String(), migrationsDir); err != nil {
		t.Fatal(err)
	}
	bd, err := store.NewPgStore(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer bd.Close()
	if _, err := bd.InstallDomains(ctx); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{seedFile, "../../../db/seed_bd_manga.sql"} {
		if err := dbmigrate.Seed(ctx, u.String(), f); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}

	rep, err := bd.ImportCatalog(ctx, exportNow(t, bd), true)
	if err != nil || len(rep.Errors) != 0 {
		t.Fatalf("seed BD invalid: %v %+v", err, rep.Errors)
	}
	if c := counts(rep, model.EntityItems); c.Unchanged != 8+21+12 { // TV, BD / Manga, romans
		t.Errorf("items: %+v", c)
	}

	// Multiple roles on a single relation: Uderzo is writer and artist of Astérix.
	uderzo := uuid.MustParse("e0000000-0000-0000-0000-000000000123")
	rels, err := bd.ListItemRelations(ctx, uderzo, nil)
	if err != nil || len(rels) != 1 || string(rels[0].Attributes) != `{"roles": ["writer", "artist"]}` {
		t.Errorf("created_by roles: %v %+v", err, rels)
	}
}
