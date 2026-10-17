//go:build integration

// Integration tests against a real PostgreSQL. They create a throw-away
// database (colec_test) next to the one given by DATABASE_URL, apply the
// migrations and the seed, and drop it at the end.
//
//	cd server && go test -tags integration ./internal/store/
package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/dbmigrate"
	"github.com/shinsaber/colec/internal/itemquery"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

const (
	testDBName    = "colec_test"
	migrationsDir = "../../../db/migrations"
	seedFile      = "../../../db/seed.sql"
)

// Seed identifiers (db/seed.sql).
var (
	adminUser     = uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	shinsaberUser = uuid.MustParse("a0000000-0000-0000-0000-000000000002")
	typeSerie     = uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	typeSaison    = uuid.MustParse("b0000000-0000-0000-0000-000000000003")
	typeEpisode   = uuid.MustParse("b0000000-0000-0000-0000-000000000004")
	rtEditeEn     = uuid.MustParse("c0000000-0000-0000-0000-000000000004")
	rtInclutEp    = uuid.MustParse("c0000000-0000-0000-0000-000000000005")
	itemSerie     = uuid.MustParse("e0000000-0000-0000-0000-000000000001")
	itemDiffusion = uuid.MustParse("e0000000-0000-0000-0000-000000000002")
	itemSaison1   = uuid.MustParse("e0000000-0000-0000-0000-000000000003")
	itemPilot     = uuid.MustParse("e0000000-0000-0000-0000-000000000010")
	itemDVD       = uuid.MustParse("e0000000-0000-0000-0000-000000000020")
	colMesDVDs    = uuid.MustParse("f0000000-0000-0000-0000-000000000001")
)

var (
	ctx = context.Background()
	s   *store.PgStore
	// baseURL is the DATABASE_URL the test databases are created next to.
	baseURL string
)

func TestMain(m *testing.M) {
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = "postgres://colec:colec@localhost:5432/colec?sslmode=disable"
	}
	baseURL = base
	u, err := url.Parse(base)
	if err != nil {
		fmt.Fprintln(os.Stderr, "DATABASE_URL:", err)
		os.Exit(1)
	}
	u.Path = "/" + testDBName
	testURL := u.String()

	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect (is PostgreSQL running? pg-start):", err)
		os.Exit(1)
	}
	if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+testDBName); err != nil {
		fmt.Fprintln(os.Stderr, "drop:", err)
		os.Exit(1)
	}
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+testDBName); err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}

	code := func() int {
		if err := dbmigrate.Migrate(ctx, testURL, migrationsDir); err != nil {
			fmt.Fprintln(os.Stderr, "migrate:", err)
			return 1
		}
		s, err = store.NewPgStore(ctx, testURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "store:", err)
			return 1
		}
		defer s.Close()
		// Types : paquets de domaine (P3-0) utilisés par seed.sql, puis les données de démonstration.
		if _, err := s.InstallDomains(ctx, "collections", "tv"); err != nil {
			fmt.Fprintln(os.Stderr, "domains:", err)
			return 1
		}
		if err := dbmigrate.Seed(ctx, testURL, seedFile); err != nil {
			fmt.Fprintln(os.Stderr, "seed:", err)
			return 1
		}
		return m.Run()
	}()

	_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+testDBName+" WITH (FORCE)")
	_ = admin.Close(ctx)
	os.Exit(code)
}

func TestPing(t *testing.T) {
	if err := s.Ping(ctx); err != nil {
		t.Fatal(err)
	}
}

// --- Graph ---

func TestGraph_MultiPath(t *testing.T) {
	g, err := s.GetItemGraph(ctx, itemPilot, 2, 500, nil)
	if err != nil {
		t.Fatal(err)
	}
	if g.Root.ID != itemPilot || g.Root.Label != "1. Pilot" { // x-label "{number}. {title}"
		t.Fatalf("root = %+v", g.Root)
	}
	if g.Truncated {
		t.Error("unexpected truncation")
	}
	depths := map[uuid.UUID]int{}
	for _, n := range g.Nodes {
		depths[n.ID] = n.Depth
	}
	// Pilot is reached through Saison 1 (Diffusion path) AND through the DVD.
	want := map[uuid.UUID]int{itemSaison1: -1, itemDVD: -1, itemDiffusion: -2, itemSerie: -2}
	for id, d := range want {
		if got, ok := depths[id]; !ok || got != d {
			t.Errorf("node %s: depth %d (found %v), want %d", id, got, ok, d)
		}
	}
	if len(g.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(g.Nodes))
	}
	incoming := 0
	for _, e := range g.Edges {
		if e.TargetID == itemPilot {
			incoming++
		}
	}
	if incoming != 2 {
		t.Errorf("Pilot must have 2 incoming edges (Saison + DVD), got %d", incoming)
	}
}

func TestGraph_DepthAndFilter(t *testing.T) {
	g, err := s.GetItemGraph(ctx, itemSerie, 1, 500, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) != 2 { // Diffusion + DVD
		t.Errorf("depth 1 from Serie: expected 2 nodes, got %d", len(g.Nodes))
	}
	for _, n := range g.Nodes {
		if n.Depth != 1 {
			t.Errorf("node %s depth %d, want 1", n.Label, n.Depth)
		}
	}

	only, err := s.GetItemGraph(ctx, itemSerie, 5, 500, []uuid.UUID{rtEditeEn, rtInclutEp})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range only.Nodes {
		if n.Type == "Broadcast" || n.Type == "Season" {
			t.Errorf("relation filter ignored: reached %s", n.Type)
		}
	}
	if len(only.Nodes) != 4 { // DVD + 3 episodes
		t.Errorf("filtered graph: expected 4 nodes, got %d", len(only.Nodes))
	}
}

func TestGraph_Truncated(t *testing.T) {
	g, err := s.GetItemGraph(ctx, itemSerie, 5, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !g.Truncated {
		t.Error("expected truncated with maxNodes=2")
	}
	if len(g.Nodes) != 1 {
		t.Errorf("expected 1 node besides root, got %d", len(g.Nodes))
	}
}

func TestGraph_NotFound(t *testing.T) {
	if _, err := s.GetItemGraph(ctx, uuid.New(), 2, 500, nil); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- Items ---

func TestListItems_Filters(t *testing.T) {
	pg := model.Pagination{Page: 1, PerPage: 20}

	byType, total, err := s.ListItems(ctx, pg, store.ItemFilters{ItemTypeID: &typeSerie})
	if err != nil || total != 1 || len(byType) != 1 || byType[0].ID != itemSerie {
		t.Fatalf("type filter: %d items, total %d, err %v", len(byType), total, err)
	}
	if byType[0].ItemType == nil || byType[0].ItemType.Name != "Series" {
		t.Error("item type must be embedded")
	}

	_, total, err = s.ListItems(ctx, pg, store.ItemFilters{Search: "breaking bad"})
	if err != nil || total != 2 { // the series and the DVD box set
		t.Errorf("search: total %d, err %v", total, err)
	}

	// A search without any letter or digit carries no term: no filter.
	_, total, err = s.ListItems(ctx, pg, store.ItemFilters{Search: "%"})
	if err != nil || total != 8 {
		t.Errorf("search %%: total %d, err %v", total, err)
	}

	tagged, total, err := s.ListItems(ctx, pg, store.ItemFilters{Tag: "DRAMA"})
	if err != nil || total != 1 || tagged[0].ID != itemSerie {
		t.Errorf("tag filter (case-insensitive): total %d, err %v", total, err)
	}
	if len(tagged[0].Tags) != 3 { // drama, thriller, crime
		t.Errorf("tags must be loaded, got %v", tagged[0].Tags)
	}

	page, total, err := s.ListItems(ctx, model.Pagination{Page: 1, PerPage: 3}, store.ItemFilters{Sort: "-created_at"})
	if err != nil || total != 8 || len(page) != 3 {
		t.Errorf("pagination: %d items, total %d, err %v", len(page), total, err)
	}
}

func TestItem_CreateUpdateDeleteCascade(t *testing.T) {
	item, err := s.CreateItem(ctx, model.ItemCreate{
		ItemTypeID: typeEpisode,
		Attributes: json.RawMessage(`{"number": 99, "title": "Temporaire", "duration": 1}`),
		Tags:       []string{"drama", "new-tag"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(item.Tags) != 2 {
		t.Errorf("expected 2 tags, got %v", item.Tags)
	}

	rel, err := s.CreateRelation(ctx, model.ItemRelationCreate{
		RelationTypeID: rtInclutEp, SourceItemID: itemDVD, TargetItemID: item.ID,
		Attributes: json.RawMessage(`{"disc": 9, "track": 9}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	noTags := []string{}
	updated, err := s.UpdateItem(ctx, item.ID, model.ItemUpdate{
		Attributes: json.RawMessage(`{"number": 99, "title": "Renommé"}`),
		Tags:       &noTags,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated.Attributes), "Renommé") || len(updated.Tags) != 0 {
		t.Errorf("update not applied: %s / %v", updated.Attributes, updated.Tags)
	}

	if err := s.DeleteItem(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetItem(ctx, item.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("item still readable after delete: %v", err)
	}
	if _, err := s.GetRelation(ctx, rel.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("relation must be cascaded on item delete: %v", err)
	}
	if err := s.DeleteItem(ctx, item.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("second delete: expected ErrNotFound, got %v", err)
	}
}

func TestItem_UnknownType(t *testing.T) {
	_, err := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: uuid.New(), Attributes: json.RawMessage(`{}`)})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown item type, got %v", err)
	}
}

// --- Relations ---

func TestRelation_Conflicts(t *testing.T) {
	_, err := s.CreateRelation(ctx, model.ItemRelationCreate{
		RelationTypeID: rtEditeEn, SourceItemID: itemSerie, TargetItemID: itemDVD, Attributes: json.RawMessage(`{}`),
	})
	if !errors.Is(err, store.ErrConflict) {
		t.Errorf("duplicate relation: expected ErrConflict, got %v", err)
	}
	_, err = s.CreateRelation(ctx, model.ItemRelationCreate{
		RelationTypeID: rtEditeEn, SourceItemID: itemSerie, TargetItemID: uuid.New(), Attributes: json.RawMessage(`{}`),
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown target: expected ErrNotFound, got %v", err)
	}

	rels, err := s.ListItemRelations(ctx, itemPilot, nil)
	if err != nil || len(rels) != 2 {
		t.Errorf("Pilot relations: %d, err %v", len(rels), err)
	}
	rels, err = s.ListItemRelations(ctx, itemPilot, &rtInclutEp)
	if err != nil || len(rels) != 1 {
		t.Errorf("Pilot relations filtered: %d, err %v", len(rels), err)
	}
}

// --- Relation types ---

func TestRelationType_Junctions(t *testing.T) {
	rt, err := s.CreateRelationType(ctx, model.RelationTypeCreate{
		Name:              "test_junctions",
		SourceItemTypeIDs: []uuid.UUID{typeSerie},
		TargetItemTypeIDs: []uuid.UUID{typeEpisode},
		AttributesSchema:  json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rt.SourceItemTypeIDs) != 1 || len(rt.TargetItemTypeIDs) != 1 {
		t.Fatalf("junctions not loaded: %+v", rt)
	}

	updated, err := s.UpdateRelationType(ctx, rt.ID, model.RelationTypeUpdate{
		SourceItemTypeIDs: []uuid.UUID{typeSerie, typeSaison},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.SourceItemTypeIDs) != 2 || len(updated.TargetItemTypeIDs) != 1 {
		t.Errorf("junctions after update: %d sources, %d targets", len(updated.SourceItemTypeIDs), len(updated.TargetItemTypeIDs))
	}

	bySource, _, err := s.ListRelationTypes(ctx, model.Pagination{Page: 1, PerPage: 50}, &typeSaison, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range bySource {
		if r.ID == rt.ID {
			found = true
		}
	}
	if !found {
		t.Error("ListRelationTypes(source=Saison) must include the updated type")
	}

	if _, err := s.CreateRelationType(ctx, model.RelationTypeCreate{Name: "test_junctions"}); !errors.Is(err, store.ErrConflict) {
		t.Errorf("duplicate name: expected ErrConflict, got %v", err)
	}
	if _, err := s.UpdateRelationType(ctx, rt.ID, model.RelationTypeUpdate{SourceItemTypeIDs: []uuid.UUID{uuid.New()}}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown item type in junction: expected ErrNotFound, got %v", err)
	}
	if err := s.DeleteRelationType(ctx, rt.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRelationType(ctx, rtInclutEp); !errors.Is(err, store.ErrConflict) {
		t.Errorf("deleting a relation type still in use: expected ErrConflict, got %v", err)
	}
}

// --- Collections ---

func TestCollections(t *testing.T) {
	_, err := s.AddCollectionItem(ctx, colMesDVDs, model.CollectionItemAdd{ItemID: itemDVD, Attributes: json.RawMessage(`{}`)})
	if !errors.Is(err, store.ErrConflict) {
		t.Errorf("duplicate collection item: expected ErrConflict, got %v", err)
	}

	mine, err := s.ListItemCollections(ctx, shinsaberUser, itemSerie)
	if err != nil || len(mine) != 1 || mine[0].Collection.Name != "Séries vues" {
		t.Errorf("memberships for owner: %+v, err %v", mine, err)
	}
	other, err := s.ListItemCollections(ctx, adminUser, itemSerie)
	if err != nil || len(other) != 0 {
		t.Errorf("memberships must be filtered by user: %+v, err %v", other, err)
	}

	_, total, err := s.ListCollections(ctx, adminUser, model.Pagination{Page: 1, PerPage: 20}, nil)
	if err != nil || total != 0 {
		t.Errorf("admin has no collection in the seed: total %d, err %v", total, err)
	}
	cols, total, err := s.ListCollections(ctx, shinsaberUser, model.Pagination{Page: 1, PerPage: 20}, nil)
	if err != nil || total != 2 {
		t.Fatalf("shinsaber collections: total %d, err %v", total, err)
	}
	for _, c := range cols {
		if c.ItemCount != 1 {
			t.Errorf("collection %s: item_count %d, want 1", c.Name, c.ItemCount)
		}
	}

	items, total, err := s.ListCollectionItems(ctx, colMesDVDs, model.Pagination{Page: 1, PerPage: 20}, nil, "")
	if err != nil || total != 1 || items[0].Item == nil || items[0].Item.ID != itemDVD {
		t.Errorf("collection items: total %d, err %v", total, err)
	}
}

// --- Refresh tokens ---

func TestRefreshTokens(t *testing.T) {
	hash := strings.Repeat("a", 64)
	if err := s.CreateRefreshToken(ctx, shinsaberUser, hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	userID, err := s.ConsumeRefreshToken(ctx, hash)
	if err != nil || userID != shinsaberUser {
		t.Fatalf("consume: %s, %v", userID, err)
	}
	if _, err := s.ConsumeRefreshToken(ctx, hash); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("second consume (rotation): expected ErrNotFound, got %v", err)
	}

	expired := strings.Repeat("b", 64)
	if err := s.CreateRefreshToken(ctx, shinsaberUser, expired, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConsumeRefreshToken(ctx, expired); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expired token: expected ErrNotFound, got %v", err)
	}
	n, err := s.DeleteExpiredRefreshTokens(ctx)
	if err != nil || n < 1 {
		t.Errorf("purge: %d, %v", n, err)
	}
	if err := s.DeleteRefreshToken(ctx, strings.Repeat("c", 64)); err != nil {
		t.Errorf("deleting an unknown token must not fail: %v", err)
	}
	if err := s.CreateRefreshToken(ctx, uuid.New(), strings.Repeat("d", 64), time.Now().Add(time.Hour)); err == nil {
		t.Error("expected a foreign key error for an unknown user")
	}
}

// --- Users ---

func TestUserByUsername_CaseInsensitive(t *testing.T) {
	for _, name := range []string{"admin", "Admin", "ADMIN"} {
		u, _, err := s.GetUserByUsername(ctx, name)
		if err != nil || u.ID != adminUser || u.Username != "admin" {
			t.Errorf("%s: %+v, %v", name, u, err)
		}
	}

	// Migration 004: two accounts differing only by case are refused.
	u, _ := url.Parse(baseURL)
	u.Path = "/" + testDBName
	conn, err := pgx.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	_, err = conn.Exec(ctx,
		`INSERT INTO users (username, email, password, role) VALUES ('Shinsaber', 'dup@example.org', 'x', 'user')`)
	if err == nil || !strings.Contains(err.Error(), "users_username_lower_key") {
		t.Errorf("expected a unique violation on lower(username), got %v", err)
	}
}

// --- Offline snapshot ---

func TestGetOfflineSnapshot(t *testing.T) {
	snap, err := s.GetOfflineSnapshot(ctx, shinsaberUser)
	if err != nil {
		t.Fatal(err)
	}
	if snap.User.Username != "shinsaber" || len(snap.ItemTypes) == 0 || len(snap.CollectionTypes) == 0 {
		t.Fatalf("snapshot head: %+v", snap.User)
	}
	cols := map[uuid.UUID]bool{}
	for _, c := range snap.Collections {
		if c.UserID != shinsaberUser || c.CollectionType != nil {
			t.Errorf("collection %+v", c)
		}
		cols[c.ID] = true
	}
	items := map[uuid.UUID]bool{}
	for _, it := range snap.Items {
		if items[it.ID] {
			t.Errorf("item %s listed twice", it.ID)
		}
		items[it.ID] = true
	}
	if !cols[colMesDVDs] || !items[itemDVD] || !items[itemSerie] {
		t.Errorf("expected Mes DVDs, the DVD and the series: %d collections, %d items", len(cols), len(items))
	}
	for _, e := range snap.Entries {
		if !cols[e.CollectionID] || !items[e.ItemID] {
			t.Errorf("entry outside the snapshot: %+v", e)
		}
	}
	// Admin has no collection in the seed: empty lists, not null.
	empty, err := s.GetOfflineSnapshot(ctx, adminUser)
	if err != nil || empty.Collections == nil || empty.Items == nil || empty.Entries == nil || len(empty.Items) != 0 {
		t.Errorf("admin: %+v, %v", empty, err)
	}
}

// --- Relation list badge ---

func TestListRelatedItemCollections(t *testing.T) {
	// Series → DVD (released_as) ; the DVD is in shinsaber's « Mes DVDs ».
	got, err := s.ListRelatedItemCollections(ctx, shinsaberUser, itemSerie)
	if err != nil {
		t.Fatal(err)
	}
	var dvd *model.RelatedItemCollections
	for i := range got {
		if got[i].ItemID == itemDVD {
			dvd = &got[i]
		}
	}
	if dvd == nil || len(dvd.Collections) != 1 || dvd.Collections[0].ID != colMesDVDs ||
		!strings.Contains(string(dvd.Collections[0].Attributes), `"condition"`) {
		t.Errorf("shinsaber: %+v", got)
	}
	// Another user's collections are not reported.
	if got, err := s.ListRelatedItemCollections(ctx, adminUser, itemSerie); err != nil || len(got) != 0 {
		t.Errorf("admin: %+v, %v", got, err)
	}
}

// --- Uploads ---

func TestReferencedUploads(t *testing.T) {
	item, err := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: typeSerie, Attributes: json.RawMessage(`{
		"title": "Uploads",
		"images": {"default_image": "/api/v1/uploads/a.jpg", "poster": ["/api/v1/uploads/a.jpg", "http://host/api/v1/uploads/b.png"]}
	}`)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.DeleteItem(ctx, item.ID) }()
	names, err := s.ReferencedUploads(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(names, "a.jpg") || !slices.Contains(names, "b.png") || slices.Contains(names, "Uploads") {
		t.Errorf("referenced uploads: %v", names)
	}
	if n := len(slices.Compact(slices.Sorted(slices.Values(names)))); n != len(names) {
		t.Errorf("duplicates in %v", names)
	}
}

// --- Item types: display order ---

func TestItemTypes_DisplayOrder(t *testing.T) {
	names := func() []string {
		its, _, err := s.ListItemTypes(ctx, model.Pagination{Page: 1, PerPage: 100})
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, len(its))
		for i, it := range its {
			out[i] = it.Name
		}
		return out
	}
	// Seed values (migration 005 / seed.sql): main types first, bulky ones last.
	want := []string{"Series", "VideoRelease", "Season", "Broadcast", "Episode"}
	seedOnly := slices.DeleteFunc(names(), func(n string) bool { return !slices.Contains(want, n) })
	if got := seedOnly; !slices.Equal(got, want) {
		t.Errorf("list order = %v, want %v", got, want)
	}
	stats, err := s.GetStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.ItemsByType) == 0 || stats.ItemsByType[0].TypeName != "Series" || stats.ItemsByType[0].TypeID != typeSerie {
		t.Errorf("stats order: %+v", stats.ItemsByType)
	}

	created, err := s.CreateItemType(ctx, model.ItemTypeCreate{Name: "Zine", AttributesSchema: []byte(`{"type":"object"}`)})
	if err != nil || created.DisplayOrder != 100 {
		t.Fatalf("default display_order: %+v, %v", created, err)
	}
	defer func() { _ = s.DeleteItemType(ctx, created.ID) }()
	first := 1
	updated, err := s.UpdateItemType(ctx, created.ID, model.ItemTypeUpdate{DisplayOrder: &first})
	if err != nil || updated.DisplayOrder != 1 || updated.Name != "Zine" {
		t.Fatalf("update display_order: %+v, %v", updated, err)
	}
	if got := names(); got[0] != "Zine" { // other tests may add types (default 100)
		t.Errorf("after update, list starts with %v", got)
	}
}

// --- Stats & tags ---

func TestStatsAndTags(t *testing.T) {
	st, err := s.GetStats(ctx)
	if err != nil || st.TotalItems < 8 || st.TotalCollections != 2 {
		t.Errorf("stats: %+v, err %v", st, err)
	}
	tags, err := s.ListTags(ctx, "dr")
	if err != nil || len(tags) != 1 || tags[0].Name != "drama" {
		t.Errorf("tags prefix search: %v, err %v", tags, err)
	}
	tags, err = s.ListTags(ctx, "%")
	if err != nil || len(tags) != 0 {
		t.Errorf("tag wildcard must be literal: %v, err %v", tags, err)
	}
}

// --- Advanced search (lot 14) ---

// ids lists the ids of a page, in order.
func ids(items []model.Item) []uuid.UUID {
	out := make([]uuid.UUID, len(items))
	for i := range items {
		out[i] = items[i].ID
	}
	return out
}

func TestListItems_FullText(t *testing.T) {
	pg := model.Pagination{Page: 1, PerPage: 20}
	cases := []struct {
		search string
		want   int
	}{
		{"break", 2},          // prefix
		{"BREAKING bad", 2},   // case, all terms required
		{"integrale", 1},      // accents ignored (« Intégrale »)
		{"chimie", 1},         // i18n-text, French
		{"chemistry", 1},      // i18n-text, English
		{"2008", 3},           // numbers: series, broadcast, season 1
		{"Breaking%Bad", 2},   // punctuation separates terms, never a wildcard
		{"uploads", 0},        // image URLs are not indexed
		{"breaking pilot", 0}, // no item has both terms
	}
	for _, c := range cases {
		_, total, err := s.ListItems(ctx, pg, store.ItemFilters{Search: c.search})
		if err != nil || total != c.want {
			t.Errorf("search %q: total %d, want %d (err %v)", c.search, total, c.want, err)
		}
	}
	// Relevance first by default: the series title matches better than the DVD's.
	page, _, err := s.ListItems(ctx, pg, store.ItemFilters{Search: "breaking bad"})
	if err != nil || len(page) != 2 {
		t.Fatalf("relevance: %v %v", ids(page), err)
	}
}

func TestListItems_AttributeFilters(t *testing.T) {
	pg := model.Pagination{Page: 1, PerPage: 20}
	typeBroadcast := uuid.MustParse("b0000000-0000-0000-0000-000000000002")
	itemSaison2 := uuid.MustParse("e0000000-0000-0000-0000-000000000004")
	f := func(key string, kind itemquery.Kind, op itemquery.Op, values ...any) itemquery.Filter {
		return itemquery.Filter{Key: key, Kind: kind, Op: op, Values: values}
	}
	cases := []struct {
		name    string
		filters store.ItemFilters
		want    []uuid.UUID
	}{
		{"enum equality (OR)", store.ItemFilters{ItemTypeID: &typeBroadcast,
			Attrs: []itemquery.Filter{f("type", itemquery.KindString, itemquery.OpEq, "tv", "cable")}},
			[]uuid.UUID{itemDiffusion}},
		{"enum equality, no match", store.ItemFilters{ItemTypeID: &typeBroadcast,
			Attrs: []itemquery.Filter{f("type", itemquery.KindString, itemquery.OpEq, "web")}}, nil},
		{"integer range", store.ItemFilters{ItemTypeID: &typeSaison,
			Attrs: []itemquery.Filter{f("year", itemquery.KindInteger, itemquery.OpGte, int64(2009))}},
			[]uuid.UUID{itemSaison2}},
		{"integer equality", store.ItemFilters{ItemTypeID: &typeEpisode,
			Attrs: []itemquery.Filter{f("duration", itemquery.KindInteger, itemquery.OpEq, int64(58))}},
			[]uuid.UUID{itemPilot}},
		{"text contains, accents ignored", store.ItemFilters{ItemTypeID: &typeSaison,
			Attrs: []itemquery.Filter{f("name", itemquery.KindString, itemquery.OpContains, "SAISON 1")}},
			[]uuid.UUID{itemSaison1}},
		{"contains is literal", store.ItemFilters{ItemTypeID: &typeSaison,
			Attrs: []itemquery.Filter{f("name", itemquery.KindString, itemquery.OpContains, "%")}}, nil},
		{"i18n contains, any language", store.ItemFilters{ItemTypeID: &typeEpisode,
			Attrs: []itemquery.Filter{f("synopsis", itemquery.KindI18n, itemquery.OpContains, "diagnosis")}},
			[]uuid.UUID{itemPilot}},
		{"related to the series", store.ItemFilters{RelatedTo: &itemSerie,
			Sort: "created_at"}, []uuid.UUID{itemDiffusion, itemDVD}},
		{"related through one relation type", store.ItemFilters{RelatedTo: &itemSerie,
			RelationTypeID: &rtEditeEn}, []uuid.UUID{itemDVD}},
		{"related, both directions", store.ItemFilters{RelatedTo: &itemSaison1,
			ItemTypeID: &typeEpisode, Attrs: []itemquery.Filter{f("number", itemquery.KindInteger, itemquery.OpLte, int64(1))}},
			[]uuid.UUID{itemPilot}},
	}
	for _, c := range cases {
		page, total, err := s.ListItems(ctx, pg, c.filters)
		if err != nil || total != len(c.want) || !reflect.DeepEqual(ids(page), append([]uuid.UUID{}, c.want...)) {
			t.Errorf("%s: got %v (total %d), want %v, err %v", c.name, ids(page), total, c.want, err)
		}
	}
}

func TestListItems_AttributeSort(t *testing.T) {
	pg := model.Pagination{Page: 1, PerPage: 20}
	page, _, err := s.ListItems(ctx, pg, store.ItemFilters{ItemTypeID: &typeEpisode,
		SortAttr: &itemquery.Sort{Key: "duration", Numeric: true, Desc: true}})
	if err != nil || len(page) != 3 || page[0].ID != itemPilot {
		t.Fatalf("numeric sort desc: %v %v", ids(page), err)
	}
	// Numeric, not lexicographic: 9 < 10 would fail as text; here 48 < 58 and ties by id.
	page, _, err = s.ListItems(ctx, pg, store.ItemFilters{ItemTypeID: &typeEpisode,
		SortAttr: &itemquery.Sort{Key: "number", Numeric: true}})
	if err != nil || len(page) != 3 || page[0].ID != itemPilot || page[2].ID != uuid.MustParse("e0000000-0000-0000-0000-000000000012") {
		t.Fatalf("numeric sort asc: %v %v", ids(page), err)
	}
	page, _, err = s.ListItems(ctx, pg, store.ItemFilters{ItemTypeID: &typeSaison,
		SortAttr: &itemquery.Sort{Key: "name", Desc: true}})
	if err != nil || len(page) != 2 || page[1].ID != itemSaison1 {
		t.Fatalf("text sort desc: %v %v", ids(page), err)
	}
}

func TestCreateUser(t *testing.T) {
	u, err := s.CreateUser(ctx, "Newcomer", "newcomer@example.org", "hash", "admin")
	if err != nil || u.Username != "Newcomer" || u.Role != "admin" {
		t.Fatalf("create: %+v, %v", u, err)
	}
	if got, _, err := s.GetUserByUsername(ctx, "newcomer"); err != nil || got.ID != u.ID {
		t.Errorf("lookup: %+v, %v", got, err)
	}
	// Same name in another case, or same email: conflict (migration 004).
	if _, err := s.CreateUser(ctx, "NEWCOMER", "other@example.org", "hash", "user"); !errors.Is(err, store.ErrConflict) {
		t.Errorf("case duplicate: %v", err)
	}
	if _, err := s.CreateUser(ctx, "other", "newcomer@example.org", "hash", "user"); !errors.Is(err, store.ErrConflict) {
		t.Errorf("email duplicate: %v", err)
	}
}
