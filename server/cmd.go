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
	"github.com/shinsaber/colec/internal/domains"
	"github.com/shinsaber/colec/internal/model"
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

// applySeed installs every domain package (the seeds only hold demonstration data, the
// types come from the packages), then applies SEED_FILE (comma-separated list) or the
// default seeds, in order.
func applySeed(ctx context.Context, databaseURL string) int {
	if code := installDomains(ctx, databaseURL, nil, false); code != 0 {
		return code
	}
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

// runCreateUser implements `colec-server create-user <username> <email> [--admin|--contributor]`:
// the first account of a production instance (seed disabled), or any later one.
// The password is typed twice without echo on a terminal, otherwise read from the
// first line of standard input (scripts: `echo … | colec-server create-user …`).
func runCreateUser(args []string) int {
	role := model.RoleUser
	var pos []string
	for _, a := range args {
		switch a {
		case "--admin":
			role = model.RoleAdmin
		case "--contributor":
			role = model.RoleContributor
		default:
			pos = append(pos, a)
		}
	}
	if len(pos) != 2 || pos[0] == "" || !strings.Contains(pos[1], "@") {
		fmt.Fprintln(os.Stderr, "usage: colec-server create-user <username> <email> [--admin|--contributor]")
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
	if len([]rune(password)) < model.MinPasswordLength {
		return "", fmt.Errorf("password too short (%d characters minimum)", model.MinPasswordLength)
	}
	return password, nil
}

// runInstallDomain implements `colec-server install-domain [--dry-run] [<domaine>…|--all]`:
// without argument, lists the packages and their state; otherwise installs or
// updates them (merge: the types changed on the instance keep their changes).
func runInstallDomain(args []string) int {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		return 1
	}
	dryRun := false
	var ids []string
	all := false
	for _, a := range args {
		switch a {
		case "--dry-run":
			dryRun = true
		case "--all":
			all = true
		default:
			ids = append(ids, a)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbCommandTimeout)
	defer cancel()
	if !all && len(ids) == 0 {
		pg, err := store.NewPgStore(ctx, databaseURL)
		if err != nil {
			slog.Error("database", "err", err)
			return 1
		}
		defer pg.Close()
		list, err := pg.ListDomains(ctx)
		if err != nil {
			slog.Error("domains", "err", err)
			return 1
		}
		for _, d := range list {
			installed := "-"
			if d.InstalledVersion != nil {
				installed = strconv.Itoa(*d.InstalledVersion)
			}
			local := ""
			if d.ModifiedLocally {
				local = " (modified locally)"
			}
			if len(d.Requires) > 0 {
				local += " — requires " + strings.Join(d.Requires, ", ")
			}
			fmt.Printf("%-12s package v%d, installed v%s: %s%s\n", d.ID, d.PackageVersion, installed, d.Status, local)
		}
		fmt.Println("\nusage: colec-server install-domain [--dry-run] <domain>… | --all")
		return 0
	}
	return installDomains(ctx, databaseURL, ids, dryRun)
}

// installDomains installs the given packages (all when ids is nil) and prints their report.
func installDomains(ctx context.Context, databaseURL string, ids []string, dryRun bool) int {
	pg, err := store.NewPgStore(ctx, databaseURL)
	if err != nil {
		slog.Error("database", "err", err)
		return 1
	}
	defer pg.Close()
	pkgs, err := domains.All()
	if err != nil {
		slog.Error("domains", "err", err)
		return 1
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	for _, p := range pkgs {
		if ids != nil && !wanted[p.ID] {
			continue
		}
		delete(wanted, p.ID)
		rep, err := pg.InstallDomain(ctx, p, dryRun)
		if err != nil {
			slog.Error("install domain", "domain", p.ID, "err", err)
			return 1
		}
		if !logDomainReport(rep) {
			return 1
		}
	}
	for id := range wanted {
		slog.Error("unknown domain", "domain", id)
		return 2
	}
	if !dryRun {
		// Relations d'autres domaines laissées de côté faute de types : ajoutées maintenant.
		if err := pg.RefreshInstalledDomains(ctx, ""); err != nil {
			slog.Error("refresh domains", "err", err)
			return 1
		}
	}
	return 0
}

// logDomainReport logs an install report, after those of the packages it
// required; false when one of them reports an error.
func logDomainReport(rep *model.DomainInstallReport) bool {
	ok := true
	for _, r := range rep.Required {
		ok = logDomainReport(r) && ok
	}
	counts := map[string]int{}
	for _, e := range rep.Entries {
		counts[e.Outcome]++
		if len(e.Conflicts) > 0 {
			slog.Warn("local value kept", "domain", rep.Domain, "type", e.Name, "paths", e.Conflicts)
		}
	}
	for _, e := range rep.Errors {
		slog.Error("install domain", "domain", rep.Domain, "entity", e.Entity, "err", e.Message)
	}
	slog.Info("domain", "domain", rep.Domain, "version", rep.ToVersion, "dry_run", rep.DryRun,
		"applied", rep.Applied, "outcomes", counts)
	return ok && len(rep.Errors) == 0
}
