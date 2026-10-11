package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/shinsaber/colec/internal/dbmigrate"
	"golang.org/x/crypto/bcrypt"
)

const dbCommandTimeout = 5 * time.Minute

// runMigrate implements `colec-server migrate`. Only DATABASE_URL is required
// (JWT_SECRET is not needed to migrate), so config.Load is deliberately not used.
func runMigrate() int {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		return 1
	}
	dir, err := dbmigrate.ResolveMigrationsDir(os.Getenv("MIGRATIONS_DIR"))
	if err != nil {
		slog.Error("migrations", "err", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbCommandTimeout)
	defer cancel()

	if err := dbmigrate.Migrate(ctx, databaseURL, dir); err != nil {
		slog.Error("migrate", "err", err)
		return 1
	}

	seedOnEmpty, _ := strconv.ParseBool(os.Getenv("SEED_ON_EMPTY"))
	if !seedOnEmpty {
		return 0
	}
	empty, err := dbmigrate.UsersEmpty(ctx, databaseURL)
	if err != nil {
		slog.Error("seed", "err", err)
		return 1
	}
	if !empty {
		slog.Info("seed skipped: users table not empty")
		return 0
	}
	return applySeed(ctx, databaseURL)
}

// runSeed implements `colec-server seed` (unconditional, development only).
func runSeed() int {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbCommandTimeout)
	defer cancel()
	return applySeed(ctx, databaseURL)
}

func applySeed(ctx context.Context, databaseURL string) int {
	file, err := dbmigrate.ResolveSeedFile(os.Getenv("SEED_FILE"))
	if err != nil {
		slog.Error("seed", "err", err)
		return 1
	}
	if err := dbmigrate.Seed(ctx, databaseURL, file); err != nil {
		slog.Error("seed", "err", err)
		return 1
	}
	return 0
}

// runHashPassword implements `colec-server hash-password <mot de passe>` and
// prints a bcrypt hash usable in db/seed.sql or in a manual INSERT INTO users.
func runHashPassword(args []string) int {
	if len(args) != 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "usage: colec-server hash-password <password>")
		return 2
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(args[0]), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("bcrypt", "err", err)
		return 1
	}
	fmt.Println(string(hash))
	return 0
}
