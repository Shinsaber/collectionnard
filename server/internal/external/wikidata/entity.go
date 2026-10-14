package wikidata

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Entity is a Wikidata item reduced to what the connectors read.
type Entity struct {
	ID           string
	Labels       map[string]string // language → label
	Descriptions map[string]string
	// Claims: property → statements, deprecated ones dropped, preferred first.
	Claims map[string][]Statement
}

// Statement is one value of a property, with its qualifiers.
type Statement struct {
	Value      Value
	Qualifiers map[string][]Value
}

// Value is a statement value; only the fields of its datatype are set.
type Value struct {
	ID        string // wikibase-entityid: "Q17"
	Text      string // string, external-id, url, monolingualtext
	Lang      string // monolingualtext language
	Time      string // time: "+1984-11-20T00:00:00Z"
	Precision int    // time: 9 = year, 10 = month, 11 = day
	Amount    string // quantity: "+42"
	Unit      string // quantity: "http://www.wikidata.org/entity/Q1238720" or "1"
}

// Label returns the first label found in langs.
func (e *Entity) Label(langs ...string) string {
	for _, l := range langs {
		if v := e.Labels[l]; v != "" {
			return v
		}
	}
	return ""
}

// IDs returns the entity values of a property, in statement order.
func (e *Entity) IDs(property string) []string {
	var out []string
	for _, st := range e.Claims[property] {
		if st.Value.ID != "" {
			out = append(out, st.Value.ID)
		}
	}
	return out
}

// HasID reports whether property has one of ids among its values.
func (e *Entity) HasID(property string, ids map[string]bool) bool {
	for _, id := range e.IDs(property) {
		if ids[id] {
			return true
		}
	}
	return false
}

// Text returns the first string (or monolingual text) value of a property.
func (e *Entity) Text(property string) (text, lang string) {
	for _, st := range e.Claims[property] {
		if st.Value.Text != "" {
			return st.Value.Text, st.Value.Lang
		}
	}
	return "", ""
}

// Year returns the year of the first time value of the properties, tried in
// order (precision year or finer).
func (e *Entity) Year(properties ...string) (int, bool) {
	for _, p := range properties {
		for _, st := range e.Claims[p] {
			if y, ok := st.Value.year(); ok {
				return y, true
			}
		}
	}
	return 0, false
}

// Date returns the first time value of a property precise to the day, as
// YYYY-MM-DD.
func (e *Entity) Date(property string) (string, bool) {
	for _, st := range e.Claims[property] {
		v := st.Value
		if v.Precision >= 11 && strings.HasPrefix(v.Time, "+") && len(v.Time) >= 11 {
			d := v.Time[1:11]
			if d[5:7] != "00" && d[8:10] != "00" {
				return d, true
			}
		}
	}
	return "", false
}

// Quantity returns the first integer amount of a property whose unit is one of
// units ("1" = no unit).
func (e *Entity) Quantity(property string, units ...string) (int, bool) {
	for _, st := range e.Claims[property] {
		v := st.Value
		if v.Amount == "" {
			continue
		}
		unit := strings.TrimPrefix(v.Unit, "http://www.wikidata.org/entity/")
		for _, u := range units {
			if unit == u {
				n, err := strconv.Atoi(strings.TrimPrefix(v.Amount, "+"))
				if err == nil {
					return n, true
				}
			}
		}
	}
	return 0, false
}

func (v Value) year() (int, bool) {
	if v.Precision < 9 || !strings.HasPrefix(v.Time, "+") {
		return 0, false
	}
	end := strings.IndexByte(v.Time[1:], '-')
	if end <= 0 {
		return 0, false
	}
	y, err := strconv.Atoi(v.Time[1 : 1+end])
	return y, err == nil
}

// ── JSON decoding ───────────────────────────────────────────────────

type rawEntity struct {
	ID           string                    `json:"id"`
	Missing      *string                   `json:"missing"`
	Labels       map[string]rawText        `json:"labels"`
	Descriptions map[string]rawText        `json:"descriptions"`
	Claims       map[string][]rawStatement `json:"claims"`
}

type rawText struct {
	Value string `json:"value"`
}

type rawStatement struct {
	Rank       string               `json:"rank"`
	Mainsnak   rawSnak              `json:"mainsnak"`
	Qualifiers map[string][]rawSnak `json:"qualifiers"`
}

type rawSnak struct {
	Datavalue *struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	} `json:"datavalue"`
}

func (r rawEntity) entity() *Entity {
	e := &Entity{
		ID:           r.ID,
		Labels:       texts(r.Labels),
		Descriptions: texts(r.Descriptions),
		Claims:       make(map[string][]Statement, len(r.Claims)),
	}
	for p, sts := range r.Claims {
		for _, st := range orderStatements(sts) {
			s := Statement{Value: st.value()}
			for qp, snaks := range st.Qualifiers {
				for _, sn := range snaks {
					if s.Qualifiers == nil {
						s.Qualifiers = map[string][]Value{}
					}
					s.Qualifiers[qp] = append(s.Qualifiers[qp], sn.value())
				}
			}
			e.Claims[p] = append(e.Claims[p], s)
		}
	}
	return e
}

func texts(m map[string]rawText) map[string]string {
	out := make(map[string]string, len(m))
	for l, t := range m {
		out[l] = t.Value
	}
	return out
}

// orderStatements drops deprecated statements and puts preferred ones first.
func orderStatements(sts []rawStatement) []rawStatement {
	out := make([]rawStatement, 0, len(sts))
	for _, st := range sts {
		if st.Rank == "preferred" {
			out = append(out, st)
		}
	}
	for _, st := range sts {
		if st.Rank != "preferred" && st.Rank != "deprecated" {
			out = append(out, st)
		}
	}
	return out
}

func (st rawStatement) value() Value { return st.Mainsnak.value() }

func (sn rawSnak) value() Value {
	var v Value
	if sn.Datavalue == nil {
		return v // "somevalue" / "novalue"
	}
	raw := sn.Datavalue.Value
	switch sn.Datavalue.Type {
	case "wikibase-entityid":
		var x struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &x)
		v.ID = x.ID
	case "string":
		_ = json.Unmarshal(raw, &v.Text)
	case "monolingualtext":
		var x struct {
			Text     string `json:"text"`
			Language string `json:"language"`
		}
		_ = json.Unmarshal(raw, &x)
		v.Text, v.Lang = x.Text, x.Language
	case "time":
		var x struct {
			Time      string `json:"time"`
			Precision int    `json:"precision"`
		}
		_ = json.Unmarshal(raw, &x)
		v.Time, v.Precision = x.Time, x.Precision
	case "quantity":
		var x struct {
			Amount string `json:"amount"`
			Unit   string `json:"unit"`
		}
		_ = json.Unmarshal(raw, &x)
		v.Amount, v.Unit = x.Amount, x.Unit
	}
	return v
}
