package csvexport

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const schema = `{
	"type": "object",
	"x-order": ["title", "synopsis", "year"],
	"properties": {
		"year":     {"type": "integer"},
		"title":    {"type": "string"},
		"synopsis": {"type": "object", "x-widget": "i18n-text", "properties": {"fr": {}, "en": {}, "de": {}}},
		"genres":   {"type": "array", "items": {"type": "string"}},
		"images":   {"type": "object", "x-widget": "image-gallery"}
	}
}`

func TestColumns(t *testing.T) {
	var headers []string
	for _, c := range Columns(json.RawMessage(schema)) {
		headers = append(headers, c.Header())
	}
	want := []string{"title", "synopsis.fr", "synopsis.en", "synopsis.de", "year", "genres", "images"}
	if !reflect.DeepEqual(headers, want) {
		t.Fatalf("got %v, want %v", headers, want)
	}
	if Columns(nil) != nil || Columns(json.RawMessage(`not json`)) != nil {
		t.Error("no column expected for an empty or invalid schema")
	}
}

func TestValues(t *testing.T) {
	attrs := `{"title": "=HYPERLINK(1)", "synopsis": {"fr": "Bonjour", "en": "Hello"},
		"year": 2008, "genres": ["drama", "crime"], "images": {"poster": ["/a.jpg"]}}`
	got := Values(Columns(json.RawMessage(schema)), json.RawMessage(attrs))
	want := []string{"'=HYPERLINK(1)", "Bonjour", "Hello", "", "2008", "drama; crime", `{"poster":["/a.jpg"]}`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if v := Values(Columns(json.RawMessage(schema)), nil); len(v) != 7 || v[0] != "" {
		t.Errorf("empty attributes: %q", v)
	}
}

func TestCell(t *testing.T) {
	cases := map[string]any{
		"":            nil,
		"-3":          json.Number("-3"), // numbers are never quoted
		"1.5":         1.5,
		"true":        true,
		"'-negative":  "-negative",
		"'@sum":       "@sum",
		"plain":       "plain",
		`[{"a":1},2]`: []any{map[string]any{"a": 1}, 2},
	}
	for want, in := range cases {
		if got := Cell(in); got != want {
			t.Errorf("Cell(%#v) = %q, want %q", in, got, want)
		}
	}
}

func TestDelimiterAndWriter(t *testing.T) {
	for name, want := range map[string]rune{"": ',', "comma": ',', "semicolon": ';', "tab": '\t'} {
		if got, err := Delimiter(name); err != nil || got != want {
			t.Errorf("Delimiter(%q) = %q, %v", name, got, err)
		}
	}
	if _, err := Delimiter("pipe"); err == nil {
		t.Error("error expected")
	}
	var b strings.Builder
	w, err := Writer(&b, ';')
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Write([]string{"a", "b;c"})
	w.Flush()
	if b.String() != BOM+"a;\"b;c\"\r\n" {
		t.Errorf("got %q", b.String())
	}
}
