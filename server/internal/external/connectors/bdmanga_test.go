package connectors_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/connectors"
	"github.com/shinsaber/colec/internal/external/connectors/connectorstest"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/validator"
)

const (
	dragonBall = "Q15051307"
	glenat1993 = "ark:/12148/cb343384394"
)

var dragonBallReq = model.ExternalImportRequest{Work: dragonBall, Editions: []string{glenat1993}}

func attrs(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func nodeByKey(t *testing.T, p *model.ExternalPlan, key string) model.ExternalNode {
	t.Helper()
	for _, n := range p.Nodes {
		if n.Key == key {
			return n
		}
	}
	t.Fatalf("no node %q", key)
	return model.ExternalNode{}
}

func relation(p *model.ExternalPlan, from, to string) *model.ExternalRelation {
	for i := range p.Relations {
		if p.Relations[i].From == from && p.Relations[i].To == to {
			return &p.Relations[i]
		}
	}
	return nil
}

func TestPreview_NewCatalogue(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	plan, err := src.Connector(cat).Preview(context.Background(), dragonBallReq)
	if err != nil {
		t.Fatal(err)
	}
	if plan.RetrievedAt != "2026-10-11" {
		t.Errorf("retrieved_at = %q", plan.RetrievedAt)
	}

	work := nodeByKey(t, plan, "wikidata:"+dragonBall)
	if work.Role != model.NodeWork || work.Status != model.NodeNew || !work.Selected ||
		work.ItemTypeID != connectorstest.TypeID(t, "Manga") {
		t.Errorf("work node = %+v", work)
	}
	w := attrs(t, work.Attributes)
	for k, want := range map[string]any{
		"start_year": 1984.0, "end_year": 1995.0, "status": "completed", "volumes_vo": 42.0,
		"demographic": "shonen", "sub_type": "manga", "original_title": "ドラゴンボール",
	} {
		if w[k] != want {
			t.Errorf("work %s = %v, want %v", k, w[k], want)
		}
	}
	if w["title"].(map[string]any)["fr"] != "Dragon Ball" {
		t.Errorf("work title = %v", w["title"])
	}
	if src := w["sources"].([]any)[0].(map[string]any); src["source"] != "wikidata" || src["retrieved_at"] != "2026-10-11" {
		t.Errorf("work sources = %v", w["sources"])
	}

	toriyama := attrs(t, nodeByKey(t, plan, "wikidata:Q208582").Attributes)
	for k, want := range map[string]any{
		"last_name": "Toriyama", "first_name": "Akira", "nationality": "JP", "birth_date": "1955-04-05",
	} {
		if toriyama[k] != want {
			t.Errorf("person %s = %v, want %v", k, toriyama[k], want)
		}
	}
	if ids := toriyama["external_ids"].(map[string]any); ids["bnf_ark"] != "ark:/12148/cb12336978f" {
		t.Errorf("person external_ids = %v (P268 → ARK with its check character)", ids)
	}
	created := relation(plan, "wikidata:"+dragonBall, "wikidata:Q208582")
	if created == nil || string(created.Attributes) != `{"roles":["author","artist"]}` {
		t.Errorf("created_by = %+v", created)
	}
	if relation(plan, "wikidata:Q2020", "wikidata:"+dragonBall) == nil {
		t.Error("missing contains_work universe → work")
	}

	edition := nodeByKey(t, plan, "bnf:cb343384394")
	e := attrs(t, edition.Attributes)
	if e["language"] != "fr" || e["start_year"] != 1993.0 || edition.Parent != "wikidata:"+dragonBall {
		t.Errorf("edition = %v (parent %q)", e, edition.Parent)
	}
	if relation(plan, "wikidata:"+dragonBall, "bnf:cb343384394") == nil ||
		relation(plan, "bnf:cb343384394", "publisher:glénat") == nil {
		t.Error("missing published_as / edited_by")
	}

	v1 := nodeByKey(t, plan, "bnf:cb356004006")
	a1 := attrs(t, v1.Attributes)
	if a1["number"] != 1.0 || a1["isbn"] != "9782876952058" || a1["page_count"] != 182.0 ||
		a1["title"].(map[string]any)["fr"] != "Sangoku" || a1["retail_price"] != nil {
		t.Errorf("volume 1 = %v (price in francs must be ignored)", a1)
	}
	if r := relation(plan, "bnf:cb343384394", "bnf:cb356004006"); r == nil || string(r.Attributes) != `{"order":1}` {
		t.Errorf("contains_volume = %+v", r)
	}
	// Two records numbered 8: the oldest is selected, the other is a reprint.
	var eights []model.ExternalNode
	for _, n := range plan.Nodes {
		if n.Role == model.NodeVolume && attrs(t, n.Attributes)["number"] == 8.0 {
			eights = append(eights, n)
		}
	}
	if len(eights) != 2 || !eights[0].Selected || eights[1].Selected || eights[1].Note != model.NoteReprint {
		t.Errorf("volumes 8 = %+v", eights)
	}

	// Every proposed item is valid against the schema of its type.
	its, _ := connectorstest.Domain(t)
	for _, n := range plan.Nodes {
		i := slices.IndexFunc(its, func(it model.ItemType) bool { return it.ID == n.ItemTypeID })
		if err := validator.ValidateAttributes(its[i].AttributesSchema, n.Attributes); err != nil {
			t.Errorf("node %s invalid: %v", n.Key, err)
		}
	}
}

func TestPreview_ExistingItemsKeepTheirValues(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	workID, personID, relID := uuid.New(), uuid.New(), uuid.New()
	cat.Items = []model.Item{
		{ID: workID, ItemTypeID: connectorstest.TypeID(t, "Manga"), Tags: []model.Tag{{Name: "culte"}},
			Attributes: json.RawMessage(`{"title":{"fr":"Dragon Ball (titre local)"},"status":"ongoing","external_ids":{"wikidata":"Q15051307"},"sources":[{"source":"manual"}]}`)},
		{ID: personID, ItemTypeID: connectorstest.TypeID(t, "Person"),
			Attributes: json.RawMessage(`{"last_name":"Toriyama","external_ids":{"wikidata":"Q208582"}}`)},
	}
	cat.Relations = []model.ItemRelation{{ID: relID, RelationTypeID: connectorstest.TypeID(t, "created_by"),
		SourceItemID: workID, TargetItemID: personID, Attributes: json.RawMessage(`{"roles":["author"]}`)}}

	plan, err := src.Connector(cat).Preview(context.Background(), model.ExternalImportRequest{Work: dragonBall})
	if err != nil {
		t.Fatal(err)
	}
	work := nodeByKey(t, plan, "wikidata:"+dragonBall)
	if work.Status != model.NodeUpdate || work.ExistingItemID == nil || *work.ExistingItemID != workID {
		t.Fatalf("work node = %+v", work)
	}
	w := attrs(t, work.Attributes)
	title := w["title"].(map[string]any)
	if title["fr"] != "Dragon Ball (titre local)" || title["en"] != "Dragon Ball" {
		t.Errorf("title = %v: the local value is kept, the missing language added", title)
	}
	if w["status"] != "ongoing" || w["start_year"] != 1984.0 {
		t.Errorf("status / start_year = %v / %v", w["status"], w["start_year"])
	}
	if s := w["sources"].([]any); len(s) != 2 || s[0].(map[string]any)["source"] != "manual" {
		t.Errorf("sources = %v", s)
	}
	r := relation(plan, "wikidata:"+dragonBall, "wikidata:Q208582")
	if r == nil || r.ExistingID == nil || *r.ExistingID != relID || string(r.Attributes) != `{"roles":["author"]}` {
		t.Errorf("created_by = %+v: existing relation reused, existing roles kept", r)
	}

	// The import keeps the tags of the existing item and reuses the ids.
	keys := []string{"wikidata:" + dragonBall, "wikidata:Q208582"}
	file, ids, err := src.Connector(cat).Build(context.Background(), model.ExternalImportRequest{Work: dragonBall, Selected: keys})
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Items) != 2 || file.Items[0].ID != workID || !slices.Equal(file.Items[0].Tags, []string{"culte"}) {
		t.Errorf("items = %+v", file.Items)
	}
	if len(file.Relations) != 1 || file.Relations[0].ID != relID {
		t.Errorf("relations = %+v", file.Relations)
	}
	if ids["wikidata:"+dragonBall] != workID {
		t.Errorf("item_ids = %v", ids)
	}
}

func TestBuild_Selection(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	conn := src.Connector(cat)
	ctx := context.Background()

	// Work and edition only: the relation to the unselected new person is dropped.
	req := dragonBallReq
	req.Selected = []string{"wikidata:" + dragonBall, "bnf:cb343384394", "bnf:cb356004006"}
	file, ids, err := conn.Build(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if file.Format != model.CatalogFormat || len(file.Items) != 3 || len(ids) != 3 {
		t.Fatalf("file = %+v, ids = %v", file, ids)
	}
	if len(file.Relations) != 2 { // published_as + contains_volume
		t.Errorf("relations = %+v", file.Relations)
	}
	for _, r := range file.Relations {
		if r.SourceItemID == uuid.Nil || r.TargetItemID == uuid.Nil || r.ID == uuid.Nil {
			t.Errorf("relation with nil id: %+v", r)
		}
	}

	// Corrected volume: its attributes replace the proposed ones, the order in
	// the edition follows the corrected number.
	req.Overrides = map[string]json.RawMessage{
		"bnf:cb356004006": json.RawMessage(`{"number":5,"title":{"fr":"Sangoku (corrigé)"}}`),
	}
	file, _, err = conn.Build(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	var fixedItem, fixedRel bool
	for _, it := range file.Items {
		fixedItem = fixedItem || string(it.Attributes) == `{"number":5,"title":{"fr":"Sangoku (corrigé)"}}`
	}
	for _, r := range file.Relations {
		fixedRel = fixedRel || string(r.Attributes) == `{"order":5}`
	}
	if !fixedItem || !fixedRel {
		t.Errorf("override not applied: items %+v relations %+v", file.Items, file.Relations)
	}
	for name, o := range map[string]map[string]json.RawMessage{
		"unselected node": {"wikidata:Q208582": json.RawMessage(`{"last_name":"X"}`)},
		"not an object":   {"bnf:cb356004006": json.RawMessage(`[1]`)},
	} {
		req.Overrides = o
		if _, _, err := conn.Build(ctx, req); !errors.As(err, new(connectors.InputError)) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	req.Overrides = nil

	req.Selected = []string{"wikidata:Q1"}
	if _, _, err := conn.Build(ctx, req); !errors.As(err, new(connectors.InputError)) {
		t.Errorf("unknown key: err = %v", err)
	}
	req.Selected = nil
	if _, _, err := conn.Build(ctx, req); !errors.As(err, new(connectors.InputError)) {
		t.Errorf("empty selection: err = %v", err)
	}
}

func TestPreview_InvalidRequests(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.Connector(connectorstest.NewCatalog(t))
	for name, req := range map[string]model.ExternalImportRequest{
		"empty":        {},
		"bad qid":      {Work: "42"},
		"bad ark":      {Editions: []string{"ark:/12148/cb34338439x"}}, // wrong check character
		"not a record": {Editions: []string{"https://example.org"}},
	} {
		if _, err := conn.Preview(context.Background(), req); !errors.As(err, new(connectors.InputError)) {
			t.Errorf("%s: err = %v, want InputError", name, err)
		}
	}
	if _, err := conn.Preview(context.Background(), model.ExternalImportRequest{Work: "Q999"}); !connectors.IsNotFound(err) {
		t.Errorf("unknown work: err = %v, want not found", err)
	}
}

func TestDomainMissing(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	cat.ItemTypes = slices.DeleteFunc(cat.ItemTypes, func(it model.ItemType) bool { return it.Name == "Volume" })
	_, err := src.Connector(cat).SearchWorks(context.Background(), "dragon ball")
	var unavailable *connectors.UnavailableError
	if !errors.As(err, &unavailable) || !slices.Equal(unavailable.Missing, []string{"Volume"}) {
		t.Errorf("err = %v", err)
	}
}

// Types renamed on the instance (Comic → bd, created_by → par) are still found by their package id.
func TestDomainRenamedTypes(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	for i := range cat.ItemTypes {
		if cat.ItemTypes[i].Name == "Comic" {
			cat.ItemTypes[i].Name = "bd"
		}
	}
	for i := range cat.RelationTypes {
		if cat.RelationTypes[i].Name == "created_by" {
			cat.RelationTypes[i].Name = "par"
		}
	}
	if _, err := src.Connector(cat).SearchWorks(context.Background(), "dragon ball"); err != nil {
		t.Errorf("renamed types: %v", err)
	}
}

func TestSearchWorks(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	existing := uuid.New()
	cat.Items = []model.Item{{ID: existing, ItemTypeID: connectorstest.TypeID(t, "Manga"),
		Attributes: json.RawMessage(`{"title":{"fr":"DB"},"external_ids":{"wikidata":"Q64230248"}}`)}}
	hits, err := src.Connector(cat).SearchWorks(context.Background(), "dragon ball")
	if err != nil {
		t.Fatal(err)
	}
	// Q3715032 is absent from the recorded entities: skipped.
	if len(hits) != 2 || hits[0].WikidataID != dragonBall || hits[0].Label != "Dragon Ball" ||
		hits[0].StartYear == nil || *hits[0].StartYear != 1984 || hits[0].ExistingItemID != nil {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[1].ExistingItemID == nil || *hits[1].ExistingItemID != existing {
		t.Errorf("Dragon Ball Super should match the existing item: %+v", hits[1])
	}
}

func TestSearchEditions(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.Connector(connectorstest.NewCatalog(t))
	ctx := context.Background()

	byTitle, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{Title: "dragon ball"})
	if err != nil {
		t.Fatal(err)
	}
	var arks []string
	for _, h := range byTitle {
		arks = append(arks, h.BnfArk)
	}
	// cb39272675f is a multimedia set (leader 6 = m): excluded.
	if slices.Contains(arks, "ark:/12148/cb39272675f") || !slices.Contains(arks, glenat1993) {
		t.Errorf("editions by title = %v", arks)
	}

	byISBN, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{ISBN: "9782876952058"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byISBN) != 2 {
		t.Fatalf("editions by ISBN = %+v", byISBN)
	}
	for _, h := range byISBN {
		if h.MatchedVolume == nil || h.MatchedVolume.Number == nil || *h.MatchedVolume.Number != 1 {
			t.Errorf("matched volume of %s = %+v", h.BnfArk, h.MatchedVolume)
		}
	}
}

func TestSourcesDownAndCache(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.Connector(connectorstest.NewCatalog(t))
	ctx := context.Background()
	if _, err := conn.Preview(ctx, dragonBallReq); err != nil {
		t.Fatal(err)
	}
	first := src.Requests.Load()
	// Same selection again (preview, then import): served from the cache.
	if _, err := conn.Preview(ctx, dragonBallReq); err != nil {
		t.Fatal(err)
	}
	if n := src.Requests.Load(); n != first {
		t.Errorf("second preview made %d requests, want 0", n-first)
	}

	src.Down.Store(true)
	if _, err := conn.SearchWorks(ctx, "naruto"); !errors.Is(err, external.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
}
