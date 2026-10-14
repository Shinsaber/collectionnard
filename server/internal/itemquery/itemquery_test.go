package itemquery

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

const schema = `{
	"type": "object",
	"properties": {
		"title":      {"type": "string"},
		"kind":       {"type": "string", "enum": ["tv", "web"]},
		"start_year": {"type": "integer"},
		"price":      {"type": "number"},
		"owned":      {"type": "boolean"},
		"aired":      {"type": "string", "format": "date"},
		"genres":     {"type": "array", "items": {"type": "string"}},
		"synopsis":   {"type": "object", "x-widget": "i18n-text"},
		"images":     {"type": "object", "x-widget": "image-gallery"}
	}
}`

func props(t *testing.T) map[string]Property {
	t.Helper()
	p, err := Properties(json.RawMessage(schema))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseFilters(t *testing.T) {
	q := url.Values{
		"attr.kind":              {"tv", "web"},
		"attr.start_year.gte":    {"2008"},
		"attr.price.lte":         {"9.5"},
		"attr.owned":             {"true"},
		"attr.title.contains":    {"bad"},
		"attr.synopsis.contains": {"chimie"},
		"attr.genres":            {"drama"},
		"attr.aired.gte":         {"2008-01-20"},
		"attr.start_year":        {""}, // empty form field: ignored
		"search":                 {"ignored"},
	}
	got, err := ParseFilters(q, props(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []Filter{
		{Key: "aired", Kind: KindDate, Op: OpGte, Values: []any{"2008-01-20"}},
		{Key: "genres", Kind: KindArray, Op: OpEq, Values: []any{"drama"}},
		{Key: "kind", Kind: KindString, Op: OpEq, Values: []any{"tv", "web"}},
		{Key: "owned", Kind: KindBoolean, Op: OpEq, Values: []any{true}},
		{Key: "price", Kind: KindNumber, Op: OpLte, Values: []any{9.5}},
		{Key: "start_year", Kind: KindInteger, Op: OpGte, Values: []any{int64(2008)}},
		{Key: "synopsis", Kind: KindI18n, Op: OpContains, Values: []any{"chimie"}},
		{Key: "title", Kind: KindString, Op: OpContains, Values: []any{"bad"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %#v\nwant %#v", got, want)
	}
}

func TestParseFiltersErrors(t *testing.T) {
	cases := map[string]url.Values{
		"unknown attribute":   {"attr.nope": {"x"}},
		"bad suffix":          {"attr.title.like": {"x"}},
		"op not for kind":     {"attr.title.gte": {"a"}},
		"contains on integer": {"attr.start_year.contains": {"1"}},
		"not an integer":      {"attr.start_year": {"abc"}},
		"not a boolean":       {"attr.owned": {"maybe"}},
		"gallery":             {"attr.images": {"x"}},
		"i18n equality":       {"attr.synopsis": {"x"}},
		"repeated range":      {"attr.start_year.gte": {"1", "2"}},
		"too long":            {"attr.title": {strings.Repeat("x", MaxValueLength+1)}},
		"too many":            {"attr.kind": make([]string, MaxFilters+1)},
	}
	for name, q := range cases {
		if _, err := ParseFilters(q, props(t)); err == nil {
			t.Errorf("%s: error expected", name)
		}
	}
}

func TestParseSort(t *testing.T) {
	p := props(t)
	s, ok, err := ParseSort("-attr.start_year", p)
	if err != nil || !ok || s != (Sort{Key: "start_year", Numeric: true, Desc: true}) {
		t.Fatalf("got %+v %v %v", s, ok, err)
	}
	s, ok, err = ParseSort("attr.title", p)
	if err != nil || !ok || s != (Sort{Key: "title"}) {
		t.Fatalf("got %+v %v %v", s, ok, err)
	}
	if _, ok, err := ParseSort("-created_at", p); ok || err != nil {
		t.Fatal("created_at is not an attribute sort")
	}
	for _, bad := range []string{"attr.nope", "attr.synopsis", "attr.genres"} {
		if _, _, err := ParseSort(bad, p); err == nil {
			t.Errorf("%s: error expected", bad)
		}
	}
}

func TestHasAttrParams(t *testing.T) {
	if HasAttrParams(url.Values{"search": {"x"}, "sort": {"-created_at"}}) {
		t.Error("no attribute parameter expected")
	}
	if !HasAttrParams(url.Values{"sort": {"-attr.title"}}) || !HasAttrParams(url.Values{"attr.x": {"1"}}) {
		t.Error("attribute parameters expected")
	}
}

func TestTSQuery(t *testing.T) {
	cases := map[string]string{
		"Breaking Bad":         "breaking:* & bad:*",
		"  série!  ":           "série:*",
		"a&b|!c:*(d)":          "a:* & b:* & c:* & d:*",
		"'; DROP TABLE items;": "drop:* & table:* & items:*",
		"--- ":                 "",
	}
	for in, want := range cases {
		if got := TSQuery(in); got != want {
			t.Errorf("TSQuery(%q) = %q, want %q", in, got, want)
		}
	}
}
