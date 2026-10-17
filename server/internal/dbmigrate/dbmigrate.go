// Package dbmigrate applies the SQL migrations of db/migrations (tern format,
// numbered 001_xxx.sql) and, optionally, the development seed. It is driven by
// the `colec-server migrate` and `colec-server seed` subcommands.
package dbmigrate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
)

// VersionTable is the table tern uses to record the applied migration number.
const VersionTable = "schema_version"

// Default locations tried, in order, when MIGRATIONS_DIR / SEED_FILE are unset:
// from the repository root and from server/ (`go run . migrate`).
var (
	defaultMigrationDirs = []string{"db/migrations", "../db/migrations", "/app/db/migrations"}
	defaultSeedDirs      = []string{"db", "../db", "/app/db"}
	// defaultSeedNames are applied in this order; seed.sql is required, the
	// domain seeds that follow (they reuse its users and types) are optional.
	defaultSeedNames = []string{"seed.sql", "seed_bd_manga.sql", "seed_music.sql"}
)

// ResolveMigrationsDir returns the migrations directory: the explicit value if
// non-empty, otherwise the first default candidate that exists.
func ResolveMigrationsDir(explicit string) (string, error) {
	return resolve(explicit, defaultMigrationDirs, "MIGRATIONS_DIR", true)
}

// ResolveSeedFiles returns the seed files to apply, in order: the explicit
// comma-separated list if non-empty (each file must exist), otherwise
// seed.sql then the optional domain seeds of the first default directory that
// holds a seed.sql.
func ResolveSeedFiles(explicit string) ([]string, error) {
	if explicit != "" {
		var files []string
		for _, f := range strings.Split(explicit, ",") {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if err := checkPath(f, false); err != nil {
				return nil, fmt.Errorf("SEED_FILE: %w", err)
			}
			files = append(files, f)
		}
		return files, nil
	}
	for _, dir := range defaultSeedDirs {
		if checkPath(filepath.Join(dir, defaultSeedNames[0]), false) != nil {
			continue
		}
		var files []string
		for _, name := range defaultSeedNames {
			if f := filepath.Join(dir, name); checkPath(f, false) == nil {
				files = append(files, f)
			}
		}
		return files, nil
	}
	return nil, fmt.Errorf("SEED_FILE unset and no seed.sql found in %v", defaultSeedDirs)
}

func resolve(explicit string, candidates []string, envName string, wantDir bool) (string, error) {
	if explicit != "" {
		if err := checkPath(explicit, wantDir); err != nil {
			return "", fmt.Errorf("%s: %w", envName, err)
		}
		return explicit, nil
	}
	for _, c := range candidates {
		if checkPath(c, wantDir) == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("%s not set and none of %v exists", envName, candidates)
}

func checkPath(p string, wantDir bool) error {
	info, err := os.Stat(p)
	if err != nil {
		return err
	}
	if info.IsDir() != wantDir {
		if wantDir {
			return fmt.Errorf("%s is not a directory", p)
		}
		return fmt.Errorf("%s is a directory", p)
	}
	return nil
}

// Migrate applies all pending migrations found in dir. It is safe to run
// concurrently from several instances: tern takes a PostgreSQL advisory lock.
func Migrate(ctx context.Context, databaseURL, dir string) error {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	m, err := migrate.NewMigrator(ctx, conn, VersionTable)
	if err != nil {
		return fmt.Errorf("migrator: %w", err)
	}
	if err := m.LoadMigrations(os.DirFS(dir)); err != nil {
		return fmt.Errorf("load migrations from %s: %w", dir, err)
	}
	before, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("current version: %w", err)
	}
	if err := m.Migrate(ctx); err != nil {
		var pgErr migrate.MigrationPgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("migration %s failed: %w", pgErr.MigrationName, err)
		}
		return err
	}
	after, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("current version: %w", err)
	}
	slog.Info("migrations applied", "dir", dir, "from", before, "to", after, "files", len(m.Migrations))
	return nil
}

// Seed executes the given SQL file as one script (simple protocol, several statements).
func Seed(ctx context.Context, databaseURL, file string) error {
	sql, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return fmt.Errorf("read seed: %w", err)
	}
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	if _, err := conn.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("apply seed %s: %w", file, err)
	}
	slog.Info("seed applied", "file", file)
	return nil
}

// UsersEmpty reports whether the users table has no row (fresh database).
func UsersEmpty(ctx context.Context, databaseURL string) (bool, error) {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return false, fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	var exists bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users)`).Scan(&exists); err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return !exists, nil
}
