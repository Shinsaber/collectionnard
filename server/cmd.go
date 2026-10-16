package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shinsaber/colec/internal/dbmigrate"
	"github.com/shinsaber/colec/internal/store"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
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

// applySeed applies SEED_FILE (comma-separated list) or the default seeds, in order.
func applySeed(ctx context.Context, databaseURL string) int {
	files, err := dbmigrate.ResolveSeedFiles(os.Getenv("SEED_FILE"))
	if err != nil {
		slog.Error("seed", "err", err)
		return 1
	}
	for _, file := range files {
		if err := dbmigrate.Seed(ctx, databaseURL, file); err != nil {
			slog.Error("seed", "err", err)
			return 1
		}
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

// minPasswordLength is the minimum length accepted by create-user.
const minPasswordLength = 8

// runCreateUser implements `colec-server create-user <username> <email> [--admin]`:
// the first account of a production instance (seed disabled), or any later one.
// The password is typed twice without echo on a terminal, otherwise read from the
// first line of standard input (scripts: `echo … | colec-server create-user …`).
func runCreateUser(args []string) int {
	role := "user"
	var pos []string
	for _, a := range args {
		if a == "--admin" {
			role = "admin"
		} else {
			pos = append(pos, a)
		}
	}
	if len(pos) != 2 || pos[0] == "" || !strings.Contains(pos[1], "@") {
		fmt.Fprintln(os.Stderr, "usage: colec-server create-user <username> <email> [--admin]")
		return 2
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		return 1
	}
	password, err := readNewPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("bcrypt", "err", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbCommandTimeout)
	defer cancel()
	pg, err := store.NewPgStore(ctx, databaseURL)
	if err != nil {
		slog.Error("database", "err", err)
		return 1
	}
	defer pg.Close()
	u, err := pg.CreateUser(ctx, pos[0], pos[1], string(hash), role)
	if errors.Is(err, store.ErrConflict) {
		fmt.Fprintf(os.Stderr, "username %q or email %q already in use\n", pos[0], pos[1])
		return 1
	}
	if err != nil {
		slog.Error("create user", "err", err)
		return 1
	}
	fmt.Printf("created %s %q (%s)\n", u.Role, u.Username, u.ID)
	return 0
}

func readNewPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	var password string
	if term.IsTerminal(fd) {
		ask := func(prompt string) (string, error) {
			fmt.Fprint(os.Stderr, prompt)
			b, err := term.ReadPassword(fd)
			fmt.Fprintln(os.Stderr)
			return string(b), err
		}
		p1, err := ask("Password: ")
		if err != nil {
			return "", err
		}
		p2, err := ask("Confirm password: ")
		if err != nil {
			return "", err
		}
		if p1 != p2 {
			return "", errors.New("passwords do not match")
		}
		password = p1
	} else {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		password = strings.TrimRight(line, "\r\n")
	}
	if len([]rune(password)) < minPasswordLength {
		return "", fmt.Errorf("password too short (%d characters minimum)", minPasswordLength)
	}
	return password, nil
}
