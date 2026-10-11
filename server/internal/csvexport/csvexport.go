// Package csvexport flattens dynamic attributes into CSV columns derived from
// a JSON Schema: one column per property in `x-order`, one per language for an
// i18n-text object, arrays of scalars joined by "; ", other objects as JSON.
package csvexport

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/shinsaber/colec/internal/label"
)

// BOM makes spreadsheets open the file as UTF-8.
const BOM = "\xef\xbb\xbf"

// Delimiter maps the `delimiter` query parameter to a separator rune.
func Delimiter(name string) (rune, error) {
	switch name {
	case "", "comma":
		return ',', nil
	case "semicolon":
		return ';', nil
	case "tab":
		return '\t', nil
	}
	return 0, fmt.Errorf("invalid delimiter %q: accepted values are comma, semicolon, tab", name)
}

// Column is one attribute column: Key, plus Lang for an i18n-text object.
type Column struct {
	Key  string
	Lang string
}

// Header returns the column header: the key, or "key.lang".
func (c Column) Header() string {
	if c.Lang != "" {
		return c.Key + "." + c.Lang
	}
	return c.Key
}

type property struct {
	Widget     string                     `json:"x-widget"`
	Properties map[string]json.RawMessage `json:"properties"`
}

// Columns derives the attribute columns of a schema (x-order first, then the
// remaining properties sorted by key).
func Columns(schema json.RawMessage) []Column {
	var root struct {
		Order      []string            `json:"x-order"`
		Properties map[string]property `json:"properties"`
	}
	if len(schema) == 0 || json.Unmarshal(schema, &root) != nil {
		return nil
	}
	keys := make([]string, 0, len(root.Properties))
	seen := map[string]bool{}
	for _, k := range root.Order {
		if _, ok := root.Properties[k]; ok && !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	rest := make([]string, 0, len(root.Properties))
	for k := range root.Properties {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	keys = append(keys, rest...)

	var cols []Column
	for _, k := range keys {
		p := root.Properties[k]
		if p.Widget != "i18n-text" {
			cols = append(cols, Column{Key: k})
			continue
		}
		for _, l := range langsOf(p.Properties) {
			cols = append(cols, Column{Key: k, Lang: l})
		}
	}
	return cols
}

// langsOf orders the languages of an i18n-text object: label.Langs first,
// then the other declared codes sorted (at least the known languages).
func langsOf(props map[string]json.RawMessage) []string {
	out := append([]string{}, label.Langs...)
	var extra []string
	for l := range props {
		if !slices.Contains(label.Langs, l) {
			extra = append(extra, l)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

// Values renders the cells of attrs for cols.
func Values(cols []Column, attrs json.RawMessage) []string {
	var values map[string]any
	dec := json.NewDecoder(bytes.NewReader(attrs))
	dec.UseNumber()
	_ = dec.Decode(&values) // invalid or empty attributes: empty cells
	out := make([]string, len(cols))
	for i, c := range cols {
		v := values[c.Key]
		if c.Lang != "" {
			obj, _ := v.(map[string]any)
			v = obj[c.Lang]
		}
		out[i] = Cell(v)
	}
	return out
}

// Cell renders one value. Text that a spreadsheet would run as a formula
// (=, +, -, @, tab, CR) is prefixed with a quote (CSV injection).
func Cell(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return Text(t)
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case []any:
		parts := make([]string, 0, len(t))
		scalar := true
		for _, e := range t {
			switch e.(type) {
			case map[string]any, []any:
				scalar = false
			}
			parts = append(parts, Cell(e))
		}
		if scalar {
			return strings.Join(parts, "; ")
		}
	}
	b, _ := json.Marshal(v)
	return Text(string(b))
}

// Text neutralises a free-text cell (CSV injection, OWASP).
func Text(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// Writer writes a BOM-prefixed CSV.
func Writer(w io.Writer, delimiter rune) (*csv.Writer, error) {
	if _, err := io.WriteString(w, BOM); err != nil {
		return nil, err
	}
	cw := csv.NewWriter(w)
	cw.Comma = delimiter
	cw.UseCRLF = true
	return cw, nil
}
