package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shinsaber/colec/internal/model"
)

// --- Liens de partage (P4-2) : seul le hash SHA-256 du jeton est stocké. ---

func (s *PgStore) CreateShareLink(ctx context.Context, collectionID uuid.UUID, tokenHash string) (*model.ShareLink, error) {
	var l model.ShareLink
	err := s.pool.QueryRow(ctx,
		`INSERT INTO share_links (collection_id, token_hash) VALUES ($1, $2)
		 RETURNING id, collection_id, created_at`, collectionID, tokenHash).Scan(&l.ID, &l.CollectionID, &l.CreatedAt)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

// ListShareLinks lists the active (not revoked) links of a collection, newest first.
func (s *PgStore) ListShareLinks(ctx context.Context, collectionID uuid.UUID) ([]model.ShareLink, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, collection_id, created_at FROM share_links
		 WHERE collection_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC, id`, collectionID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.ShareLink, error) {
		var l model.ShareLink
		err := row.Scan(&l.ID, &l.CollectionID, &l.CreatedAt)
		return l, err
	})
	if out == nil {
		out = []model.ShareLink{}
	}
	return out, err
}

// RevokeShareLink revokes an active link of the collection (ErrNotFound otherwise).
func (s *PgStore) RevokeShareLink(ctx context.Context, collectionID, linkID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE share_links SET revoked_at = now()
		 WHERE id = $1 AND collection_id = $2 AND revoked_at IS NULL`, linkID, collectionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetSharedCollectionID resolves an active link to its collection.
func (s *PgStore) GetSharedCollectionID(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT collection_id FROM share_links WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}

// ListMembers is the members directory: active accounts whose profile is
// visible to members or public.
func (s *PgStore) ListMembers(ctx context.Context, pg model.Pagination) ([]model.User, int, error) {
	const where = ` FROM users WHERE disabled_at IS NULL AND profile_visibility IN ('members', 'public')`
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*)`+where).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+userColumns+where+` ORDER BY lower(username) LIMIT $1 OFFSET $2`,
		pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *u)
	}
	return out, total, rows.Err()
}
