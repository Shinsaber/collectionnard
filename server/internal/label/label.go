// Package label computes the human-readable label of an item from its
// ItemType schema (`x-label` extension) and its attributes. It is the Go
// counterpart of web/src/lib/schema.ts: both must stay in sync.
package label

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// placeholder matches `{key}` in an x-label template.
var placeholder = regexp.MustCompile(`\{([A-Za-z0-9_.-]+)\}`)

// FallbackKeys are tried, in order, when the schema carries no `x-label`.
// English first (project convention), then the historical French keys.
var FallbackKeys = []string{"title", "name", "label", "titre", "nom", "number", "numero"}

// Langs is the preference order used to pick a value in an i18n object ({fr, en});
// Langs[0] is the default language. It must stay aligned with web/src/i18n/langs.ts.
var Langs = []string{"fr", "en"}

// Template returns the `x-label` template of a schema, or "" when absent.
func Template(schema json.RawMessage) string {
	if len(schema) == 0 {
		return ""
	}
	var root struct {
		XLabel string `json:"x-label"`
	}
	if err := json.Unmarshal(schema, &root); err != nil {
		return ""
	}
	return strings.TrimSpace(root.XLabel)
}

// Render returns the label of an item: the `x-label` template of schema with
// its placeholders replaced by the attribute values (lang chooses the variant
// of i18n objects), or the first fallback key found, or "" when nothing
// usable exists (callers then fall back to the id).
func Render(schema, attrs json.RawMessage, lang string) string {
	var values map[string]any
	if err := json.Unmarshal(attrs, &values); err != nil || values == nil {
		return ""
	}
	if tpl := Template(schema); tpl != "" {
		out := placeholder.ReplaceAllStringFunc(tpl, func(m string) string {
			key := m[1 : len(m)-1]
			return Stringify(values[key], lang)
		})
		return strings.Join(strings.Fields(out), " ")
	}
	for _, key := range FallbackKeys {
		if v, ok := values[key]; ok {
			if s := Stringify(v, lang); s != "" {
				return s
			}
		}
	}
	return ""
}

// Stringify renders an attribute value for display: strings as is, numbers
// without trailing zeros, booleans, i18n objects by preferred language, arrays
// joined by ", ". An i18n object tries lang, then Langs, then its remaining
// keys sorted alphabetically.
func Stringify(v any, lang string) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	case []any:
		parts := make([]string, 0, len(t))
		for _, e := range t {
			if s := Stringify(e, lang); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		if s, ok := t[lang].(string); ok && s != "" {
			return s
		}
		for _, l := range Langs {
			if s, ok := t[l].(string); ok && s != "" {
				return s
			}
		}
		// Last resort: any other key, in alphabetical order (same order as the front).
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if s, ok := t[k].(string); ok && s != "" {
				return s
			}
		}
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}
