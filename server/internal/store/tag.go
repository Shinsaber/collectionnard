package store

import (
	"context"

	"github.com/shinsaber/colec/internal/model"
)

func (s *PgStore) ListTags(ctx context.Context, search string) ([]model.Tag, error) {
	query := `SELECT id, name FROM tags`
	args := []any{}

	if search != "" {
		query += ` WHERE lower(name) LIKE lower($1)`
		args = append(args, search+"%")
	}
	query += ` ORDER BY name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
