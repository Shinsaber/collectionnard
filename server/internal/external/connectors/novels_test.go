package connectors_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/shinsaber/colec/internal/external/connectors/connectorstest"
	"github.com/shinsaber/colec/internal/model"
)

// Réponses réelles enregistrées le 2026-10-17 (testdata/novel_*, wikidata_*_novels.json).
const (
	etranger    = "Q163297"
	miserables  = "Q180736"
	folio2013   = "ark:/12148/cb436623939"
	outsider    = "ark:/12148/cb35215528k" // traduction anglaise : traducteur et préfacier
	pocketSet   = "ark:/12148/cb45767941c" // Les Misérables, Pocket 2019, 3 volumes
	ocVolume    = "ark:/12148/cb330461815" // volume des « Œuvres complètes » de Victor Hugo
	cycleAbsurd = "Q3008298"
)

func TestNovelsSearchWorks(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.NovelsConnector(connectorstest.NewCatalog(t))
	if conn.ID() != "novels" || conn.Domain() != "books" {
		t.Errorf("id %s, domain %s", conn.ID(), conn.Domain())
	}
	hits, err := conn.SearchWorks(context.Background(), "L'Étranger Camus")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].WikidataID != etranger || hits[0].ItemTypeID != connectorstest.TypeID(t, "Novel") {
		t.Fatalf("hits: %+v", hits)
	}
	if hits[0].StartYear == nil || *hits[0].StartYear != 1942 {
		t.Errorf("start year (publication): %v", hits[0].StartYear)
	}
}

func TestNovelsSearchEditionsByWork(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.NovelsConnector(connectorstest.NewCatalog(t))
	hits, err := conn.SearchEditions(context.Background(), model.ExternalEditionQuery{Work: etranger})
	if err != nil {
		t.Fatal(err)
	}
	// 5 notices : 3 éditions, la compilation allemande (titre en 423$t), une notice quasi vide écartée.
	if len(hits) != 4 {
		t.Fatalf("hits: %+v", hits)
	}
	// Éditions françaises d'abord ; les traductions gardent leur langue (pas de « fr » par défaut).
	first := hits[0]
	if first.BnfArk != folio2013 || first.Language != "fr" || first.ISBN != "9782070453177" ||
		first.Collection != "Collection Folio" || !slices.Equal(first.Publishers, []string{"Gallimard"}) {
		t.Errorf("Folio: %+v", first)
	}
	langs := []string{}
	titles := map[string]string{}
	for _, h := range hits[1:] {
		langs = append(langs, h.Language)
		titles[h.Language] = h.Title
	}
	slices.Sort(langs)
	if !slices.Equal(langs, []string{"de", "en", "hat"}) || titles["de"] != "Der Fremde" {
		t.Errorf("translations: %v %v", langs, titles)
	}
}

func TestNovelsSearchEditionsSets(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.NovelsConnector(connectorstest.NewCatalog(t))
	hits, err := conn.SearchEditions(context.Background(), model.ExternalEditionQuery{Work: miserables})
	if err != nil {
		t.Fatal(err)
	}
	arks := map[string]model.ExternalEditionHit{}
	for _, h := range hits {
		arks[h.BnfArk] = h
	}
	// Le volume Pocket représente son ensemble (titre identique) ; celui des Œuvres complètes reste une édition.
	if _, ok := arks[pocketSet]; !ok {
		t.Errorf("Pocket set missing: %+v", hits)
	}
	if _, ok := arks[ocVolume]; !ok || len(hits) != 2 {
		t.Errorf("collected works volume must stay a single edition: %+v", hits)
	}
}

func TestNovelsPreview(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	conn := src.NovelsConnector(cat)
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{
		Work: etranger, Editions: []string{folio2013, outsider},
	})
	if err != nil {
		t.Fatal(err)
	}
	nodes := map[string]model.ExternalNode{}
	for _, n := range plan.Nodes {
		nodes[n.Key] = n
	}
	work := nodes["wikidata:"+etranger]
	if work.ItemTypeID != connectorstest.TypeID(t, "Novel") {
		t.Fatalf("work: %+v", work)
	}
	for k, want := range map[string]any{"form": "novel", "original_language": "fr", "first_publication_year": float64(1942)} {
		if got := jsonValue(t, work.Attributes, k); got != want {
			t.Errorf("work %s = %v, want %v", k, got, want)
		}
	}
	if s := nodes["wikidata:"+cycleAbsurd]; s.ItemTypeID != connectorstest.TypeID(t, "Saga") {
		t.Errorf("cycle → Saga: %+v", s)
	}
	folio := nodes["bnf:cb436623939"]
	for k, want := range map[string]any{"isbn": "9782070453177", "collection": "Collection Folio", "language": "fr", "retail_price": 8.7} {
		if got := jsonValue(t, folio.Attributes, k); got != want {
			t.Errorf("Folio %s = %v, want %v", k, got, want)
		}
	}
	if jsonValue(t, nodes["bnf:cb35215528k"].Attributes, "language") != "en" {
		t.Errorf("translation language: %s", nodes["bnf:cb35215528k"].Attributes)
	}

	// Relations : cycle (ordre 1), auteur, traducteur et préfacier de la traduction.
	createdBy := connectorstest.TypeID(t, "created_by")
	containsWork := connectorstest.TypeID(t, "contains_work")
	roles := map[string]string{}
	order := ""
	for _, r := range plan.Relations {
		switch r.RelationTypeID {
		case createdBy:
			roles[r.From+" → "+r.To] = string(r.Attributes)
		case containsWork:
			if r.From == "wikidata:"+cycleAbsurd {
				order = string(r.Attributes)
			}
		}
	}
	if order != `{"order":1}` {
		t.Errorf("cycle order: %s", order)
	}
	for link, want := range map[string]string{
		"wikidata:" + etranger + " → wikidata:Q34670": `{"roles":["author"]}`,
		"bnf:cb35215528k → bnf:cb14505573c":           `{"roles":["translator"]}`,
		"bnf:cb35215528k → bnf:cb11897525z":           `{"roles":["preface"]}`,
	} {
		if roles[link] != want {
			t.Errorf("%s: %q, want %q (all: %v)", link, roles[link], want, roles)
		}
	}

	// Import : chaque nœud a son type, le roman est un Novel.
	sel := make([]string, 0, len(plan.Nodes))
	for _, n := range plan.Nodes {
		sel = append(sel, n.Key)
	}
	exp, ids, err := conn.Build(context.Background(), model.ExternalImportRequest{
		Work: etranger, Editions: []string{folio2013, outsider}, Selected: sel,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(exp.Items) != len(plan.Nodes) || len(ids) != len(plan.Nodes) {
		t.Errorf("build: %d items, %d ids for %d nodes", len(exp.Items), len(ids), len(plan.Nodes))
	}
}

func TestNovelsPreviewMultiVolume(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.NovelsConnector(connectorstest.NewCatalog(t))
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Editions: []string{pocketSet}})
	if err != nil {
		t.Fatal(err)
	}
	var edition model.ExternalNode
	var numbers []float64
	for _, n := range plan.Nodes {
		switch n.Role {
		case model.NodeEdition:
			edition = n
		case model.NodeVolume:
			numbers = append(numbers, jsonValue(t, n.Attributes, "number").(float64))
			if jsonValue(t, n.Attributes, "isbn") == nil {
				t.Errorf("volume without ISBN: %s", n.Attributes)
			}
		}
	}
	slices.Sort(numbers)
	if !slices.Equal(numbers, []float64{1, 2, 3}) || jsonValue(t, edition.Attributes, "volume_count") != float64(3) {
		t.Errorf("edition %s, volumes %v", edition.Attributes, numbers)
	}
	// L'ISBN est porté par les tomes, pas par l'ensemble.
	if jsonValue(t, edition.Attributes, "isbn") != nil {
		t.Errorf("set with an ISBN: %s", edition.Attributes)
	}
}

// jsonValue reads one top-level attribute of a node.
func jsonValue(t *testing.T, raw json.RawMessage, key string) any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m[key]
}
