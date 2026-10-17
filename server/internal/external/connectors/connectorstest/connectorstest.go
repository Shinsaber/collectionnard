// Package connectorstest serves recorded answers of the sources (Wikidata and
// the BnF: Dragon Ball, L'Étranger, Les Misérables, Amélie, Breaking Bad, Mario Kart 8 ; MusicBrainz ;
// TVmaze) and an in-memory catalogue
// holding the types of the seeds, for the tests of the connector and of its
// handlers.
package connectorstest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/domains"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/connectors"
	"github.com/shinsaber/colec/internal/model"
)

// Today is the retrieval date of the recorded answers.
var Today = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

// Recorded BnF queries → files of testdata/.
var sruAnswers = map[string]string{
	`bib.recordid all "34338439"`:                              "bnf_set.xml",
	`bib.ens2bib all "34338439" or bib.col2bib all "34338439"`: "bnf_volumes.xml",
	`bib.fuzzyIsbn all "9782876952058"`:                        "bnf_isbn.xml",
	`bib.recordid any "34338597 34338439"`:                     "bnf_isbn_sets.xml",
	`bib.serialtitle all "dragon ball"`:                        "bnf_serialtitle.xml",
	// Romans (P3-3) : L'Étranger (BnF œuvre 11958514), Les Misérables (13516296, édition Pocket 45767941).
	`bib.work2bib all "11958514"`:          "novel_work2bib.xml",
	`bib.recordid all "43662393"`:          "novel_folio.xml",
	`bib.recordid all "35215528"`:          "novel_outsider.xml",
	`bib.work2bib all "13516296"`:          "novel_mis_work2bib.xml",
	`bib.recordid any "33046180 45767941"`: "novel_mis_sets.xml",
	`bib.recordid all "45767941"`:          "novel_mis_set.xml",
	`bib.ens2bib all "45767941"`:           "novel_mis_volumes.xml",
	// Films (P3-5) : Amélie (notice d'œuvre BnF 14404106), DVD TF1 2010 (42284376) et Blu-ray collector (42284132).
	`bib.work2bib all "14404106"`: "film_work2bib.xml",
	`bib.recordid all "42284376"`: "film_dvd.xml",
	`bib.recordid all "42284132"`: "film_bluray.xml",
	`bib.ean all "3384442032803"`: "film_ean.xml",
	// Séries (P3-6) : coffrets Breaking Bad (saison 1 en DVD, « Saison finale [Saison 5] »).
	`bib.title all "Breaking Bad" and bib.doctype any "h"`: "series_title.xml",
	`bib.recordid all "43511598"`:                          "series_s1_dvd.xml",
	`bib.recordid all "43745269"`:                          "series_final_dvd.xml",
	`bib.ean all "3333297204842"`:                          "series_ean.xml",
	// Jeux vidéo (P3-7) : Mario Kart 8 Deluxe (cartouche Switch, notice brève avec EAN), Mario Kart 8 sur Wii U.
	`bib.title all "Mario Kart 8 Deluxe"`:                     "game_title.xml",
	`bib.recordid all "45382491"`:                             "game_switch.xml",
	`bib.recordid all "48815889"`:                             "game_brief.xml",
	`bib.recordid all "43877942"`:                             "game_wiiu.xml",
	`bib.ean all "045496333133" or bib.ean all "45496333133"`: "game_ean.xml",
}

// Recorded MusicBrainz answers (2026-10-17, tags and genres removed: not CC0): OK Computer
// (release group b1392450…), its 1997 double vinyl (541a0976…) and 2016 XL CD (0c324f3a…), Radiohead.
var mbEntities = map[string]string{
	"/ws/2/release-group/" + OKComputer:                 "mb_release_group.json",
	"/ws/2/release/" + OKComputerVinyl:                  "mb_release_vinyl.json",
	"/ws/2/release/" + OKComputerCD:                     "mb_release_cd.json",
	"/ws/2/artist/a74b1b7f-71a5-4011-9441-d0b5e4122711": "mb_artist.json",
}

// MusicBrainz identifiers of the recorded answers.
const (
	OKComputer      = "b1392450-e666-3926-a536-22c65f834433"
	OKComputerVinyl = "541a0976-ca45-3c0f-89e5-26bc376f58d1"
	OKComputerCD    = "0c324f3a-dcbb-4f09-b33a-c03c676b4044"
	OKComputerUPC   = "634904078126"
)

const emptySRU = `<?xml version="1.0" encoding="UTF-8"?><srw:searchRetrieveResponse xmlns:srw="http://www.loc.gov/zing/srw/"><srw:version>1.2</srw:version><srw:numberOfRecords>0</srw:numberOfRecords><srw:records></srw:records></srw:searchRetrieveResponse>`

// Sources is a fake Wikidata (/w/api.php), BnF (/api/SRU) and MusicBrainz (/ws/2).
type Sources struct {
	*httptest.Server
	// Requests counts the answered requests (cache tests).
	Requests atomic.Int32
	// Down makes every request fail with HTTP 503.
	Down atomic.Bool
}

func testdata(t testing.TB, name string) []byte {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "testdata", name)) //nolint:gosec // test fixtures: fixed directory, names from sruAnswers and mbEntities
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// NewSources starts the fake sources (closed at the end of the test).
func NewSources(t testing.TB) *Sources {
	t.Helper()
	var entities struct {
		Entities map[string]json.RawMessage `json:"entities"`
	}
	for _, name := range []string{"wikidata_entities.json", "wikidata_entities_novels.json", "wikidata_entities_films.json", "wikidata_entities_series.json", "wikidata_entities_games.json"} {
		if err := json.Unmarshal(testdata(t, name), &entities); err != nil {
			t.Fatal(err)
		}
	}
	search := testdata(t, "wikidata_search.json")
	searchNovels := testdata(t, "wikidata_search_novels.json")
	searchFilms := testdata(t, "wikidata_search_films.json")
	searchFilmBnF := testdata(t, "wikidata_search_film_bnf.json")
	searchSeries := testdata(t, "wikidata_search_series.json")
	searchGames := testdata(t, "wikidata_search_games.json")
	searchGamesMK8 := testdata(t, "wikidata_search_games_mk8.json")
	s := &Sources{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Requests.Add(1)
		if s.Down.Load() {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		q := r.URL.Query()
		switch r.URL.Path {
		case "/w/api.php":
			w.Header().Set("Content-Type", "application/json")
			switch q.Get("action") {
			case "query":
				switch srsearch := q.Get("srsearch"); {
				case strings.Contains(srsearch, "P268=14404106q"):
					_, _ = w.Write(searchFilmBnF) // film d'une notice d'œuvre BnF
					return
				case strings.HasPrefix(srsearch, "Mario kart 8 haswbstatement:P31=Q7889"):
					_, _ = w.Write(searchGamesMK8) // titre d'une notice BnF (édition seule)
					return
				case strings.Contains(srsearch, "P31=Q7889"):
					_, _ = w.Write(searchGames) // classe des jeux vidéo
					return
				case strings.Contains(srsearch, "Q5398426"):
					_, _ = w.Write(searchSeries) // classes des séries
					return
				case strings.Contains(srsearch, "Q11424"):
					_, _ = w.Write(searchFilms) // classes des films
					return
				}
				if strings.Contains(q.Get("srsearch"), "Q7725634") {
					_, _ = w.Write(searchNovels) // classes des romans
					return
				}
				_, _ = w.Write(search)
			case "wbgetentities":
				out := map[string]json.RawMessage{}
				for _, id := range strings.Split(q.Get("ids"), "|") {
					e, ok := entities.Entities[id]
					if !ok {
						_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
							"code": "no-such-entity", "info": "Could not find an entity with the ID " + id,
						}})
						return
					}
					out[id] = e
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"entities": out})
			case "wbgetclaims":
				code := map[string]string{"Q17": "JP", "Q142": "FR", "Q150": "fr", "Q183": "DE", "Q31": "BE", "Q30": "US", "Q1860": "en"}[q.Get("entity")]
				claims := []any{}
				if code != "" {
					claims = append(claims, map[string]any{"rank": "normal", "mainsnak": map[string]any{
						"datavalue": map[string]any{"type": "string", "value": code}}})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"claims": map[string]any{q.Get("property"): claims}})
			default:
				http.NotFound(w, r)
			}
		case "/tvmaze/lookup/shows", "/tvmaze/shows/169/seasons", "/tvmaze/shows/169/episodes":
			// TVmaze (CC BY-SA, résumés et images retirés) : Breaking Bad.
			name := map[string]string{"/tvmaze/lookup/shows": "tvmaze_show.json", "/tvmaze/shows/169/seasons": "tvmaze_seasons.json",
				"/tvmaze/shows/169/episodes": "tvmaze_episodes.json"}[r.URL.Path]
			if r.URL.Path == "/tvmaze/lookup/shows" && q.Get("imdb") != "tt0903747" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(testdata(t, name))
		case "/ws/2/release-group", "/ws/2/release":
			w.Header().Set("Content-Type", "application/json")
			switch {
			case strings.Contains(q.Get("query"), "releasegroup:computer"):
				_, _ = w.Write(testdata(t, "mb_search_release_groups.json"))
			case q.Get("query") == "barcode:"+OKComputerUPC:
				_, _ = w.Write(testdata(t, "mb_search_barcode.json"))
			case q.Get("release-group") == OKComputer:
				_, _ = w.Write(testdata(t, "mb_releases_of.json"))
			default:
				_, _ = w.Write([]byte(`{"release-groups":[],"releases":[]}`))
			}
		case "/api/SRU":
			w.Header().Set("Content-Type", "application/xml")
			if name, ok := sruAnswers[q.Get("query")]; ok {
				_, _ = w.Write(testdata(t, name))
				return
			}
			_, _ = w.Write([]byte(emptySRU))
		default:
			if name, ok := mbEntities[r.URL.Path]; ok {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(testdata(t, name))
				return
			}
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

// Connector returns a connector reading these sources and cat, dated Today.
func (s *Sources) Connector(cat connectors.Catalog) *connectors.Connector {
	c := connectors.New(cat, external.NewClient("collectionnard-test"))
	c.WD.BaseURL = s.URL + "/w/api.php"
	c.BnF.BaseURL = s.URL + "/api/SRU"
	c.Now = func() time.Time { return Today }
	return c
}

// Domain returns the item types and relation types of the domain packages
// (server/internal/domains/packages).
func Domain(t testing.TB) ([]model.ItemType, []model.RelationType) {
	t.Helper()
	pkgs, err := domains.All()
	if err != nil {
		t.Fatal(err)
	}
	var its []model.ItemType
	var rts []model.RelationType
	for _, p := range pkgs {
		for _, it := range p.ItemTypes {
			its = append(its, model.ItemType{ID: it.ID, Name: it.Name, AttributesSchema: it.AttributesSchema})
		}
		for _, rt := range p.RelationTypes {
			rts = append(rts, model.RelationType{ID: rt.ID, Name: rt.Name, AttributesSchema: rt.AttributesSchema})
		}
	}
	return its, rts
}

// TypeID returns the id of a seed type by name.
func TypeID(t testing.TB, name string) uuid.UUID {
	t.Helper()
	its, rts := Domain(t)
	for _, it := range its {
		if it.Name == name {
			return it.ID
		}
	}
	for _, rt := range rts {
		if rt.Name == name {
			return rt.ID
		}
	}
	t.Fatalf("no seed type %q", name)
	return uuid.Nil
}

// Catalog is an in-memory connectors.Catalog.
type Catalog struct {
	ItemTypes     []model.ItemType
	RelationTypes []model.RelationType
	Items         []model.Item
	Relations     []model.ItemRelation
}

// NewCatalog returns a catalogue holding the seed types and no item.
func NewCatalog(t testing.TB) *Catalog {
	its, rts := Domain(t)
	return &Catalog{ItemTypes: its, RelationTypes: rts}
}

func (c *Catalog) ListItemTypes(context.Context, model.Pagination) ([]model.ItemType, int, error) {
	return c.ItemTypes, len(c.ItemTypes), nil
}

func (c *Catalog) ListRelationTypes(context.Context, model.Pagination, *uuid.UUID, *uuid.UUID) ([]model.RelationType, int, error) {
	return c.RelationTypes, len(c.RelationTypes), nil
}

// FindItemsByAttribute compares exactly, or case-insensitively when fold is
// set (accents are folded by PostgreSQL only).
func (c *Catalog) FindItemsByAttribute(_ context.Context, typeID uuid.UUID, path, values []string, fold bool) ([]model.Item, error) {
	var out []model.Item
	for _, it := range c.Items {
		if it.ItemTypeID != typeID {
			continue
		}
		var cur any
		_ = json.Unmarshal(it.Attributes, &cur)
		for _, k := range path {
			m, _ := cur.(map[string]any)
			cur = m[k]
		}
		v, _ := cur.(string)
		if slices.ContainsFunc(values, func(x string) bool { return x == v || (fold && strings.EqualFold(x, v)) }) {
			out = append(out, it)
		}
	}
	return out, nil
}

// ListItemRelations returns the relations of an item (source or target), with their target item.
func (c *Catalog) ListItemRelations(_ context.Context, itemID uuid.UUID, relationTypeID *uuid.UUID) ([]model.ItemRelation, error) {
	var out []model.ItemRelation
	for _, r := range c.Relations {
		if (r.SourceItemID != itemID && r.TargetItemID != itemID) || (relationTypeID != nil && r.RelationTypeID != *relationTypeID) {
			continue
		}
		for i := range c.Items {
			if c.Items[i].ID == r.TargetItemID {
				it := c.Items[i]
				r.TargetItem = &it
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func (c *Catalog) ListRelationsBetween(_ context.Context, ids []uuid.UUID) ([]model.ItemRelation, error) {
	var out []model.ItemRelation
	for _, r := range c.Relations {
		if slices.Contains(ids, r.SourceItemID) && slices.Contains(ids, r.TargetItemID) {
			out = append(out, r)
		}
	}
	return out, nil
}

// NovelsConnector returns the novels connector reading these sources and cat, dated Today.
func (s *Sources) NovelsConnector(cat connectors.Catalog) *connectors.Connector {
	c := connectors.NewNovels(cat, external.NewClient("collectionnard-test"))
	c.WD.BaseURL = s.URL + "/w/api.php"
	c.BnF.BaseURL = s.URL + "/api/SRU"
	c.Now = func() time.Time { return Today }
	return c
}

// MusicConnector returns the music connector reading these sources and cat, dated Today.
func (s *Sources) MusicConnector(cat connectors.Catalog) *connectors.Connector {
	c := connectors.NewMusic(cat, external.NewClient("collectionnard-test"))
	c.MB.BaseURL = s.URL + "/ws/2"
	c.Now = func() time.Time { return Today }
	return c
}

// FilmsConnector returns the films connector reading these sources and cat, dated Today.
func (s *Sources) FilmsConnector(cat connectors.Catalog) *connectors.Connector {
	c := connectors.NewFilms(cat, external.NewClient("collectionnard-test"))
	c.WD.BaseURL = s.URL + "/w/api.php"
	c.BnF.BaseURL = s.URL + "/api/SRU"
	c.Now = func() time.Time { return Today }
	return c
}

// SeriesConnector returns the series connector reading these sources and cat, dated Today.
func (s *Sources) SeriesConnector(cat connectors.Catalog) *connectors.Connector {
	c := connectors.NewSeries(cat, external.NewClient("collectionnard-test"))
	c.WD.BaseURL = s.URL + "/w/api.php"
	c.BnF.BaseURL = s.URL + "/api/SRU"
	c.TV.BaseURL = s.URL + "/tvmaze"
	c.Now = func() time.Time { return Today }
	return c
}

// GamesConnector returns the video games connector reading these sources and cat, dated Today.
func (s *Sources) GamesConnector(cat connectors.Catalog) *connectors.Connector {
	c := connectors.NewGames(cat, external.NewClient("collectionnard-test"))
	c.WD.BaseURL = s.URL + "/w/api.php"
	c.BnF.BaseURL = s.URL + "/api/SRU"
	c.Now = func() time.Time { return Today }
	return c
}
