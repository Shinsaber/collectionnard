package connectors

import (
	"bytes"
	"encoding/json"
	"slices"
)

func parseObject(raw json.RawMessage) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return map[string]any{}
	}
	return m
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

// sameJSON compares two values once encoded (encoding/json sorts map keys).
func sameJSON(a, b any) bool { return bytes.Equal(mustJSON(a), mustJSON(b)) }

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case map[string]any:
		return len(x) == 0
	case []any:
		return len(x) == 0
	}
	return false
}

// fillEmpty completes existing with the values of source without overwriting
// anything: a missing or empty field takes the source value, objects (i18n
// texts, external_ids) are completed key by key, `sources` gains the entries
// of the sources it does not list yet.
func fillEmpty(existing, source map[string]any) map[string]any {
	out := make(map[string]any, len(existing)+len(source))
	for k, v := range existing {
		out[k] = v
	}
	for k, sv := range source {
		ev, ok := out[k]
		if !ok || isEmpty(ev) {
			out[k] = sv
			continue
		}
		if k == "sources" {
			out[k] = mergeSources(ev, sv)
			continue
		}
		em, eok := ev.(map[string]any)
		sm, sok := sv.(map[string]any)
		if eok && sok {
			out[k] = fillEmpty(em, sm)
		}
	}
	return out
}

func mergeSources(existing, source any) any {
	el, ok1 := existing.([]any)
	sl, ok2 := source.([]any)
	if !ok1 || !ok2 {
		return existing
	}
	seen := map[any]bool{}
	for _, e := range el {
		if m, ok := e.(map[string]any); ok {
			seen[m["source"]] = true
		}
	}
	out := slices.Clone(el)
	for _, s := range sl {
		if m, ok := s.(map[string]any); ok && !seen[m["source"]] {
			out = append(out, s)
			seen[m["source"]] = true
		}
	}
	return out
}

// fit drops the attributes the schema does not declare and the enum values
// it does not accept, so that a customised schema (field removed, enum value
// renamed) gets what it can take instead of failing validation.
func fit(schema, attrs map[string]any) map[string]any {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return attrs
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		p, declared := props[k].(map[string]any)
		if !declared {
			continue
		}
		if enum, ok := p["enum"].([]any); ok && !slices.Contains(enum, v) {
			continue
		}
		if items, ok := p["items"].(map[string]any); ok {
			if enum, ok := items["enum"].([]any); ok {
				if list, ok := v.([]any); ok {
					kept := []any{}
					for _, x := range list {
						if slices.Contains(enum, x) {
							kept = append(kept, x)
						}
					}
					if len(kept) == 0 {
						continue
					}
					v = kept
				}
			}
		}
		out[k] = v
	}
	return out
}
