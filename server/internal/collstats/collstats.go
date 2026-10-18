// Package collstats derives the statistics of a collection from the JSON
// Schema of its CollectionType: numeric attributes are summed, enum and boolean
// ones counted per value, date ones counted per month. No attribute key is
// hard-coded: everything follows the schema (P4-1).
package collstats

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/shinsaber/colec/internal/model"
)

// Entry is one collection item, as far as statistics are concerned.
type Entry struct {
	Attributes json.RawMessage
	AddedAt    time.Time
}

// Result holds the schema-driven part of model.CollectionStats.
type Result struct {
	Numbers       []model.NumberStat
	Distributions []model.Distribution
	Timelines     []model.Timeline
	Added         []model.MonthCount
}

type kind int

const (
	kindNone kind = iota
	kindNumber
	kindEnum
	kindDate
)

type field struct {
	key  string
	kind kind
	enum []string // kindEnum: declared values, in order
}

type property struct {
	Type   any    `json:"type"`
	Format string `json:"format"`
	Enum   []any  `json:"enum"`
}

// maxFilledMonths bounds the gap filling of a timeline (50 years): beyond, only
// the months that carry values are returned.
const maxFilledMonths = 600

var monthRE = regexp.MustCompile(`^(\d{4})-(\d{2})`)

// Aggregate computes the statistics of entries against schema.
func Aggregate(schema json.RawMessage, entries []Entry) Result {
	fields := fieldsOf(schema)
	values := make([]map[string]any, len(entries))
	for i, e := range entries {
		_ = json.Unmarshal(e.Attributes, &values[i]) // invalid attributes: no value
	}

	res := Result{
		Numbers:       []model.NumberStat{},
		Distributions: []model.Distribution{},
		Timelines:     []model.Timeline{},
	}
	for _, f := range fields {
		switch f.kind {
		case kindNumber:
			res.Numbers = append(res.Numbers, numberStat(f.key, values))
		case kindEnum:
			res.Distributions = append(res.Distributions, distribution(f, values))
		case kindDate:
			counts := map[string]int{}
			for _, v := range values {
				if s, ok := v[f.key].(string); ok {
					if m := monthRE.FindStringSubmatch(s); m != nil {
						counts[m[1]+"-"+m[2]]++
					}
				}
			}
			res.Timelines = append(res.Timelines, model.Timeline{Field: f.key, Months: Months(counts)})
		}
	}
	added := map[string]int{}
	for _, e := range entries {
		if !e.AddedAt.IsZero() {
			added[e.AddedAt.UTC().Format("2006-01")]++
		}
	}
	res.Added = Months(added)
	return res
}

func numberStat(key string, values []map[string]any) model.NumberStat {
	st := model.NumberStat{Field: key}
	for _, v := range values {
		n, ok := v[key].(float64)
		if !ok {
			continue
		}
		if st.Count == 0 || n < st.Min {
			st.Min = n
		}
		if st.Count == 0 || n > st.Max {
			st.Max = n
		}
		st.Sum += n
		st.Count++
	}
	if st.Count > 0 {
		st.Avg = st.Sum / float64(st.Count)
	}
	return st
}

func distribution(f field, values []map[string]any) model.Distribution {
	d := model.Distribution{Field: f.key, Values: []model.ValueCount{}}
	counts := map[string]int{}
	for _, v := range values {
		s, ok := scalar(v[f.key])
		if !ok {
			d.Missing++
			continue
		}
		counts[s]++
	}
	declared := map[string]bool{}
	for _, e := range f.enum {
		declared[e] = true
		d.Values = append(d.Values, model.ValueCount{Value: e, Count: counts[e]})
	}
	var extra []model.ValueCount
	for v, c := range counts {
		if !declared[v] {
			extra = append(extra, model.ValueCount{Value: v, Count: c})
		}
	}
	sort.Slice(extra, func(i, j int) bool {
		if extra[i].Count != extra[j].Count {
			return extra[i].Count > extra[j].Count
		}
		return extra[i].Value < extra[j].Value
	})
	d.Values = append(d.Values, extra...)
	return d
}

// scalar renders an enum or boolean value; false for a missing or non-scalar one.
func scalar(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, x != ""
	case bool:
		return strconv.FormatBool(x), true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	}
	return "", false
}

// Months turns per-month counts into a sorted timeline, months without values
// included between the first and the last one.
func Months(counts map[string]int) []model.MonthCount {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []model.MonthCount{}
	if len(keys) == 0 {
		return out
	}
	first, err1 := time.Parse("2006-01", keys[0])
	last, err2 := time.Parse("2006-01", keys[len(keys)-1])
	span := (last.Year()-first.Year())*12 + int(last.Month()-first.Month())
	if err1 != nil || err2 != nil || span > maxFilledMonths {
		for _, k := range keys {
			out = append(out, model.MonthCount{Month: k, Count: counts[k]})
		}
		return out
	}
	for m := first; !m.After(last); m = m.AddDate(0, 1, 0) {
		k := m.Format("2006-01")
		out = append(out, model.MonthCount{Month: k, Count: counts[k]})
	}
	return out
}

// fieldsOf lists the properties of schema that yield a statistic, in x-order
// then key order.
func fieldsOf(schema json.RawMessage) []field {
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
	var rest []string
	for k := range root.Properties {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	keys = append(keys, rest...)

	var out []field
	for _, k := range keys {
		if f := classify(k, root.Properties[k]); f.kind != kindNone {
			out = append(out, f)
		}
	}
	return out
}

func classify(key string, p property) field {
	types := map[string]bool{}
	switch t := p.Type.(type) {
	case string:
		types[t] = true
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok {
				types[s] = true
			}
		}
	}
	f := field{key: key}
	switch {
	case len(p.Enum) > 0:
		f.kind = kindEnum
		for _, e := range p.Enum {
			if s, ok := scalar(e); ok {
				f.enum = append(f.enum, s)
			}
		}
	case types["boolean"]:
		f.kind = kindEnum
		f.enum = []string{"true", "false"}
	case types["number"] || types["integer"]:
		f.kind = kindNumber
	case types["string"] && (p.Format == "date" || p.Format == "date-time"):
		f.kind = kindDate
	}
	return f
}
