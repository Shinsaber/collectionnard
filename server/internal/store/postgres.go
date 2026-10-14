package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore implements Store backed by PostgreSQL.
type PgStore struct {
	pool *pgxpool.Pool
}

// statementTimeout bounds every statement server-side so that a pathological
// query cannot hold a connection indefinitely.
const statementTimeout = "10000" // ms

func NewPgStore(ctx context.Context, databaseURL string) (*PgStore, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if _, set := cfg.ConnConfig.RuntimeParams["statement_timeout"]; !set {
		cfg.ConnConfig.RuntimeParams["statement_timeout"] = statementTimeout
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &PgStore{pool: pool}, nil
}

func (s *PgStore) Close() {
	s.pool.Close()
}

// Ping checks the database connection (used by GET /healthz).
func (s *PgStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
