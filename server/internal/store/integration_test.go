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
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/dbmigrate"
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
)

func TestMain(m *testing.M) {
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = "postgres://colec:colec@localhost:5432/colec?sslmode=disable"
	}
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
		if err := dbmigrate.Seed(ctx, testURL, seedFile); err != nil {
			fmt.Fprintln(os.Stderr, "seed:", err)
			return 1
		}
		s, err = store.NewPgStore(ctx, testURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "store:", err)
			return 1
		}
		defer s.Close()
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

	// Wildcards are literal: no attribute contains a percent sign.
	_, total, err = s.ListItems(ctx, pg, store.ItemFilters{Search: "%"})
	if err != nil || total != 0 {
		t.Errorf("search %%: total %d (wildcard not escaped?), err %v", total, err)
	}
	// Unescaped, "Breaking%Bad" would match "Breaking Bad" through the % wildcard.
	_, total, err = s.ListItems(ctx, pg, store.ItemFilters{Search: "Breaking%Bad"})
	if err != nil || total != 0 {
		t.Errorf("search Breaking%%Bad: total %d (wildcard not escaped?), err %v", total, err)
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
