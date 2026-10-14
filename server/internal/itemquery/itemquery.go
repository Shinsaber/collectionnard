// Package itemquery parses the advanced search parameters of GET /items
// (attribute filters `attr.<key>[.<op>]`, attribute sort `sort=attr.<key>`,
// full-text `search`) and checks them against the JSON Schema of the item type.
// It is pure (no SQL): the store turns the result into parameterised SQL.
package itemquery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Op is a comparison operator of an attribute filter.
type Op string

const (
	OpEq       Op = "eq"       // attr.<key>=v (repeatable: OR); array → contains the element
	OpGte      Op = "gte"      // attr.<key>.gte=v (numbers, dates)
	OpLte      Op = "lte"      // attr.<key>.lte=v (numbers, dates)
	OpContains Op = "contains" // attr.<key>.contains=v (free text, i18n-text: any language)
)

// Kind is how an attribute is stored and compared, derived from its schema.
type Kind string

const (
	KindString  Kind = "string"  // plain or enum string
	KindDate    Kind = "date"    // string with format date / date-time (ISO, ordered as text)
	KindInteger Kind = "integer" //
	KindNumber  Kind = "number"  //
	KindBoolean Kind = "boolean" //
	KindArray   Kind = "array"   // array of scalars (eq = contains the element)
	KindI18n    Kind = "i18n"    // object {lang: text} (x-widget i18n-text)
)

const (
	// Prefix of attribute filter and sort parameters.
	Prefix = "attr."
	// MaxFilters caps the number of attribute filter values in one request.
	MaxFilters = 20
	// MaxValueLength caps the length of a filter value.
	MaxValueLength = 200
)

// Filter is one validated attribute condition. Values holds typed JSON values
// (string, int64, float64, bool) — several for OpEq means OR.
type Filter struct {
	Key    string
	Kind   Kind
	Op     Op
	Values []any
}

// Sort is a validated attribute sort.
type Sort struct {
	Key     string
	Numeric bool
	Desc    bool
}

// Property is the part of a schema property the parser needs.
type Property struct {
	Type   string   `json:"type"`
	Format string   `json:"format"`
	Enum   []any    `json:"enum"`
	Widget string   `json:"x-widget"`
	Items  *ItemDef `json:"items"`
}

// ItemDef describes the elements of an array property.
type ItemDef struct {
	Type string `json:"type"`
}

// Kind returns the comparison kind of a property, or "" when it cannot be
// filtered (nested objects, image galleries…).
func (p Property) Kind() Kind {
	switch p.Type {
	case "string":
		if p.Format == "date" || p.Format == "date-time" {
			return KindDate
		}
		return KindString
	case "integer":
		return KindInteger
	case "number":
		return KindNumber
	case "boolean":
		return KindBoolean
	case "array":
		if p.Items != nil && isScalar(p.Items.Type) {
			return KindArray
		}
	case "object":
		if p.Widget == "i18n-text" {
			return KindI18n
		}
	}
	return ""
}

func isScalar(t string) bool {
	return t == "string" || t == "integer" || t == "number" || t == "boolean"
}

// Ops lists the operators a kind accepts.
func (k Kind) Ops() []Op {
	switch k {
	case KindString:
		return []Op{OpEq, OpContains}
	case KindDate, KindInteger, KindNumber:
		return []Op{OpEq, OpGte, OpLte}
	case KindBoolean, KindArray:
		return []Op{OpEq}
	case KindI18n:
		return []Op{OpContains}
	}
	return nil
}

// Sortable reports whether ORDER BY on this kind is meaningful.
func (k Kind) Sortable() bool {
	switch k {
	case KindString, KindDate, KindInteger, KindNumber, KindBoolean:
		return true
	}
	return false
}

// Properties extracts the top-level properties of an attributes schema.
func Properties(schema json.RawMessage) (map[string]Property, error) {
	if len(schema) == 0 {
		return map[string]Property{}, nil
	}
	var root struct {
		Properties map[string]Property `json:"properties"`
	}
	if err := json.Unmarshal(schema, &root); err != nil {
		return nil, fmt.Errorf("invalid item type schema: %w", err)
	}
	if root.Properties == nil {
		root.Properties = map[string]Property{}
	}
	return root.Properties, nil
}

// HasAttrParams reports whether the query carries attribute filters or an
// attribute sort (both need the item type schema).
func HasAttrParams(q url.Values) bool {
	if strings.HasPrefix(strings.TrimPrefix(q.Get("sort"), "-"), Prefix) {
		return true
	}
	for name := range q {
		if strings.HasPrefix(name, Prefix) {
			return true
		}
	}
	return false
}

// ParseFilters reads every `attr.<key>[.<op>]` parameter of q and validates it
// against props. Errors are user-facing (400).
func ParseFilters(q url.Values, props map[string]Property) ([]Filter, error) {
	names := make([]string, 0, len(q))
	for name := range q {
		if strings.HasPrefix(name, Prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names) // deterministic SQL (and tests)

	var out []Filter
	count := 0
	for _, name := range names {
		key, op := splitParam(strings.TrimPrefix(name, Prefix))
		prop, ok := props[key]
		if !ok {
			return nil, fmt.Errorf("unknown attribute %q for this item type", key)
		}
		kind := prop.Kind()
		if !acceptsOp(kind, op) {
			return nil, fmt.Errorf("operator %q is not supported on attribute %q", op, key)
		}
		raw := q[name]
		if op != OpEq && len(raw) > 1 {
			return nil, fmt.Errorf("%s accepts a single value", name)
		}
		count += len(raw)
		if count > MaxFilters {
			return nil, fmt.Errorf("at most %d attribute filter values", MaxFilters)
		}
		f := Filter{Key: key, Kind: kind, Op: op}
		for _, s := range raw {
			if s == "" {
				continue // empty form field: no condition
			}
			if len(s) > MaxValueLength {
				return nil, fmt.Errorf("%s is limited to %d characters", name, MaxValueLength)
			}
			v, err := coerce(prop, kind, op, s)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			f.Values = append(f.Values, v)
		}
		if len(f.Values) > 0 {
			out = append(out, f)
		}
	}
	return out, nil
}

// splitParam splits "year.gte" into ("year", OpGte); a suffix that is not an
// operator is kept in the key (and then rejected as unknown attribute).
func splitParam(s string) (string, Op) {
	if i := strings.LastIndexByte(s, '.'); i > 0 {
		switch op := Op(s[i+1:]); op {
		case OpEq, OpGte, OpLte, OpContains:
			return s[:i], op
		}
	}
	return s, OpEq
}

func acceptsOp(k Kind, op Op) bool {
	for _, o := range k.Ops() {
		if o == op {
			return true
		}
	}
	return false
}

// coerce converts a query string value to the JSON type of the attribute.
func coerce(p Property, kind Kind, op Op, s string) (any, error) {
	scalar := p.Type
	if kind == KindArray {
		scalar = p.Items.Type
	}
	if op == OpContains {
		return s, nil
	}
	switch scalar {
	case "integer":
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, errors.New("integer expected")
		}
		return n, nil
	case "number":
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, errors.New("number expected")
		}
		return f, nil
	case "boolean":
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, errors.New("true or false expected")
		}
		return b, nil
	}
	return s, nil
}

// ParseSort validates `sort=[-]attr.<key>`; ok is false when sort does not
// target an attribute (created_at… are handled by the caller).
func ParseSort(raw string, props map[string]Property) (s Sort, ok bool, err error) {
	desc := strings.HasPrefix(raw, "-")
	field := strings.TrimPrefix(raw, "-")
	if !strings.HasPrefix(field, Prefix) {
		return Sort{}, false, nil
	}
	key := strings.TrimPrefix(field, Prefix)
	prop, found := props[key]
	if !found {
		return Sort{}, true, fmt.Errorf("unknown attribute %q for this item type", key)
	}
	kind := prop.Kind()
	if !kind.Sortable() {
		return Sort{}, true, fmt.Errorf("attribute %q cannot be sorted", key)
	}
	numeric := kind == KindInteger || kind == KindNumber
	return Sort{Key: key, Numeric: numeric, Desc: desc}, true, nil
}

// TSQuery turns free text into a prefix tsquery ("Breaking ba" →
// "breaking:* & ba:*"). Only letters and digits survive, so the result never
// carries tsquery syntax; "" means no usable term.
func TSQuery(text string) string {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	terms := make([]string, 0, len(words))
	for _, w := range words {
		terms = append(terms, w+":*")
	}
	return strings.Join(terms, " & ")
}
