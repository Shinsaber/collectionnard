package privacy

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

const ownership = `{
  "type": "object",
  "x-order": ["condition", "purchase_price", "acquisition_date"],
  "required": ["condition", "purchase_price"],
  "x-labels": {"fr": {"condition": "État", "purchase_price": "Prix d'achat"}, "en": {"purchase_price": "Price"}},
  "properties": {
    "condition": {"type": "string"},
    "purchase_price": {"type": "number", "x-private": true},
    "acquisition_date": {"type": "string", "format": "date"}
  }
}`

func TestCanView(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	cases := []struct {
		vis    string
		viewer uuid.UUID
		want   bool
	}{
		{Private, owner, true},
		{Private, other, false},
		{Private, uuid.Nil, false},
		{Members, other, true},
		{Members, uuid.Nil, false},
		{Public, uuid.Nil, true},
		{"bogus", other, false},
	}
	for _, c := range cases {
		if got := CanView(c.vis, owner, c.viewer); got != c.want {
			t.Errorf("CanView(%s, viewer=%v) = %v", c.vis, c.viewer, got)
		}
	}
	if CanView(Private, uuid.Nil, uuid.Nil) {
		t.Error("an anonymous viewer is never the owner")
	}
}

func TestHiddenFields(t *testing.T) {
	schema := json.RawMessage(ownership)
	if got := HiddenFields(schema, nil); !reflect.DeepEqual(got, []string{"purchase_price"}) {
		t.Errorf("default = %v", got)
	}
	if got := HiddenFields(schema, []string{}); len(got) != 0 {
		t.Errorf("empty override shows everything, got %v", got)
	}
	if got := HiddenFields(schema, []string{"acquisition_date"}); !reflect.DeepEqual(got, []string{"acquisition_date"}) {
		t.Errorf("override = %v", got)
	}
	if got := UnknownFields(schema, []string{"condition", "nope"}); !reflect.DeepEqual(got, []string{"nope"}) {
		t.Errorf("unknown = %v", got)
	}
}

func TestStripAttributes(t *testing.T) {
	got := string(StripAttributes(json.RawMessage(`{"condition":"new","purchase_price":12.50,"n":12345678901234567890}`), []string{"purchase_price"}))
	if strings.Contains(got, "purchase_price") || !strings.Contains(got, `"condition":"new"`) || !strings.Contains(got, "12345678901234567890") {
		t.Errorf("got %s", got)
	}
	if got := string(StripAttributes(json.RawMessage(`not json`), nil)); got != `{}` {
		t.Errorf("invalid attributes: %s", got)
	}
	if got := string(StripAttributes(json.RawMessage(`[1]`), []string{"x"})); got != `{}` {
		t.Errorf("non-object attributes: %s", got)
	}
}

func TestStripSchema(t *testing.T) {
	got := string(StripSchema(json.RawMessage(ownership), []string{"purchase_price"}))
	if strings.Contains(got, "purchase_price") || strings.Contains(got, "Prix") || strings.Contains(got, "Price") {
		t.Errorf("hidden property still mentioned: %s", got)
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(got), &m)
	if !reflect.DeepEqual(m["x-order"], []any{"condition", "acquisition_date"}) || !reflect.DeepEqual(m["required"], []any{"condition"}) {
		t.Errorf("order / required: %v %v", m["x-order"], m["required"])
	}
	if same := StripSchema(json.RawMessage(ownership), nil); string(same) != ownership {
		t.Error("nothing hidden: schema returned as is")
	}
}
