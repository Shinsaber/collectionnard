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

// Réponses réelles enregistrées le 2026-10-17 (testdata/film_*, wikidata_*_film*.json) :
// Le Fabuleux Destin d'Amélie Poulain, notice d'œuvre BnF 14404106.
const (
	amelie     = "Q484048"
	ameliePDVD = "ark:/12148/cb42284376s" // DVD TF1 vidéo 2010, EAN 3384442032803
	amelieBR   = "ark:/12148/cb42284132m" // Blu-ray « édition collector » 2010
	jeunet     = "Q217324"
	tautou     = "Q304123"
)

func TestFilmsSearchWorks(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.FilmsConnector(connectorstest.NewCatalog(t))
	if conn.ID() != "films" || conn.Domain() != "video" || conn.EditionCode() != "barcode" || !conn.EditionsByWork() {
		t.Errorf("id %s, domain %s", conn.ID(), conn.Domain())
	}
	hits, err := conn.SearchWorks(context.Background(), "Amélie Poulain")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != amelie || hits[0].ItemTypeID != connectorstest.TypeID(t, "Film") ||
		hits[0].StartYear == nil || *hits[0].StartYear != 2001 {
		t.Fatalf("hits: %+v", hits)
	}
}

func TestFilmsSearchEditions(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.FilmsConnector(connectorstest.NewCatalog(t))
	ctx := context.Background()
	hits, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{Work: amelie})
	if err != nil {
		t.Fatal(err)
	}
	// 9 notices vidéo : DVD, Blu-ray et VHS, les plus récentes d'abord.
	if len(hits) != 9 || hits[0].StartYear == nil || *hits[0].StartYear != 2010 {
		t.Fatalf("hits: %+v", hits)
	}
	formats := map[string]int{}
	for _, h := range hits {
		formats[h.Format]++
	}
	if formats["DVD · zone 2"] != 2 || formats["2 DVD · zone 2"] != 2 || formats["Blu-ray"] != 1 || formats["VHS"] != 4 {
		t.Errorf("formats: %v", formats)
	}
	i := slices.IndexFunc(hits, func(h model.ExternalEditionHit) bool { return h.ID == amelieBR })
	if i < 0 || hits[i].Barcode != "3384442196406" || !slices.Equal(hits[i].Publishers, []string{"TF1 vidéo"}) ||
		hits[i].Title != "Le fabuleux destin d'Amélie Poulain : édition collector" {
		t.Errorf("Blu-ray: %+v", hits)
	}

	hits, err = conn.SearchEditions(ctx, model.ExternalEditionQuery{Barcode: "3384442032803"})
	if err != nil || len(hits) != 1 || hits[0].ID != ameliePDVD {
		t.Errorf("by EAN: %v %+v", err, hits)
	}
	for _, q := range []model.ExternalEditionQuery{{ISBN: "9782723433358"}, {Work: connectorstest.OKComputer}} {
		var input external.InputError
		if _, err := conn.SearchEditions(ctx, q); !errors.As(err, &input) {
			t.Errorf("%+v: %v, want InputError", q, err)
		}
	}
}

func TestFilmsPreview(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.FilmsConnector(connectorstest.NewCatalog(t))
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Work: amelie, Editions: []string{amelieBR}})
	if err != nil {
		t.Fatal(err)
	}
	film := nodeByKey(t, plan, "wikidata:"+amelie)
	a := attrs(t, film.Attributes)
	if film.Role != model.NodeWork || jsonPath(a, "title", "fr") != "Le Fabuleux Destin d'Amélie Poulain" ||
		a["release_year"] != 2001.0 || a["duration"] != 122.0 || a["kind"] != "feature" || a["original_language"] != "fr" ||
		jsonPath(a, "external_ids", "imdb") != "tt0211915" || jsonPath(a, "external_ids", "bnf_ark") != "ark:/12148/cb14404106q" {
		t.Errorf("film: %v", a)
	}
	if c, _ := a["countries"].([]any); !slices.Equal(c, []any{"FR", "DE"}) {
		t.Errorf("countries: %v", a["countries"])
	}
	if g, _ := a["genres"].([]any); !slices.Contains(g, any("comedy")) {
		t.Errorf("genres: %v", a["genres"])
	}

	// Jeunet : réalisateur et scénariste, une seule relation ; Tautou en tête d'affiche.
	r := relation(plan, film.Key, "wikidata:"+jeunet)
	if r == nil || !slices.Equal(jsonValue(t, r.Attributes, "roles").([]any), []any{"director", "screenwriter"}) {
		t.Errorf("Jeunet: %+v", r)
	}
	r = relation(plan, film.Key, "wikidata:"+tautou)
	if r == nil || jsonValue(t, r.Attributes, "order") != 1.0 || attrs(t, nodeByKey(t, plan, "wikidata:"+tautou).Attributes)["last_name"] != "Tautou" {
		t.Errorf("Tautou: %+v", r)
	}
	actors, selected := 0, 0
	for _, n := range plan.Nodes {
		if rel := relation(plan, film.Key, n.Key); rel != nil && n.Role == model.NodePerson &&
			slices.Contains(jsonValue(t, rel.Attributes, "roles").([]any), any("actor")) {
			actors++
			if n.Selected {
				selected++
			}
		}
	}
	if actors != 10 || selected != 5 {
		t.Errorf("cast: %d actors, %d selected", actors, selected)
	}

	release := nodeByKey(t, plan, "bnf:cb42284132m")
	a = attrs(t, release.Attributes)
	if release.Role != model.NodeEdition || release.Parent != film.Key || a["format"] != "blu_ray" || a["ean"] != "3384442196406" ||
		a["edition"] != "édition collector" || a["release_year"] != 2010.0 || a["publisher"] != "TF1 vidéo" || a["runtime"] != 120.0 {
		t.Errorf("release: %v", a)
	}
	if relation(plan, film.Key, release.Key) == nil {
		t.Error("released_as missing")
	}
	pub := nodeByKey(t, plan, "publisher:tf1 vidéo")
	if r := relation(plan, release.Key, pub.Key); r == nil {
		t.Error("published_by missing")
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

// A release chosen alone (barcode) brings its film: the Wikidata film of its
// BnF work record (P268).
func TestFilmsPreviewReleaseAlone(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.FilmsConnector(connectorstest.NewCatalog(t))
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Editions: []string{ameliePDVD}})
	if err != nil {
		t.Fatal(err)
	}
	release := nodeByKey(t, plan, "bnf:cb42284376s")
	a := attrs(t, release.Attributes)
	if release.Parent != "wikidata:"+amelie || a["format"] != "dvd" || a["region"] != "2" || a["disc_count"] != 1.0 ||
		a["aspect_ratio"] != "16/9" || a["ean"] != "3384442032803" {
		t.Errorf("DVD: %+v %v", release, a)
	}
	if relation(plan, "wikidata:"+amelie, release.Key) == nil {
		t.Error("released_as of the film found by P268 missing")
	}
}

func TestFilmsMatchingAndInvalid(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	filmID := uuid.New()
	cat.Items = []model.Item{{ID: filmID, ItemTypeID: connectorstest.TypeID(t, "Film"),
		Attributes: []byte(`{"title":{"fr":"Amélie"},"external_ids":{"bnf_ark":"ark:/12148/cb14404106q"}}`)}}
	conn := src.FilmsConnector(cat)
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Work: amelie})
	if err != nil {
		t.Fatal(err)
	}
	film := nodeByKey(t, plan, "wikidata:"+amelie)
	if film.ExistingItemID == nil || *film.ExistingItemID != filmID || jsonPath(attrs(t, film.Attributes), "title", "fr") != "Amélie" {
		t.Errorf("film matched by its BnF work, keeping its title: %+v", film)
	}
	for _, req := range []model.ExternalImportRequest{{Work: connectorstest.OKComputer}, {Editions: []string{"nope"}}} {
		var input external.InputError
		if _, err := conn.Preview(context.Background(), req); !errors.As(err, &input) {
			t.Errorf("%+v: %v, want InputError", req, err)
		}
	}
}
