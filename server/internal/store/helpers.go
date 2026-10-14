package store

import (
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505"
	}
	return false
}

func isForeignKeyViolation(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23503"
	}
	return false
}

func nullableJSON(data json.RawMessage) any {
	if len(data) == 0 {
		return nil
	}
	return data
}

func buildSortClause(sort, defaultSort string, allowed map[string]string) string {
	if sort == "" {
		return defaultSort
	}
	desc := false
	field := sort
	if strings.HasPrefix(sort, "-") {
		desc = true
		field = sort[1:]
	}
	col, ok := allowed[field]
	if !ok {
		return defaultSort
	}
	if desc {
		return col + " DESC"
	}
	return col + " ASC"
}

// escapeLike neutralises the LIKE/ILIKE metacharacters of a user-supplied
// string so that it is matched literally (to be used with ESCAPE '\').
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
