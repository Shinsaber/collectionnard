package collstats

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/shinsaber/colec/internal/model"
)

// Schéma façon « Possession » : un enum, un nombre, une date, un booléen, un texte (ignoré).
const ownership = `{
  "type": "object",
  "x-order": ["condition", "purchase_price", "acquisition_date"],
  "properties": {
    "condition": {"type": "string", "enum": ["new", "good", "poor"]},
    "purchase_price": {"type": "number", "minimum": 0},
    "acquisition_date": {"type": "string", "format": "date"},
    "signed": {"type": ["boolean", "null"]},
    "notes": {"type": "string"}
  }
}`

func entry(attrs string, added string) Entry {
	t, _ := time.Parse("2006-01-02", added)
	return Entry{Attributes: json.RawMessage(attrs), AddedAt: t}
}

func TestAggregate(t *testing.T) {
	res := Aggregate(json.RawMessage(ownership), []Entry{
		entry(`{"condition":"new","purchase_price":10,"acquisition_date":"2026-01-15","signed":true}`, "2026-01-20"),
		entry(`{"condition":"new","purchase_price":4.5,"acquisition_date":"2026-03-02"}`, "2026-03-03"),
		entry(`{"condition":"mint","purchase_price":"12"}`, "2026-03-04"), // hors enum ; prix non numérique
		entry(`{}`, "2026-03-05"),
		entry(`not json`, ""),
	})

	wantNumbers := []model.NumberStat{{Field: "purchase_price", Count: 2, Sum: 14.5, Avg: 7.25, Min: 4.5, Max: 10}}
	if !reflect.DeepEqual(res.Numbers, wantNumbers) {
		t.Errorf("numbers = %+v", res.Numbers)
	}

	if len(res.Distributions) != 2 {
		t.Fatalf("distributions = %+v", res.Distributions)
	}
	cond := res.Distributions[0]
	wantCond := []model.ValueCount{{Value: "new", Count: 2}, {Value: "good"}, {Value: "poor"}, {Value: "mint", Count: 1}}
	if cond.Field != "condition" || !reflect.DeepEqual(cond.Values, wantCond) || cond.Missing != 2 {
		t.Errorf("condition = %+v", cond)
	}
	signed := res.Distributions[1]
	if signed.Field != "signed" || signed.Values[0] != (model.ValueCount{Value: "true", Count: 1}) || signed.Missing != 4 {
		t.Errorf("signed = %+v", signed)
	}

	wantMonths := []model.MonthCount{{Month: "2026-01", Count: 1}, {Month: "2026-02"}, {Month: "2026-03", Count: 1}}
	if len(res.Timelines) != 1 || res.Timelines[0].Field != "acquisition_date" || !reflect.DeepEqual(res.Timelines[0].Months, wantMonths) {
		t.Errorf("timelines = %+v", res.Timelines)
	}
	wantAdded := []model.MonthCount{{Month: "2026-01", Count: 1}, {Month: "2026-02"}, {Month: "2026-03", Count: 3}}
	if !reflect.DeepEqual(res.Added, wantAdded) {
		t.Errorf("added = %+v", res.Added)
	}
}

func TestAggregate_EmptyOrInvalidSchema(t *testing.T) {
	for _, schema := range []string{``, `{}`, `[]`} {
		res := Aggregate(json.RawMessage(schema), nil)
		if res.Numbers == nil || res.Distributions == nil || res.Timelines == nil || res.Added == nil {
			t.Errorf("%q: nil slice in %+v (JSON null instead of [])", schema, res)
		}
	}
}

func TestMonths_LongSpanNotFilled(t *testing.T) {
	got := Months(map[string]int{"1900-01": 1, "2026-10": 2})
	if len(got) != 2 {
		t.Errorf("got %d months, want the 2 with values", len(got))
	}
}
