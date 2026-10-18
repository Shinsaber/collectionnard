// Package privacy decides who may read a collection or a profile and strips
// the hidden attributes of a collection from what other readers receive (P4-2).
//
// Hidden fields default to the properties marked `x-private: true` in the
// CollectionType schema; the owner may replace that list per collection
// (`collections.hidden_fields`). Every read made for someone else than the
// owner goes through Strip*: the values never leave the server, admins
// included.
package privacy

import (
	"bytes"
	"encoding/json"
	"slices"
	"sort"

	"github.com/google/uuid"
)

// Visibility levels of a collection or a profile.
const (
	Private = "private" // the owner only
	Members = "members" // any signed-in account
	Public  = "public"  // anyone, signed in or not
)

// Valid tells whether v is a visibility level.
func Valid(v string) bool {
	return v == Private || v == Members || v == Public
}

// CanView tells whether viewer (uuid.Nil when anonymous) may read something
// of owner with the given visibility.
func CanView(visibility string, owner, viewer uuid.UUID) bool {
	switch {
	case viewer != uuid.Nil && viewer == owner:
		return true
	case visibility == Public:
		return true
	case visibility == Members:
		return viewer != uuid.Nil
	}
	return false
}

// Levels lists the visibilities a non-owner viewer may read.
func Levels(viewer uuid.UUID) []string {
	if viewer == uuid.Nil {
		return []string{Public}
	}
	return []string{Members, Public}
}

type rootSchema struct {
	Order      []string                   `json:"x-order"`
	Properties map[string]json.RawMessage `json:"properties"`
}

// Properties lists the property keys of a schema, x-order first then by key.
func Properties(schema json.RawMessage) []string {
	var root rootSchema
	if len(schema) == 0 || json.Unmarshal(schema, &root) != nil {
		return nil
	}
	var keys []string
	for _, k := range root.Order {
		if _, ok := root.Properties[k]; ok && !slices.Contains(keys, k) {
			keys = append(keys, k)
		}
	}
	var rest []string
	for k := range root.Properties {
		if !slices.Contains(keys, k) {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

// DefaultHidden lists the properties marked `x-private: true`.
func DefaultHidden(schema json.RawMessage) []string {
	var root rootSchema
	if len(schema) == 0 || json.Unmarshal(schema, &root) != nil {
		return nil
	}
	out := []string{}
	for _, k := range Properties(schema) {
		var p struct {
			Private bool `json:"x-private"`
		}
		if json.Unmarshal(root.Properties[k], &p) == nil && p.Private {
			out = append(out, k)
		}
	}
	return out
}

// HiddenFields is the effective list: the owner's override when set (non-nil,
// possibly empty), the schema default otherwise.
func HiddenFields(schema json.RawMessage, override []string) []string {
	if override != nil {
		return override
	}
	return DefaultHidden(schema)
}

// UnknownFields returns the fields that are not properties of schema.
func UnknownFields(schema json.RawMessage, fields []string) []string {
	props := Properties(schema)
	var out []string
	for _, f := range fields {
		if !slices.Contains(props, f) {
			out = append(out, f)
		}
	}
	return out
}

func decode(raw json.RawMessage) (map[string]any, bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // numbers kept as written
	var m map[string]any
	if dec.Decode(&m) != nil || m == nil {
		return nil, false
	}
	return m, true
}

// StripAttributes removes the hidden keys from an attributes object. Invalid
// JSON yields an empty object: nothing unchecked is passed on.
func StripAttributes(attrs json.RawMessage, hidden []string) json.RawMessage {
	if len(hidden) == 0 && json.Valid(attrs) {
		return attrs
	}
	m, ok := decode(attrs)
	if !ok {
		return json.RawMessage(`{}`)
	}
	for _, k := range hidden {
		delete(m, k)
	}
	out, err := json.Marshal(m)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}

// StripSchema removes the hidden properties from a CollectionType schema, with
// their mentions in `required`, `x-order` and `x-labels`, so that readers do
// not even learn their labels.
func StripSchema(schema json.RawMessage, hidden []string) json.RawMessage {
	if len(hidden) == 0 {
		return schema
	}
	m, ok := decode(schema)
	if !ok {
		return json.RawMessage(`{}`)
	}
	isHidden := func(v any) bool {
		s, ok := v.(string)
		return ok && slices.Contains(hidden, s)
	}
	if props, ok := m["properties"].(map[string]any); ok {
		for _, k := range hidden {
			delete(props, k)
		}
	}
	for _, key := range []string{"required", "x-order"} {
		if list, ok := m[key].([]any); ok {
			m[key] = slices.DeleteFunc(list, isHidden)
		}
	}
	if labels, ok := m["x-labels"].(map[string]any); ok {
		for _, perLang := range labels {
			if l, ok := perLang.(map[string]any); ok {
				for _, k := range hidden {
					delete(l, k)
				}
			}
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}
