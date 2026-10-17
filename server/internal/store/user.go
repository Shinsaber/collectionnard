package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// userColumns are read by every user query (scanUser), in this order.
const userColumns = `id, username, email, role, display_name, bio, avatar, disabled_at, created_at, updated_at`

func scanUser(row pgx.Row, extra ...any) (*model.User, error) {
	var u model.User
	dest := append([]any{&u.ID, &u.Username, &u.Email, &u.Role, &u.DisplayName, &u.Bio, &u.Avatar,
		&u.DisabledAt, &u.CreatedAt, &u.UpdatedAt}, extra...)
	if err := row.Scan(dest...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// GetUserByUsername ignores case (migration 004: unique index on lower(username)).
func (s *PgStore) GetUserByUsername(ctx context.Context, username string) (*model.User, string, error) {
	var passwordHash string
	u, err := scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userColumns+`, password FROM users WHERE lower(username) = lower($1)`, username), &passwordHash)
	if err != nil {
		return nil, "", err
	}
	return u, passwordHash, nil
}

func (s *PgStore) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

// GetUserPasswordHash returns the password hash of a user (password change).
func (s *PgStore) GetUserPasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	var h string
	err := s.pool.QueryRow(ctx, `SELECT password FROM users WHERE id = $1`, id).Scan(&h)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return h, err
}

// CreateUser adds an account (command `colec-server create-user`). A username
// already taken, whatever its case (migration 004), or an email in use gives ErrConflict.
func (s *PgStore) CreateUser(ctx context.Context, username, email, passwordHash, role string) (*model.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, $4)
		 RETURNING `+userColumns, username, email, passwordHash, role))
	if err != nil && isUniqueViolation(err) {
		return nil, ErrConflict
	}
	return u, err
}
