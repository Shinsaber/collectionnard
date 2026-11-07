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

// Q-006 : GET /tags est borné (100), la recherche par préfixe sert à affiner.
func TestListTags_Limit(t *testing.T) {
	u, _ := url.Parse(baseURL)
	u.Path = "/" + testDBName
	conn, err := pgx.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx,
		`INSERT INTO tags (name) SELECT 'zz-limit-' || lpad(g::text, 3, '0') FROM generate_series(1, 120) g`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = conn.Exec(ctx, `DELETE FROM tags WHERE name LIKE 'zz-limit-%'`) })

	tags, err := s.ListTags(ctx, "zz-limit-")
	if err != nil || len(tags) != 100 || tags[0].Name != "zz-limit-001" || tags[99].Name != "zz-limit-100" {
		t.Errorf("tags limit: %d results, err %v", len(tags), err)
	}
	if all, err := s.ListTags(ctx, ""); err != nil || len(all) != 100 {
		t.Errorf("unfiltered list must be capped too: %d, err %v", len(all), err)
	}
}

// Q-001 : GET /items/{id}/relations est paginé, trié par type puis `order`, et ne joint plus les
// méta-types (ids seulement) ; ListItemRelations (connecteurs) garde la liste complète.
func TestListItemRelationsPage(t *testing.T) {
	all, err := s.ListItemRelations(ctx, itemPilot, nil)
	if err != nil || len(all) != 2 {
		t.Fatalf("all relations: %d, err %v", len(all), err)
	}
	page, total, err := s.ListItemRelationsPage(ctx, itemPilot, nil, model.Pagination{Page: 1, PerPage: 1})
	if err != nil || total != 2 || len(page) != 1 || page[0].ID != all[0].ID {
		t.Errorf("page 1/1: %d of %d, err %v", len(page), total, err)
	}
	page2, _, err := s.ListItemRelationsPage(ctx, itemPilot, nil, model.Pagination{Page: 2, PerPage: 1})
	if err != nil || len(page2) != 1 || page2[0].ID != all[1].ID {
		t.Errorf("page 2/1: %d, err %v", len(page2), err)
	}
	for _, r := range page {
		if r.RelationType != nil || r.SourceItem == nil || r.TargetItem == nil || r.SourceItem.ItemType != nil || r.TargetItem.ItemType != nil {
			t.Errorf("meta-types must not be embedded, ends must: %+v", r)
		}
	}
	filtered, total, err := s.ListItemRelationsPage(ctx, itemPilot, &rtInclutEp, model.Pagination{Page: 1, PerPage: 100})
	if err != nil || total != 1 || len(filtered) != 1 || filtered[0].RelationTypeID != rtInclutEp {
		t.Errorf("filtered page: %d of %d, err %v", len(filtered), total, err)
	}
}

// Avis (I-0) : le SQL réel décide de ce qui est listé, de ce qui compte dans la moyenne, et le paquet de
// domaine ouvre les avis sur le bon type.
func TestReviews(t *testing.T) {
	rt, err := s.CreateReviewType(ctx, model.ReviewTypeCreate{Name: "TestReview",
		AttributesSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"integer","minimum":0,"maximum":10}}}`)})
	if err != nil {
		t.Fatal(err)
	}
	it, err := s.CreateItemType(ctx, model.ItemTypeCreate{Name: "TestReviewed",
		AttributesSchema: json.RawMessage(`{"type":"object"}`), ReviewTypeID: &rt.ID})
	if err != nil {
		t.Fatal(err)
	}
	if it.ReviewTypeID == nil || *it.ReviewTypeID != rt.ID {
		t.Fatalf("review_type_id : %+v", it.ReviewTypeID)
	}
	item, err := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: it.ID, Attributes: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetItemReviewType(ctx, item.ID)
	if err != nil || got == nil || got.ID != rt.ID {
		t.Fatalf("GetItemReviewType : %+v %v", got, err)
	}
	if _, err := s.GetItemReviewType(ctx, uuid.New()); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("item inconnu : %v", err)
	}

	// Quatre auteurs, une visibilité chacun, plus un compte désactivé.
	writers := map[string]uuid.UUID{}
	for _, v := range []string{"private", "members", "anonymous", "public", "disabled"} {
		u, err := s.CreateUser(ctx, "rev_"+v, "rev_"+v+"@example.org", "x", model.RoleUser)
		if err != nil {
			t.Fatal(err)
		}
		writers[v] = u.ID
		vis := v
		if v == "disabled" {
			vis = "public"
		}
		if _, err := s.PutReview(ctx, item.ID, u.ID, model.ReviewWrite{Rating: 8, Body: "texte de " + v, Lang: "fr", Visibility: vis}); err != nil {
			t.Fatal(err)
		}
	}
	now := true
	if _, err := s.UpdateUser(ctx, writers["disabled"], model.UserUpdate{Disabled: &now}); err != nil {
		t.Fatal(err)
	}

	// Remplacement : toujours une ligne par couple, la moyenne suit.
	crit := json.RawMessage(`{"a":6}`)
	mine, err := s.PutReview(ctx, item.ID, writers["public"], model.ReviewWrite{ReviewTypeID: &rt.ID, Criteria: crit, Rating: 6, Body: "texte de public", Lang: "en", Visibility: "public"})
	if err != nil || mine.Rating != 6 || mine.Lang != "en" || mine.Author == nil {
		t.Fatalf("remplacement : %+v %v", mine, err)
	}

	list := func(viewer uuid.UUID, levels []string) []string {
		reviews, total, err := s.ListItemReviews(ctx, item.ID, model.ReviewListParams{Viewer: viewer, Levels: levels}, model.Pagination{Page: 1, PerPage: 20})
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, r := range reviews {
			out = append(out, r.Visibility)
		}
		slices.Sort(out)
		if total != len(out) {
			t.Errorf("total %d pour %d lignes", total, len(out))
		}
		return out
	}
	if got := list(uuid.Nil, []string{"public"}); !slices.Equal(got, []string{"anonymous", "public"}) {
		t.Errorf("visiteur : %v", got)
	}
	reader := writers["private"]
	if got := list(reader, []string{"members", "public"}); !slices.Equal(got, []string{"anonymous", "members", "public"}) {
		t.Errorf("membre : %v", got)
	}
	if got := list(writers["public"], []string{"members", "public"}); !slices.Equal(got, []string{"anonymous", "members"}) {
		t.Errorf("son propre avis n'est pas dans la liste : %v", got)
	}

	// La moyenne porte sur toutes les notes, sauf le compte désactivé : (8+8+8+6)/4.
	st, err := s.GetReviewStats(ctx, item.ID, uuid.Nil, []string{"public"})
	if err != nil {
		t.Fatal(err)
	}
	if st.Count != 4 || st.Avg != 7.5 || st.NoteOnly != 2 {
		t.Errorf("stats visiteur : %+v", st)
	}
	if st.Criteria["a"].Count != 1 || st.Criteria["a"].Avg != 6 {
		t.Errorf("critères : %+v", st.Criteria)
	}
	if st, _ = s.GetReviewStats(ctx, item.ID, reader, []string{"members", "public"}); st.NoteOnly != 0 {
		t.Errorf("notes seules d'un membre, la sienne exclue : %+v", st)
	}

	// Un avis anonyme sans texte n'a rien à montrer.
	if _, err := s.PutReview(ctx, item.ID, writers["anonymous"], model.ReviewWrite{Rating: 8, Lang: "fr", Visibility: "anonymous"}); err != nil {
		t.Fatal(err)
	}
	if got := list(uuid.Nil, []string{"public"}); !slices.Equal(got, []string{"public"}) {
		t.Errorf("anonyme sans texte : %v", got)
	}

	// Un profil ne liste que le signé.
	for v, want := range map[string]int{"members": 0, "anonymous": 0, "private": 0, "public": 1} {
		reviews, _, err := s.ListUserReviews(ctx, writers[v], []string{"public"}, model.Pagination{Page: 1, PerPage: 20})
		if err != nil {
			t.Fatal(err)
		}
		if len(reviews) != want {
			t.Errorf("profil de %s : %d", v, len(reviews))
		}
		if len(reviews) > 0 && (reviews[0].Item == nil || reviews[0].Item.ItemType == nil) {
			t.Errorf("item absent : %+v", reviews[0])
		}
	}
	all, _, err := s.ListUserReviews(ctx, writers["private"], nil, model.Pagination{Page: 1, PerPage: 20})
	if err != nil || len(all) != 1 {
		t.Errorf("mes avis : %d %v", len(all), err)
	}

	// Un type d'avis utilisé ne se supprime pas ; fermer les avis libère le type d'item.
	if err := s.DeleteReviewType(ctx, rt.ID); !errors.Is(err, store.ErrConflict) {
		t.Errorf("type utilisé : %v", err)
	}
	if _, err := s.UpdateItemType(ctx, it.ID, model.ItemTypeUpdate{ReviewTypeID: model.OptionalUUID{Set: true}}); err != nil {
		t.Fatal(err)
	}
	if closed, _ := s.GetItemReviewType(ctx, item.ID); closed != nil {
		t.Errorf("avis fermés : %+v", closed)
	}
	if err := s.DeleteReview(ctx, item.ID, writers["public"]); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteReview(ctx, item.ID, writers["public"]); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("deuxième suppression : %v", err)
	}
	if err := s.DeleteReviewType(ctx, rt.ID); err != nil {
		t.Errorf("type libre : %v", err)
	}
}

// L'édition utilisée : seule une cible de l'œuvre par une relation déclarée est permise ; le filtre ne montre
// que les avis signés ; supprimer l'édition garde l'avis.
func TestReviews_Context(t *testing.T) {
	game := uuid.MustParse("b0000000-0000-0000-0000-000000000040")
	release := uuid.MustParse("b0000000-0000-0000-0000-000000000041")
	releasedAs := uuid.MustParse("c0000000-0000-0000-0000-000000000040")
	if _, err := s.InstallDomains(ctx, "games"); err != nil {
		t.Fatal(err)
	}
	gt, err := s.GetItemType(ctx, game)
	if err != nil || gt.ReviewTypeID == nil {
		t.Fatalf("Game : %+v %v", gt, err)
	}
	rt, err := s.GetReviewType(ctx, *gt.ReviewTypeID)
	if err != nil || len(rt.ContextRelationTypeIDs) != 1 || rt.ContextRelationTypeIDs[0] != releasedAs || rt.ContextLabel["fr"] != "Joué sur" {
		t.Fatalf("type d'avis du jeu : %+v %v", rt, err)
	}
	g, _ := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: game, Attributes: json.RawMessage(`{"title":"Jeu de test"}`)})
	linked, _ := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: release, Attributes: json.RawMessage(`{"title":"Édition liée","format":"cartridge"}`)})
	other, _ := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: release, Attributes: json.RawMessage(`{"title":"Autre édition","format":"cartridge"}`)})
	if g == nil || linked == nil || other == nil {
		t.Fatal("items de test")
	}
	if _, err := s.CreateRelation(ctx, model.ItemRelationCreate{RelationTypeID: releasedAs, SourceItemID: g.ID, TargetItemID: linked.ID, Attributes: json.RawMessage(`{}`)}); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListReviewContextItems(ctx, rt.ID, g.ID)
	if err != nil || len(items) != 1 || items[0].ID != linked.ID || items[0].ItemType == nil {
		t.Fatalf("éditions proposées : %+v %v", items, err)
	}
	if ok, _ := s.IsReviewContext(ctx, rt.ID, g.ID, linked.ID); !ok {
		t.Error("édition liée refusée")
	}
	if ok, _ := s.IsReviewContext(ctx, rt.ID, g.ID, other.ID); ok {
		t.Error("édition sans lien acceptée")
	}

	pub, _ := s.CreateUser(ctx, "ctx_public", "ctx_public@example.org", "x", model.RoleUser)
	anon, _ := s.CreateUser(ctx, "ctx_anon", "ctx_anon@example.org", "x", model.RoleUser)
	for _, w := range []struct {
		id  uuid.UUID
		vis string
	}{{pub.ID, "public"}, {anon.ID, "anonymous"}} {
		if _, err := s.PutReview(ctx, g.ID, w.id, model.ReviewWrite{Rating: 7, Body: "texte", Lang: "fr", Visibility: w.vis, ContextItemID: &linked.ID}); err != nil {
			t.Fatal(err)
		}
	}
	mine, _ := s.GetMyReview(ctx, g.ID, anon.ID)
	if mine.ContextItem == nil || mine.ContextItem.ItemType == nil || !strings.Contains(string(mine.ContextItem.Attributes), "Édition liée") {
		t.Errorf("édition jointe à l'avis : %+v", mine.ContextItem)
	}
	list, total, err := s.ListItemReviews(ctx, g.ID, model.ReviewListParams{Viewer: uuid.Nil, Levels: []string{"public"}, ContextItemID: &linked.ID}, model.Pagination{Page: 1, PerPage: 20})
	if err != nil || total != 1 || len(list) != 1 || list[0].Visibility != "public" {
		t.Errorf("filtre par édition : %d %+v %v", total, list, err)
	}

	if err := s.DeleteItem(ctx, linked.ID); err != nil {
		t.Fatal(err)
	}
	if mine, err = s.GetMyReview(ctx, g.ID, anon.ID); err != nil || mine.ContextItemID != nil {
		t.Errorf("édition supprimée : l'avis reste, sans précision : %+v %v", mine, err)
	}
}

// Les paquets ouvrent les avis sur les œuvres et les laissent fermés sur les personnes ; la note a
// quitté le type « Watching ».
func TestReviews_Packages(t *testing.T) {
	series, err := s.GetItemType(ctx, uuid.MustParse("b0000000-0000-0000-0000-000000000001"))
	if err != nil || series.ReviewTypeID == nil {
		t.Fatalf("Series : %+v %v", series, err)
	}
	person, err := s.GetItemType(ctx, uuid.MustParse("b0000000-0000-0000-0000-000000000016"))
	if err != nil || person.ReviewTypeID != nil {
		t.Fatalf("Person : %+v %v", person, err)
	}
	watching, err := s.GetCollectionType(ctx, uuid.MustParse("d0000000-0000-0000-0000-000000000002"))
	if err != nil || strings.Contains(string(watching.AttributesSchema), "rating") {
		t.Fatalf("Watching : %s %v", watching.AttributesSchema, err)
	}
}

// I-1 : un avis masqué sort des listes et des moyennes mais reste à son auteur ; le journal garde tout ;
// les réactions se comptent sans jamais modérer.
func TestModeration(t *testing.T) {
	it, _ := s.CreateItemType(ctx, model.ItemTypeCreate{Name: "TestModerated", AttributesSchema: json.RawMessage(`{"type":"object"}`)})
	rt, _ := s.CreateReviewType(ctx, model.ReviewTypeCreate{Name: "TestModReview"})
	if _, err := s.UpdateItemType(ctx, it.ID, model.ItemTypeUpdate{ReviewTypeID: model.OptionalUUID{Set: true, Value: &rt.ID}}); err != nil {
		t.Fatal(err)
	}
	item, _ := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: it.ID, Attributes: json.RawMessage(`{}`)})
	author, _ := s.CreateUser(ctx, "mod_author", "mod_author@example.org", "x", model.RoleUser)
	reporter, _ := s.CreateUser(ctx, "mod_reporter", "mod_reporter@example.org", "x", model.RoleUser)
	moderator, err := s.CreateUser(ctx, "mod_moderator", "mod_moderator@example.org", "x", model.RoleModerator)
	if err != nil {
		t.Fatalf("rôle moderator : %v", err)
	}
	rv, err := s.PutReview(ctx, item.ID, author.ID, model.ReviewWrite{Rating: 2, Body: "texte injurieux", Lang: "fr", Visibility: "public"})
	if err != nil {
		t.Fatal(err)
	}

	in := model.ReportCreate{TargetKind: "review", TargetID: rv.ID, Reason: "abusive"}
	if _, err := s.CreateReport(ctx, reporter.ID, in); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateReport(ctx, reporter.ID, in); !errors.Is(err, store.ErrConflict) {
		t.Errorf("deuxième signalement ouvert : %v", err)
	}
	queue, total, err := s.ListModerationQueue(ctx, model.Pagination{Page: 1, PerPage: 20})
	if err != nil || total != 1 || len(queue) != 1 || queue[0].Review == nil || len(queue[0].Reports) != 1 || queue[0].AuthorID != author.ID {
		t.Fatalf("file : %d %+v %v", total, queue, err)
	}

	if _, err := s.Moderate(ctx, "review", rv.ID, moderator.ID, model.ModerationHide, "Propos injurieux"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Moderate(ctx, "review", rv.ID, moderator.ID, model.ModerationHide, "encore"); !errors.Is(err, store.ErrConflict) {
		t.Errorf("masquer un avis masqué : %v", err)
	}
	if n, _ := s.CountModerationQueue(ctx); n != 0 {
		t.Errorf("signalement fermé par le masquage : %d", n)
	}
	list, _, _ := s.ListItemReviews(ctx, item.ID, model.ReviewListParams{Viewer: uuid.Nil, Levels: []string{"public"}}, model.Pagination{Page: 1, PerPage: 20})
	st, _ := s.GetReviewStats(ctx, item.ID, uuid.Nil, []string{"public"})
	if len(list) != 0 || st.Count != 0 {
		t.Errorf("avis masqué listé ou compté : %d %+v", len(list), st)
	}
	mine, _ := s.GetMyReview(ctx, item.ID, author.ID)
	if mine.HiddenAt == nil || mine.HiddenReason == nil || *mine.HiddenReason != "Propos injurieux" {
		t.Errorf("l'auteur voit son avis et le motif : %+v", mine)
	}
	if pub, _, _ := s.ListUserReviews(ctx, author.ID, []string{"public"}, model.Pagination{Page: 1, PerPage: 20}); len(pub) != 0 {
		t.Errorf("avis masqué sur le profil public : %d", len(pub))
	}
	if own, _, _ := s.ListUserReviews(ctx, author.ID, nil, model.Pagination{Page: 1, PerPage: 20}); len(own) != 1 {
		t.Errorf("mes avis, masqué compris : %d", len(own))
	}

	// Le journal garde la copie même si l'auteur supprime son avis.
	if err := s.DeleteReview(ctx, item.ID, author.ID); err != nil {
		t.Fatal(err)
	}
	hist, err := s.ModerationHistory(ctx, author.ID)
	if err != nil || hist.Recap.Hidden90d != 1 || len(hist.Entries) != 1 || hist.Entries[0].Excerpt != "texte injurieux" || hist.Entries[0].Moderator == nil {
		t.Errorf("historique : %+v %v", hist, err)
	}

	// Signalement classé sans suite : compté chez le signaleur.
	rv2, _ := s.PutReview(ctx, item.ID, author.ID, model.ReviewWrite{Rating: 9, Body: "avis correct", Lang: "fr", Visibility: "public"})
	_, _ = s.CreateReport(ctx, reporter.ID, model.ReportCreate{TargetKind: "review", TargetID: rv2.ID, Reason: "spam"})
	if _, err := s.Moderate(ctx, "review", rv2.ID, moderator.ID, model.ModerationDismiss, ""); err != nil {
		t.Fatal(err)
	}
	if recap, _ := s.ModerationRecap(ctx, reporter.ID); recap.Dismissed90d != 1 {
		t.Errorf("signalements classés du signaleur : %+v", recap)
	}
	if _, err := s.Moderate(ctx, "review", rv2.ID, moderator.ID, model.ModerationDismiss, ""); !errors.Is(err, store.ErrConflict) {
		t.Errorf("rien à classer : %v", err)
	}
	log, n, _ := s.ListModerationLog(ctx, model.Pagination{Page: 1, PerPage: 20})
	if n < 2 || log[0].Action != "dismiss" {
		t.Errorf("journal : %d %+v", n, log)
	}

	// Réactions : comptées, les miennes reconnues, tri « les plus intéressants ».
	for _, kind := range []string{"interesting", "funny"} {
		if err := s.SetReaction(ctx, rv2.ID, reporter.ID, kind, true); err != nil {
			t.Fatal(err)
		}
	}
	_ = s.SetReaction(ctx, rv2.ID, reporter.ID, "interesting", true) // sans effet
	_ = s.SetReaction(ctx, rv2.ID, moderator.ID, "interesting", true)
	_ = s.SetReaction(ctx, rv2.ID, reporter.ID, "funny", false)
	counts, mineR, err := s.ReactionsOf(ctx, []uuid.UUID{rv2.ID}, reporter.ID)
	if err != nil || counts[rv2.ID]["interesting"] != 2 || counts[rv2.ID]["funny"] != 0 || len(mineR[rv2.ID]) != 1 {
		t.Errorf("réactions : %+v %+v %v", counts, mineR, err)
	}
	if _, _, err := s.ListItemReviews(ctx, item.ID, model.ReviewListParams{Viewer: uuid.Nil, Levels: []string{"public"}, Sort: "interesting"}, model.Pagination{Page: 1, PerPage: 20}); err != nil {
		t.Errorf("tri interesting : %v", err)
	}
}

// I-2 : fil d'un avis, commentaires masqués, fermeture, suppression avec l'avis.
func TestComments(t *testing.T) {
	it, _ := s.CreateItemType(ctx, model.ItemTypeCreate{Name: "TestCommented", AttributesSchema: json.RawMessage(`{"type":"object"}`)})
	rt, _ := s.CreateReviewType(ctx, model.ReviewTypeCreate{Name: "TestCommentReview"})
	_, _ = s.UpdateItemType(ctx, it.ID, model.ItemTypeUpdate{ReviewTypeID: model.OptionalUUID{Set: true, Value: &rt.ID}})
	item, _ := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: it.ID, Attributes: json.RawMessage(`{}`)})
	author, _ := s.CreateUser(ctx, "com_author", "com_author@example.org", "x", model.RoleUser)
	other, _ := s.CreateUser(ctx, "com_other", "com_other@example.org", "x", model.RoleUser)
	mod, _ := s.CreateUser(ctx, "com_mod", "com_mod@example.org", "x", model.RoleModerator)
	rv, err := s.PutReview(ctx, item.ID, author.ID, model.ReviewWrite{Rating: 8, Body: "avis", Lang: "fr", Visibility: "public"})
	if err != nil {
		t.Fatal(err)
	}
	c1, err := s.CreateComment(ctx, "review", rv.ID, other.ID, "premier")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateComment(ctx, "review", rv.ID, author.ID, "réponse"); err != nil {
		t.Fatal(err)
	}
	list, total, _ := s.ListComments(ctx, "review", rv.ID, uuid.Nil, model.Pagination{Page: 1, PerPage: 50})
	if total != 2 || list[0].Body != "premier" || list[0].Author == nil || list[0].Author.Username != "com_other" {
		t.Errorf("fil : %d %+v", total, list)
	}
	if got, _ := s.GetReview(ctx, rv.ID); got.CommentCount != 2 {
		t.Errorf("compte des commentaires : %d", got.CommentCount)
	}
	if parts, _ := s.CommentParticipants(ctx, "review", rv.ID); len(parts) != 2 {
		t.Errorf("participants : %v", parts)
	}

	if _, err := s.UpdateComment(ctx, c1.ID, author.ID, "pas le mien"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("modifier le commentaire d'un autre : %v", err)
	}
	if c, err := s.UpdateComment(ctx, c1.ID, other.ID, "premier, corrigé"); err != nil || c.Body != "premier, corrigé" {
		t.Errorf("modifier le sien : %+v %v", c, err)
	}

	// Masqué : hors du fil et du compte, visible de son auteur seulement.
	if _, err := s.Moderate(ctx, "comment", c1.ID, mod.ID, model.ModerationHide, "Hors sujet"); err != nil {
		t.Fatal(err)
	}
	if list, _, _ := s.ListComments(ctx, "review", rv.ID, author.ID, model.Pagination{Page: 1, PerPage: 50}); len(list) != 1 {
		t.Errorf("commentaire masqué listé : %d", len(list))
	}
	if list, _, _ := s.ListComments(ctx, "review", rv.ID, other.ID, model.Pagination{Page: 1, PerPage: 50}); len(list) != 2 || list[0].HiddenAt == nil {
		t.Errorf("son auteur le voit avec le motif : %+v", list)
	}
	if got, _ := s.GetReview(ctx, rv.ID); got.CommentCount != 1 {
		t.Errorf("compte sans le masqué : %d", got.CommentCount)
	}
	if hist, _ := s.ModerationHistory(ctx, other.ID); hist.Recap.Hidden90d != 1 || hist.Entries[0].TargetKind != "comment" || !hist.Entries[0].StillHidden {
		t.Errorf("journal : %+v", hist)
	}

	// Fermeture par l'auteur seulement.
	if err := s.SetReviewCommentsClosed(ctx, rv.ID, other.ID, true); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("fermer le fil d'un autre : %v", err)
	}
	if err := s.SetReviewCommentsClosed(ctx, rv.ID, author.ID, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetReview(ctx, rv.ID); !got.CommentsClosed {
		t.Error("fil fermé")
	}

	// Le fil part avec l'avis.
	if err := s.DeleteReview(ctx, item.ID, author.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetComment(ctx, c1.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("commentaire orphelin : %v", err)
	}
}
