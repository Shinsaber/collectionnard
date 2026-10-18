package identifier

import (
	"encoding/json"
	"reflect"
	"testing"
)

const schema = `{"type":"object","properties":{
  "isbn":{"type":"string","x-identifier":true},
  "title":{"type":"string"},
  "external_ids":{"type":"object","properties":{
    "wikidata":{"type":"string","x-identifier":true},
    "note":{"type":"string"}}}}}`

func TestFields(t *testing.T) {
	got := Fields(json.RawMessage(schema))
	want := []Field{{"external_ids", "wikidata"}, {"isbn"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
	if Fields(json.RawMessage(`not json`)) != nil {
		t.Error("invalid schema: no field")
	}
}

func TestValue(t *testing.T) {
	attrs := json.RawMessage(`{"isbn":" 978-2 ","external_ids":{"wikidata":"Q1"},"title":"x"}`)
	if v, ok := Value(attrs, Field{"isbn"}); !ok || v != "978-2" {
		t.Errorf("isbn: %q %v", v, ok)
	}
	if v, ok := Value(attrs, Field{"external_ids", "wikidata"}); !ok || v != "Q1" {
		t.Errorf("wikidata: %q %v", v, ok)
	}
	for _, f := range []Field{{"missing"}, {"title", "deep"}, {"external_ids", "bnf_ark"}} {
		if _, ok := Value(attrs, f); ok {
			t.Errorf("%v must be absent", f)
		}
	}
	if _, ok := Value(json.RawMessage(`{"isbn":"  "}`), Field{"isbn"}); ok {
		t.Error("blank value is no identifier")
	}
}
