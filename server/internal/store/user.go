package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// GetUserByUsername ignores case (migration 004: unique index on lower(username)).
func (s *PgStore) GetUserByUsername(ctx context.Context, username string) (*model.User, string, error) {
	var u model.User
	var passwordHash string
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, email, role, password, created_at, updated_at
		 FROM users WHERE lower(username) = lower($1)`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &passwordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	return &u, passwordHash, nil
}

func (s *PgStore) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, email, role, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateUser adds an account (command `colec-server create-user`). A username
// already taken, whatever its case (migration 004), or an email in use gives ErrConflict.
func (s *PgStore) CreateUser(ctx context.Context, username, email, passwordHash, role string) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, $4)
		 RETURNING id, username, email, role, created_at, updated_at`,
		username, email, passwordHash, role,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &u, nil
}
