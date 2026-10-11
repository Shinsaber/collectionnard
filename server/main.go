package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shinsaber/colec/config"
	"github.com/shinsaber/colec/internal/handler"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/store"
)

// refreshTokenPurgeInterval is how often expired refresh tokens are deleted.
const refreshTokenPurgeInterval = time.Hour

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	pgStore, err := store.NewPgStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
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

		log.Println("Shutting down...")
		cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("Collectionnard API listening on :%d (access token TTL %s, refresh token TTL %s)",
		cfg.Port, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func purgeRefreshTokens(ctx context.Context, s *store.PgStore) {
	ticker := time.NewTicker(refreshTokenPurgeInterval)
	defer ticker.Stop()
	for {
		if n, err := s.DeleteExpiredRefreshTokens(ctx); err != nil {
			if ctx.Err() == nil {
				log.Printf("refresh token purge: %v", err)
			}
		} else if n > 0 {
			log.Printf("refresh token purge: %d expired token(s) removed", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
