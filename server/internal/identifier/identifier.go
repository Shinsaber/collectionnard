// Package identifier reads the `x-identifier: true` properties of an ItemType
// schema (ISBN, EAN, Wikidata QID, BnF ARK…), at the top level or one level
// down in an object (`external_ids.wikidata`), and the values an item carries
// for them: duplicate detection before a write or a proposal (P4-3b).
package identifier

import (
	"encoding/json"
	"sort"
	"strings"
)

// Field is the path of an identifier property: ["isbn"] or ["external_ids", "wikidata"].
type Field []string

type prop struct {
	Type       any                        `json:"type"`
	Identifier bool                       `json:"x-identifier"`
	Properties map[string]json.RawMessage `json:"properties"`
}

// Fields lists the identifier properties of schema, sorted by path.
func Fields(schema json.RawMessage) []Field {
	var root struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if len(schema) == 0 || json.Unmarshal(schema, &root) != nil {
		return nil
	}
	var out []Field
	for k, raw := range root.Properties {
		var p prop
		if json.Unmarshal(raw, &p) != nil {
			continue
		}
		if p.Identifier {
			out = append(out, Field{k})
		}
		for sub, subRaw := range p.Properties {
			var sp prop
			if json.Unmarshal(subRaw, &sp) == nil && sp.Identifier {
				out = append(out, Field{k, sub})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.Join(out[i], ".") < strings.Join(out[j], ".") })
	return out
}

// Value returns the non-empty string value of f in attrs.
func Value(attrs json.RawMessage, f Field) (string, bool) {
	var cur any
	if json.Unmarshal(attrs, &cur) != nil {
		return "", false
	}
	for _, k := range f {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		cur = m[k]
	}
	s, ok := cur.(string)
	s = strings.TrimSpace(s)
	return s, ok && s != ""
}
