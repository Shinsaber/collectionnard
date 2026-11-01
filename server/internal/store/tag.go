package store

import (
	"context"
	"strconv"

	"github.com/shinsaber/colec/internal/model"
)

// maxTags bounds GET /tags: the list feeds an autocomplete, which narrows it
// with `search`; without a bound it would grow with every imported catalogue
// (audit Q-006).
const maxTags = 100

func (s *PgStore) ListTags(ctx context.Context, search string) ([]model.Tag, error) {
	query := `SELECT id, name FROM tags`
	args := []any{}

	if search != "" {
		query += ` WHERE name ILIKE $1 ESCAPE '\'`
		args = append(args, escapeLike(search)+"%")
	}
	query += ` ORDER BY name LIMIT ` + strconv.Itoa(maxTags)

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
