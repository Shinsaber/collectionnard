package dbmigrate

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestResolveMigrationsDir_Explicit(t *testing.T) {
	dir := t.TempDir()
	got, err := ResolveMigrationsDir(dir)
	if err != nil || got != dir {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := ResolveMigrationsDir(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected error for missing explicit dir")
	}
	file := filepath.Join(dir, "f.sql")
	_ = os.WriteFile(file, []byte("select 1"), 0o600)
	if _, err := ResolveMigrationsDir(file); err == nil {
		t.Fatal("expected error when MIGRATIONS_DIR is a file")
	}
}

func TestResolveMigrationsDir_Default(t *testing.T) {
	// From server/internal/dbmigrate the repository's ../db/migrations is not a
	// candidate; chdir to server/ where `go run . migrate` is documented to run.
	wd, _ := os.Getwd()
	t.Chdir(filepath.Join(wd, "..", ".."))
	got, err := ResolveMigrationsDir("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "../db/migrations" {
		t.Errorf("got %q, want ../db/migrations", got)
	}
	seeds, err := ResolveSeedFiles("")
	if err != nil || !slices.Equal(seeds, []string{"../db/seed.sql", "../db/seed_bd_manga.sql", "../db/seed_music.sql"}) {
		t.Errorf("seeds: got %q, %v", seeds, err)
	}
	// Explicit comma-separated list, spaces and empty entries ignored.
	seeds, err = ResolveSeedFiles(" ../db/seed_bd_manga.sql, ,../db/seed.sql")
	if err != nil || !slices.Equal(seeds, []string{"../db/seed_bd_manga.sql", "../db/seed.sql"}) {
		t.Errorf("explicit seeds: got %q, %v", seeds, err)
	}
	if _, err := ResolveSeedFiles("../db/seed.sql,../db/nope.sql"); err == nil {
		t.Error("a missing explicit seed must fail")
	}
}
