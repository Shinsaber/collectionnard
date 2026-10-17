package connectors_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external/connectors/connectorstest"
	"github.com/shinsaber/colec/internal/model"
)

// linkURLs returns label → URL of the links of a node.
func linkURLs(t *testing.T, n model.ExternalNode) map[string]string {
	t.Helper()
	out := map[string]string{}
	links, _ := attrs(t, n.Attributes)["links"].([]any)
	for _, l := range links {
		m := l.(map[string]any)
		out[m["label"].(string)] = m["url"].(string)
	}
	return out
}

// Every imported item links to the records it comes from.
func TestSourceLinks(t *testing.T) {
	src := connectorstest.NewSources(t)
	ctx := context.Background()

	films, err := src.FilmsConnector(connectorstest.NewCatalog(t)).Preview(ctx, model.ExternalImportRequest{Work: amelie, Editions: []string{amelieBR}})
	if err != nil {
		t.Fatal(err)
	}
	got := linkURLs(t, nodeByKey(t, films, "wikidata:"+amelie))
	want := map[string]string{
		"Wikidata":       "https://www.wikidata.org/wiki/Q484048",
		"Wikipédia (fr)": "https://fr.wikipedia.org/wiki/Le_Fabuleux_Destin_d%27Am%C3%A9lie_Poulain",
		"Wikipedia (en)": "https://en.wikipedia.org/wiki/Am%C3%A9lie",
		"BnF":            "https://catalogue.bnf.fr/ark:/12148/cb14404106q",
		"IMDb":           "https://www.imdb.com/title/tt0211915/",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("film link %s = %q, want %q (all: %v)", k, got[k], v, got)
		}
	}
	if l := linkURLs(t, nodeByKey(t, films, "bnf:cb42284132m")); l["BnF"] != "https://catalogue.bnf.fr/ark:/12148/cb42284132m" || len(l) != 1 {
		t.Errorf("Blu-ray links: %v", l)
	}

	music, err := src.MusicConnector(connectorstest.NewCatalog(t)).Preview(ctx, model.ExternalImportRequest{Editions: []string{connectorstest.OKComputerVinyl}})
	if err != nil {
		t.Fatal(err)
	}
	if l := linkURLs(t, nodeByKey(t, music, mbKey("recording", airbag))); l["MusicBrainz"] != "https://musicbrainz.org/recording/"+airbag {
		t.Errorf("track links: %v", l)
	}

	series, err := src.SeriesConnector(connectorstest.NewCatalog(t)).Preview(ctx, model.ExternalImportRequest{Work: breakingBad})
	if err != nil {
		t.Fatal(err)
	}
	if l := linkURLs(t, nodeByKey(t, series, "wikidata:"+breakingBad)); l["TVmaze"] != "https://www.tvmaze.com/shows/169" || l["IMDb"] == "" {
		t.Errorf("series links: %v", l)
	}
	for _, n := range series.Nodes {
		if n.Role == model.NodeVolume && n.Parent == "wikidata:"+bbSeason5 {
			if l := linkURLs(t, n); len(l) == 0 || l["TVmaze"] == "" {
				t.Errorf("TVmaze episode links: %v", l)
			}
			break
		}
	}
}

// An existing item keeps its links and gains the missing ones.
func TestSourceLinksMergedIntoExisting(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	cat.Items = []model.Item{{ID: uuid.New(), ItemTypeID: connectorstest.TypeID(t, "Film"), Attributes: []byte(
		`{"title":{"fr":"Amélie"},"external_ids":{"wikidata":"Q484048"},"links":[{"label":"Site officiel","url":"https://example.org/amelie"},{"label":"Wikidata","url":"https://www.wikidata.org/wiki/Q484048"}]}`)}}
	plan, err := src.FilmsConnector(cat).Preview(context.Background(), model.ExternalImportRequest{Work: amelie})
	if err != nil {
		t.Fatal(err)
	}
	film := nodeByKey(t, plan, "wikidata:"+amelie)
	links, _ := attrs(t, film.Attributes)["links"].([]any)
	if len(links) != 6 || links[0].(map[string]any)["label"] != "Site officiel" {
		t.Errorf("merged links: %v", links)
	}
}
