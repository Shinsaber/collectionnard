package store

import "testing"

func TestEscapeLike(t *testing.T) {
	cases := map[string]string{
		"breaking":   "breaking",
		"100%":       `100\%`,
		"a_b":        `a\_b`,
		`back\slash`: `back\\slash`,
		"%_%":        `\%\_\%`,
	}
	for in, want := range cases {
		if got := escapeLike(in); got != want {
			t.Errorf("escapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildSortClause(t *testing.T) {
	allowed := map[string]string{"created_at": "i.created_at"}
	if got := buildSortClause("", "i.created_at DESC", allowed); got != "i.created_at DESC" {
		t.Errorf("default: %q", got)
	}
	if got := buildSortClause("created_at", "x", allowed); got != "i.created_at ASC" {
		t.Errorf("asc: %q", got)
	}
	if got := buildSortClause("-created_at", "x", allowed); got != "i.created_at DESC" {
		t.Errorf("desc: %q", got)
	}
	if got := buildSortClause("attributes->>'x'; DROP TABLE items", "x", allowed); got != "x" {
		t.Errorf("unknown field must fall back to default, got %q", got)
	}
}
