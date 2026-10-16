// Package bdmangatest serves recorded answers of Wikidata and the BnF (Dragon
// Ball, Glénat 1993 edition, retrieved 2026-10-11) and an in-memory catalogue
// holding the types of the seeds, for the tests of the connector and of its
// handlers.
package bdmangatest

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
	"github.com/shinsaber/colec/internal/external/bdmanga"
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
}

const emptySRU = `<?xml version="1.0" encoding="UTF-8"?><srw:searchRetrieveResponse xmlns:srw="http://www.loc.gov/zing/srw/"><srw:version>1.2</srw:version><srw:numberOfRecords>0</srw:numberOfRecords><srw:records></srw:records></srw:searchRetrieveResponse>`

// Sources is a fake Wikidata (/w/api.php) and BnF (/api/SRU).
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
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "testdata", name)) //nolint:gosec // test fixtures: fixed directory, names from sruAnswers
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
	if err := json.Unmarshal(testdata(t, "wikidata_entities.json"), &entities); err != nil {
		t.Fatal(err)
	}
	search := testdata(t, "wikidata_search.json")
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
				code := map[string]string{"Q17": "JP", "Q142": "FR"}[q.Get("entity")]
				claims := []any{}
				if code != "" {
					claims = append(claims, map[string]any{"rank": "normal", "mainsnak": map[string]any{
						"datavalue": map[string]any{"type": "string", "value": code}}})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"claims": map[string]any{q.Get("property"): claims}})
			default:
				http.NotFound(w, r)
			}
		case "/api/SRU":
			w.Header().Set("Content-Type", "application/xml")
			if name, ok := sruAnswers[q.Get("query")]; ok {
				_, _ = w.Write(testdata(t, name))
				return
			}
			_, _ = w.Write([]byte(emptySRU))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

// Connector returns a connector reading these sources and cat, dated Today.
func (s *Sources) Connector(cat bdmanga.Catalog) *bdmanga.Connector {
	c := bdmanga.New(cat, external.NewClient("collectionnard-test"))
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

// Catalog is an in-memory bdmanga.Catalog.
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

func (c *Catalog) ListRelationsBetween(_ context.Context, ids []uuid.UUID) ([]model.ItemRelation, error) {
	var out []model.ItemRelation
	for _, r := range c.Relations {
		if slices.Contains(ids, r.SourceItemID) && slices.Contains(ids, r.TargetItemID) {
			out = append(out, r)
		}
	}
	return out, nil
}
