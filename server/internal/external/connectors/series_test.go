package connectors_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/connectors/connectorstest"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/validator"
)

// Réponses réelles enregistrées le 2026-10-17 (testdata/series_*, tvmaze_*, wikidata_*_series.json) :
// Breaking Bad. Dans l'enregistrement, la saison 5 (Q2585769) a perdu sa liste d'épisodes (P527)
// pour que ses épisodes viennent de TVmaze.
const (
	breakingBad   = "Q1079"
	bbSeason1     = "Q1582890"
	bbSeason5     = "Q2585769"
	bbPilot       = "Q14625947"
	bbS1DVD       = "ark:/12148/cb435115984" // « Intégrale de la saison 1 », 3 DVD
	bbFinalDVD    = "ark:/12148/cb43745269n" // « Saison finale [Saison 5] », EAN 3333297204842
	bbBroadcast   = "broadcast:wikidata:Q1079:Q294660"
	vinceGilligan = "Q4500"
)

func TestSeriesSearch(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.SeriesConnector(connectorstest.NewCatalog(t))
	if conn.ID() != "series" || conn.Domain() != "tv" || conn.EditionCode() != "barcode" || !conn.EditionsByWork() {
		t.Errorf("id %s, domain %s", conn.ID(), conn.Domain())
	}
	ctx := context.Background()
	hits, err := conn.SearchWorks(ctx, "Breaking Bad")
	// The other two hits of the search are not recorded entities: left out.
	if err != nil || len(hits) != 1 || hits[0].ID != breakingBad || hits[0].ItemTypeID != connectorstest.TypeID(t, "Series") ||
		hits[0].StartYear == nil || *hits[0].StartYear != 2008 {
		t.Fatalf("works: %v %+v", err, hits)
	}

	eds, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{Work: breakingBad})
	if err != nil {
		t.Fatal(err)
	}
	// 21 notices au titre « Breaking bad » : 17 coffrets (dont une notice sans type, reconnue à son support), sans le film El Camino ni « SOS.com ».
	titles := map[string]bool{}
	for _, h := range eds {
		titles[h.Title] = true
	}
	if len(eds) != 17 || !titles["Breaking bad — Intégrale de la saison 1"] || !titles["Breaking bad — Intégrale des saisons 1, 2 et 3"] {
		t.Errorf("box sets: %d %v", len(eds), titles)
	}
	byEAN, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{Barcode: "3333297204842"})
	if err != nil || len(byEAN) != 1 || byEAN[0].ID != bbFinalDVD {
		t.Errorf("by EAN: %v %+v", err, byEAN)
	}
	var input external.InputError
	if _, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{ISBN: "9782723433358"}); !errors.As(err, &input) {
		t.Errorf("ISBN: %v", err)
	}
}

func TestSeriesPreview(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.SeriesConnector(connectorstest.NewCatalog(t))
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Work: breakingBad, Editions: []string{bbS1DVD}})
	if err != nil {
		t.Fatal(err)
	}
	series := nodeByKey(t, plan, "wikidata:"+breakingBad)
	a := attrs(t, series.Attributes)
	if a["title"] != "Breaking Bad" || a["start_year"] != 2008.0 || a["end_year"] != 2013.0 || a["season_count"] != 5.0 ||
		a["episode_count"] != 62.0 || jsonPath(a, "external_ids", "imdb") != "tt0903747" || jsonPath(a, "external_ids", "tvmaze") != "169" {
		t.Errorf("series: %v", a)
	}
	if c, _ := a["countries"].([]any); !slices.Equal(c, []any{"US"}) {
		t.Errorf("countries: %v", a["countries"])
	}

	// Diffusion AMC (chaîne câblée), rattachée à la série ; les saisons en dépendent.
	b := nodeByKey(t, plan, bbBroadcast)
	if ba := attrs(t, b.Attributes); ba["name"] != "AMC 2008-2013" || ba["type"] != "cable" || relation(plan, series.Key, b.Key) == nil {
		t.Errorf("broadcast: %v", ba)
	}
	count := map[string]int{}
	sources := map[string]int{}
	for _, n := range plan.Nodes {
		count[n.Role]++
		if n.Role == model.NodeVolume {
			src, _ := jsonPath(attrs(t, n.Attributes), "sources").([]any)
			sources[src[0].(map[string]any)["source"].(string)]++
		}
	}
	if count[model.NodeSeason] != 5 || count[model.NodeVolume] != 62 || count[model.NodeEdition] != 1 {
		t.Errorf("nodes by role: %v", count)
	}
	// Saisons 1 à 4 de Wikidata (46 épisodes), saison 5 de TVmaze (16).
	if sources["wikidata"] != 46 || sources["tvmaze"] != 16 {
		t.Errorf("episode sources: %v", sources)
	}
	s1 := nodeByKey(t, plan, "wikidata:"+bbSeason1)
	if s1.Role != model.NodeSeason || attrs(t, s1.Attributes)["number"] != 1.0 || relation(plan, bbBroadcast, s1.Key) == nil {
		t.Errorf("season 1: %+v", s1)
	}
	pilot := nodeByKey(t, plan, "wikidata:"+bbPilot)
	pa := attrs(t, pilot.Attributes)
	if pilot.Parent != s1.Key || pa["title"] != "Chute libre" || pa["original_title"] != "Pilot" || pa["number"] != 1.0 ||
		pa["air_date"] != "2008-01-20" || pa["duration"] != 58.0 || pa["synopsis"] != nil {
		t.Errorf("pilot: %v", pa)
	}
	if r := relation(plan, s1.Key, pilot.Key); r == nil || jsonValue(t, r.Attributes, "order") != 1.0 {
		t.Errorf("contains_episode: %+v", r)
	}
	s5 := nodeByKey(t, plan, "wikidata:"+bbSeason5)
	if eps := 0; true {
		for _, n := range plan.Nodes {
			if n.Parent == s5.Key && n.Role == model.NodeVolume {
				eps++
			}
		}
		if eps != 16 {
			t.Errorf("season 5 episodes (TVmaze): %d", eps)
		}
	}

	// Coffret de la saison 1 : ses 7 épisodes, par includes_episode.
	box := nodeByKey(t, plan, "bnf:cb435115984")
	included := 0
	for _, r := range plan.Relations {
		if r.From == box.Key && r.RelationTypeID == connectorstest.TypeID(t, "includes_episode") {
			included++
		}
	}
	if ba := attrs(t, box.Attributes); included != 7 || ba["format"] != "dvd" || ba["disc_count"] != 3.0 ||
		ba["title"] != "Breaking bad — Intégrale de la saison 1" || relation(plan, series.Key, box.Key) == nil {
		t.Errorf("box set: %d episodes, %v", included, ba)
	}
	if r := relation(plan, series.Key, "wikidata:"+vinceGilligan); r == nil || !slices.Contains(jsonValue(t, r.Attributes, "roles").([]any), any("creator")) {
		t.Errorf("creator: %+v", r)
	}

	its, rts := connectorstest.Domain(t)
	for _, n := range plan.Nodes {
		i := slices.IndexFunc(its, func(it model.ItemType) bool { return it.ID == n.ItemTypeID })
		if err := validator.ValidateAttributes(its[i].AttributesSchema, n.Attributes); err != nil {
			t.Errorf("node %s invalid: %v", n.Key, err)
		}
	}
	for _, r := range plan.Relations {
		i := slices.IndexFunc(rts, func(rt model.RelationType) bool { return rt.ID == r.RelationTypeID })
		if err := validator.ValidateAttributes(rts[i].AttributesSchema, r.Attributes); err != nil {
			t.Errorf("relation %s → %s invalid: %v", r.From, r.To, err)
		}
	}
}

// A box set alone (barcode) finds its series by title; « Saison finale » is the last season.
func TestSeriesBoxSetAlone(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.SeriesConnector(connectorstest.NewCatalog(t))
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Editions: []string{bbFinalDVD}})
	if err != nil {
		t.Fatal(err)
	}
	box := nodeByKey(t, plan, "bnf:cb43745269n")
	if box.Parent != "wikidata:"+breakingBad {
		t.Fatalf("box set parent: %q", box.Parent)
	}
	included := 0
	for _, r := range plan.Relations {
		if r.From == box.Key && r.RelationTypeID == connectorstest.TypeID(t, "includes_episode") {
			included++
		}
	}
	if included != 16 {
		t.Errorf("final season episodes: %d", included)
	}
}

// The broadcast of an existing series is found through broadcast_on (same channel).
func TestSeriesBroadcastMatchedThroughRelation(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	seriesID, broadcastID := uuid.New(), uuid.New()
	cat.Items = []model.Item{
		{ID: seriesID, ItemTypeID: connectorstest.TypeID(t, "Series"), Attributes: []byte(`{"title":"Breaking Bad","external_ids":{"wikidata":"Q1079"}}`)},
		{ID: broadcastID, ItemTypeID: connectorstest.TypeID(t, "Broadcast"), Attributes: []byte(`{"name":"AMC 2008-2013","channel":"amc","type":"cable"}`)},
	}
	cat.Relations = []model.ItemRelation{{ID: uuid.New(), RelationTypeID: connectorstest.TypeID(t, "broadcast_on"), SourceItemID: seriesID, TargetItemID: broadcastID}}
	conn := src.SeriesConnector(cat)
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Work: breakingBad})
	if err != nil {
		t.Fatal(err)
	}
	b := nodeByKey(t, plan, bbBroadcast)
	if b.ExistingItemID == nil || *b.ExistingItemID != broadcastID {
		t.Errorf("broadcast: %+v", b)
	}
	if r := relation(plan, "wikidata:"+breakingBad, bbBroadcast); r == nil || r.ExistingID == nil {
		t.Errorf("broadcast_on reused: %+v", r)
	}
}
