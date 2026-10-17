package store

import "context"

// ReferencedUploads returns the file names of /api/v1/uploads/ URLs found in
// any string attribute, at any depth, of items, relations and collection items,
// and the avatars of the users.
func (s *PgStore) ReferencedUploads(ctx context.Context) ([]string, error) {
	const path = `strict $.** ? (@.type() == "string" && @ like_regex "/api/v1/uploads/")`
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT regexp_replace(v #>> '{}', '^.*/api/v1/uploads/', '')
		FROM (
			SELECT jsonb_path_query(attributes, $1::jsonpath) AS v FROM items
			UNION ALL
			SELECT jsonb_path_query(attributes, $1::jsonpath) FROM item_relations
			UNION ALL
			SELECT jsonb_path_query(attributes, $1::jsonpath) FROM collection_items
			UNION ALL
			SELECT to_jsonb(avatar) FROM users WHERE avatar LIKE '%/api/v1/uploads/%'
		) refs`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}
