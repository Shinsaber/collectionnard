package domains

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
)

// Merge3 merges a package update into a meta-type the instance may have changed.
// `base` is the package version that was installed (nil when unknown: a type
// created by a seed or by hand with the same UUID), `local` the current one,
// `next` the new package version. Values are decoded JSON (maps, slices,
// float64, string, bool, nil).
//
//   - local unchanged since base → next;
//   - package unchanged since base → local;
//   - objects: merged key by key; a key added by the package is added, a key
//     removed by the package is kept, a key removed locally stays removed;
//   - arrays of scalars (enum, required, x-order, allowed type ids): local
//     values in their order, then the values the package added (never the
//     ones removed locally);
//   - anything else changed on both sides: local value kept, path reported.
func Merge3(base, local, next any) (any, []string) {
	var conflicts []string
	out := merge3(base, local, next, "", &conflicts)
	sort.Strings(conflicts)
	return out, conflicts
}

func merge3(b, l, n any, path string, conflicts *[]string) any {
	switch {
	case equal(l, b):
		return n
	case equal(n, b), equal(l, n):
		return l
	}
	lm, lok := l.(map[string]any)
	nm, nok := n.(map[string]any)
	if lok && nok {
		bm, _ := b.(map[string]any)
		out := make(map[string]any, len(lm)+len(nm))
		for k, lv := range lm {
			nv, inNext := nm[k]
			bv, inBase := bm[k]
			if !inNext {
				out[k] = lv // supprimé (ou jamais présent) dans le paquet : on garde
				continue
			}
			if !inBase {
				bv = nil
			}
			out[k] = merge3(bv, lv, nv, join(path, k), conflicts)
		}
		for k, nv := range nm {
			if _, inLocal := lm[k]; inLocal {
				continue
			}
			bv, inBase := bm[k]
			switch {
			case !inBase:
				out[k] = nv // ajouté par le paquet
			case !equal(nv, bv):
				*conflicts = append(*conflicts, join(path, k)) // supprimé ici, modifié dans le paquet
			}
		}
		return out
	}
	if la, ok := scalars(l); ok {
		if na, ok := scalars(n); ok {
			ba, _ := scalars(b)
			out := append([]any{}, la...)
			for _, v := range na {
				if !contains(la, v) && !contains(ba, v) {
					out = append(out, v)
				}
			}
			return out
		}
	}
	*conflicts = append(*conflicts, path)
	return l
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func equal(a, b any) bool { return reflect.DeepEqual(a, b) }

// scalars returns v as a slice when it is an array of strings, numbers or booleans.
func scalars(v any) ([]any, bool) {
	a, ok := v.([]any)
	if !ok {
		return nil, false
	}
	for _, e := range a {
		switch e.(type) {
		case string, float64, bool:
		default:
			return nil, false
		}
	}
	return a, true
}

func contains(a []any, v any) bool {
	for _, e := range a {
		if equal(e, v) {
			return true
		}
	}
	return false
}

// Snapshot is a meta-type as decoded JSON, without its id: the unit compared
// and merged (name, description, attributes_schema, display_order, junctions).
type Snapshot = map[string]any

// SnapshotOf turns an export struct (ExportItemType…) or a JSON document into a Snapshot.
func SnapshotOf(v any) (Snapshot, error) {
	raw, ok := v.(json.RawMessage)
	if !ok {
		var err error
		if raw, err = json.Marshal(v); err != nil {
			return nil, err
		}
	}
	var s Snapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	delete(s, "id")
	return s, nil
}

// junctionKeys are the snapshot keys holding item type ids.
var junctionKeys = []string{"item_type_ids", "source_item_type_ids", "target_item_type_ids"}

// keepKnown drops from the junction lists the item type ids absent from known,
// and returns them (sorted).
func keepKnown(s Snapshot, known map[string]bool) []string {
	var missing []string
	for _, k := range junctionKeys {
		ids, ok := s[k].([]any)
		if !ok {
			continue
		}
		kept := make([]any, 0, len(ids))
		for _, id := range ids {
			if str, _ := id.(string); known[strings.ToLower(str)] {
				kept = append(kept, id)
			} else {
				missing = append(missing, str)
			}
		}
		s[k] = kept
	}
	sort.Strings(missing)
	return missing
}
