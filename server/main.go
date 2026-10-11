package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/shinsaber/colec/config"
	"github.com/shinsaber/colec/internal/handler"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/store"
)

// refreshTokenPurgeInterval is how often expired refresh tokens are deleted.
const refreshTokenPurgeInterval = time.Hour

const usage = `usage: colec-server [serve|migrate|seed]

  serve    (default) start the HTTP API
  migrate  apply pending SQL migrations (MIGRATIONS_DIR), then the seed if
           SEED_ON_EMPTY=true and the users table is empty (SEED_FILE)
  seed     apply the seed file unconditionally (development only)
`

func main() {
	setupLogger()

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "serve":
		serve()
	case "migrate":
		os.Exit(runMigrate())
	case "seed":
		os.Exit(runSeed())
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", cmd, usage)
		os.Exit(2)
	}
}

// setupLogger configures slog from LOG_FORMAT (json|text, default json) and
// LOG_LEVEL (debug|info|warn|error, default info).
func setupLogger() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if strings.EqualFold(os.Getenv("LOG_FORMAT"), "text") {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}

func serve() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	pgStore, err := store.NewPgStore(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer pgStore.Close()

	// Auth
	auth := middleware.NewAuth(cfg.JWTSecret, cfg.AccessTokenTTL)

	// Handler + Router
	h := handler.New(pgStore, auth, cfg)
	r := h.Router()

	// Server
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Periodic purge of expired refresh tokens
	go purgeRefreshTokens(ctx, pgStore)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()

		slog.Info("shutting down")
		cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "err", err)
		}
	}()

	slog.Info("Collectionnard API listening",
		"port", cfg.Port,
		"access_token_ttl", cfg.AccessTokenTTL.String(),
		"refresh_token_ttl", cfg.RefreshTokenTTL.String(),
		"trusted_proxy", cfg.TrustedProxy,
	)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

func purgeRefreshTokens(ctx context.Context, s *store.PgStore) {
	ticker := time.NewTicker(refreshTokenPurgeInterval)
	defer ticker.Stop()
	for {
		if n, err := s.DeleteExpiredRefreshTokens(ctx); err != nil {
			if ctx.Err() == nil {
				slog.Error("refresh token purge", "err", err)
			}
		} else if n > 0 {
			slog.Info("refresh token purge", "removed", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
