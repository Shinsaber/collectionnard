package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shinsaber/colec/internal/itemquery"
)

// itemQuery accumulates the WHERE conditions and positional arguments of the
// item listing. Every user value goes through arg(): attribute keys included
// (`attributes -> $n::text`), so no user input is ever spliced into the SQL.
type itemQuery struct {
	where []string
	args  []any
	// tsquery is the placeholder of the full-text query, reused by the
	// relevance sort ("" without search).
	tsquery string
}

func (q *itemQuery) arg(v any) string {
	q.args = append(q.args, v)
	return fmt.Sprintf("$%d", len(q.args))
}

func (q *itemQuery) add(cond string) { q.where = append(q.where, cond) }

func (q *itemQuery) whereClause() string {
	if len(q.where) == 0 {
		return "TRUE"
	}
	return strings.Join(q.where, " AND ")
}

// buildItemQuery translates the filters into SQL conditions on `items i`.
func buildItemQuery(f ItemFilters) (*itemQuery, error) {
	q := &itemQuery{}
	if f.ItemTypeID != nil {
		q.add("i.item_type_id = " + q.arg(*f.ItemTypeID))
	}
	if ts := itemquery.TSQuery(f.Search); ts != "" {
		// f_unaccent / search_tsv: migration 003.
		q.tsquery = fmt.Sprintf("to_tsquery('simple', f_unaccent(%s))", q.arg(ts))
		q.add("i.search_tsv @@ " + q.tsquery)
	}
	if f.Tag != "" {
		// Case-insensitive exact match without lower() on the column (index-friendly, literal pattern).
		q.add(fmt.Sprintf(
			`EXISTS (SELECT 1 FROM item_tags it2 JOIN tags t ON t.id = it2.tag_id WHERE it2.item_id = i.id AND t.name ILIKE %s ESCAPE '\')`,
			q.arg(escapeLike(f.Tag))))
	}
	if f.RelatedTo != nil {
		other := q.arg(*f.RelatedTo)
		cond := fmt.Sprintf(`((r.source_item_id = %[1]s AND r.target_item_id = i.id) OR (r.target_item_id = %[1]s AND r.source_item_id = i.id))`, other)
		if f.RelationTypeID != nil {
			cond += " AND r.relation_type_id = " + q.arg(*f.RelationTypeID)
		}
		q.add("EXISTS (SELECT 1 FROM item_relations r WHERE " + cond + ")")
	}
	for _, af := range f.Attrs {
		cond, err := q.attrCondition(af)
		if err != nil {
			return nil, err
		}
		q.add(cond)
	}
	return q, nil
}

// attrCondition renders one attribute filter. Equality uses containment
// (`attributes @> {"key": v}`), served by the GIN index on attributes; ranges
// and text search read the value with a type guard so that a stored value of
// an unexpected type never makes the cast fail.
func (q *itemQuery) attrCondition(f itemquery.Filter) (string, error) {
	switch f.Op {
	case itemquery.OpEq:
		ors := make([]string, 0, len(f.Values))
		for _, v := range f.Values {
			val := v
			if f.Kind == itemquery.KindArray {
				val = []any{v} // array attribute: contains this element
			}
			doc, err := json.Marshal(map[string]any{f.Key: val})
			if err != nil {
				return "", err
			}
			ors = append(ors, fmt.Sprintf("i.attributes @> %s::jsonb", q.arg(string(doc))))
		}
		return "(" + strings.Join(ors, " OR ") + ")", nil

	case itemquery.OpGte, itemquery.OpLte:
		cmp := ">="
		if f.Op == itemquery.OpLte {
			cmp = "<="
		}
		key := q.arg(f.Key)
		if f.Kind == itemquery.KindDate {
			return fmt.Sprintf(`(CASE WHEN jsonb_typeof(i.attributes -> %[1]s::text) = 'string' THEN i.attributes ->> %[1]s::text END) %[2]s %[3]s::text`,
				key, cmp, q.arg(f.Values[0])), nil
		}
		return fmt.Sprintf(`%s %s %s::numeric`, numericAttr(key), cmp, q.arg(f.Values[0])), nil

	case itemquery.OpContains:
		key := q.arg(f.Key)
		pattern := q.arg("%" + escapeLike(fmt.Sprint(f.Values[0])) + "%")
		if f.Kind == itemquery.KindI18n {
			return fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_each_text(CASE WHEN jsonb_typeof(i.attributes -> %[1]s::text) = 'object' THEN i.attributes -> %[1]s::text ELSE '{}'::jsonb END) e WHERE f_unaccent(e.value) ILIKE f_unaccent(%[2]s) ESCAPE '\')`,
				key, pattern), nil
		}
		return fmt.Sprintf(`f_unaccent(i.attributes ->> %s::text) ILIKE f_unaccent(%s) ESCAPE '\'`, key, pattern), nil
	}
	return "", fmt.Errorf("unsupported operator %q", f.Op)
}

// numericAttr reads a numeric attribute, NULL when the stored value is not a number.
func numericAttr(key string) string {
	return fmt.Sprintf(`(CASE WHEN jsonb_typeof(i.attributes -> %[1]s::text) = 'number' THEN (i.attributes ->> %[1]s::text)::numeric END)`, key)
}

// orderBy returns the ORDER BY clause; it may append the attribute key to the
// arguments, so it must be called after the count query has captured its own.
func (q *itemQuery) orderBy(f ItemFilters) string {
	// i.id breaks ties: stable pagination on equal values.
	if f.SortAttr != nil {
		key := q.arg(f.SortAttr.Key)
		expr := fmt.Sprintf("(i.attributes ->> %s::text)", key)
		if f.SortAttr.Numeric {
			expr = numericAttr(key)
		}
		dir := "ASC"
		if f.SortAttr.Desc {
			dir = "DESC"
		}
		return fmt.Sprintf("%s %s NULLS LAST, i.id", expr, dir)
	}
	field := strings.TrimPrefix(f.Sort, "-")
	if (field == "relevance" || f.Sort == "") && q.tsquery != "" {
		dir := "DESC" // most relevant first; "relevance" ascending = least relevant first
		if f.Sort == "relevance" {
			dir = "ASC"
		}
		return fmt.Sprintf("ts_rank(i.search_tsv, %s) %s, i.created_at DESC, i.id", q.tsquery, dir)
	}
	return buildSortClause(f.Sort, "i.created_at DESC", map[string]string{
		"created_at": "i.created_at",
		"updated_at": "i.updated_at",
	}) + ", i.id"
}
