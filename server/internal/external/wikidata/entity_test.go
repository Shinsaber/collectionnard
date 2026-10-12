package wikidata

import (
	"encoding/json"
	"testing"
)

const toriyama = `{"id":"Q208582","labels":{"fr":{"value":"Akira Toriyama"}},"descriptions":{},"claims":{
 "P569":[{"rank":"normal","mainsnak":{"datavalue":{"type":"time","value":{"time":"+1955-04-05T00:00:00Z","precision":11}}}}],
 "P2635":[{"rank":"normal","mainsnak":{"datavalue":{"type":"quantity","value":{"amount":"+520","unit":"http://www.wikidata.org/entity/Q53460949"}}}},
          {"rank":"normal","mainsnak":{"datavalue":{"type":"quantity","value":{"amount":"+42","unit":"http://www.wikidata.org/entity/Q1238720"}}}}],
 "P27":[{"rank":"deprecated","mainsnak":{"datavalue":{"type":"wikibase-entityid","value":{"id":"Q30"}}}},
        {"rank":"normal","mainsnak":{"datavalue":{"type":"wikibase-entityid","value":{"id":"Q1"}}}},
        {"rank":"preferred","mainsnak":{"datavalue":{"type":"wikibase-entityid","value":{"id":"Q17"}}}}],
 "P1476":[{"rank":"normal","mainsnak":{"datavalue":{"type":"monolingualtext","value":{"text":"ドラゴンボール","language":"ja"}}}}],
 "P580":[{"rank":"normal","mainsnak":{"snaktype":"somevalue"}},
         {"rank":"normal","mainsnak":{"datavalue":{"type":"time","value":{"time":"+1984-00-00T00:00:00Z","precision":9}}}}]
}}`

func TestEntityValues(t *testing.T) {
	var raw rawEntity
	if err := json.Unmarshal([]byte(toriyama), &raw); err != nil {
		t.Fatal(err)
	}
	e := raw.entity()
	if d, ok := e.Date("P569"); !ok || d != "1955-04-05" {
		t.Errorf("Date = %q, %v", d, ok)
	}
	if _, ok := e.Date("P580"); ok {
		t.Error("a year-precision date is not a full date")
	}
	if y, ok := e.Year("P580"); !ok || y != 1984 {
		t.Errorf("Year = %d, %v", y, ok)
	}
	if n, ok := e.Quantity("P2635", "Q1238720"); !ok || n != 42 {
		t.Errorf("Quantity = %d, %v", n, ok)
	}
	// Preferred first, deprecated dropped.
	if ids := e.IDs("P27"); len(ids) != 2 || ids[0] != "Q17" || ids[1] != "Q1" {
		t.Errorf("IDs = %v", ids)
	}
	if txt, lang := e.Text("P1476"); txt != "ドラゴンボール" || lang != "ja" {
		t.Errorf("Text = %q, %q", txt, lang)
	}
	if e.Label("en", "fr") != "Akira Toriyama" || e.Label("en") != "" {
		t.Error("Label fallback")
	}
	if !IsQID("Q42") || IsQID("Q042") || IsQID("P31") {
		t.Error("IsQID")
	}
}
