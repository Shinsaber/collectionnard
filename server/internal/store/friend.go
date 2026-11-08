package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// Friends (P7-2): one row per pair of members, whatever the direction; a
// refusal, a cancellation or a break deletes it. Members of disabled accounts
// count nowhere.

// friendshipSelect: a friendship as seen by the member of parameter $1.
const friendshipSelect = `SELECT f.id, f.status, f.status = 'pending' AND f.addressee_id = $1,
	u.id, u.username, u.display_name, u.avatar, u.profile_visibility, f.created_at, f.accepted_at
	FROM friendships f JOIN users u ON u.disabled_at IS NULL
		AND u.id = CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END`

func scanFriendship(row pgx.Row) (*model.Friendship, error) {
	var f model.Friendship
	var a model.ReviewAuthor
	if err := row.Scan(&f.ID, &f.Status, &f.Incoming, &f.OtherID, &a.Username, &a.DisplayName, &a.Avatar,
		&f.OtherProfile, &f.CreatedAt, &f.AcceptedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	f.User = &a
	return &f, nil
}

// friendOf: the members of the columns (or parameters) a and b are friends.
func friendOf(a, b string) string {
	return `EXISTS (SELECT 1 FROM friendships fr WHERE fr.status = 'accepted'
		AND ((fr.requester_id = ` + a + ` AND fr.addressee_id = ` + b + `) OR (fr.requester_id = ` + b + ` AND fr.addressee_id = ` + a + `)))`
}

// FriendIDs lists the friends of userID (none for an anonymous reader).
func (s *PgStore) FriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if userID == uuid.Nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT CASE WHEN requester_id = $1 THEN addressee_id ELSE requester_id END
		FROM friendships WHERE status = 'accepted' AND (requester_id = $1 OR addressee_id = $1)`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}

// ListFriendships lists the friendships of userID with the given status:
// friends by name, or requests received then sent, newest first. A request of a
// member userID blocked stays hidden from them (silent refusal).
func (s *PgStore) ListFriendships(ctx context.Context, userID uuid.UUID, status string, pg model.Pagination) ([]model.Friendship, int, error) {
	where := ` WHERE f.status = $2 AND (f.requester_id = $1 OR f.addressee_id = $1)
		AND (f.requester_id = $1 OR ` + notBlockedBy("$1", "f.requester_id") + `)`
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM (`+friendshipSelect+where+`) t`, userID, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := ` ORDER BY lower(u.username), f.id`
	if status == model.FriendPending {
		order = ` ORDER BY f.addressee_id = $1 DESC, f.created_at DESC, f.id`
	}
	rows, err := s.pool.Query(ctx, friendshipSelect+where+order+` LIMIT $3 OFFSET $4`, userID, status, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Friendship{}
	for rows.Next() {
		f, err := scanFriendship(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *f)
	}
	return out, total, rows.Err()
}

// RequestFriend records a request of from to to, or accepts the one to sent
// to from. ErrConflict when they are friends or from already asked.
func (s *PgStore) RequestFriend(ctx context.Context, from, to uuid.UUID) (*model.Friendship, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id, requester uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `SELECT id, requester_id, status FROM friendships
		WHERE LEAST(requester_id, addressee_id) = LEAST($1::uuid, $2::uuid)
		  AND GREATEST(requester_id, addressee_id) = GREATEST($1::uuid, $2::uuid) FOR UPDATE`, from, to).Scan(&id, &requester, &status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `INSERT INTO friendships (requester_id, addressee_id) VALUES ($1, $2) RETURNING id`, from, to).Scan(&id)
		if isUniqueViolation(err) {
			return nil, ErrConflict // demande croisée au même instant
		}
	case err != nil:
	case status == model.FriendAccepted || requester == from:
		return nil, ErrConflict
	default: // sa demande attendait : la mienne vaut acceptation
		_, err = tx.Exec(ctx, `UPDATE friendships SET status = 'accepted', accepted_at = now() WHERE id = $1`, id)
	}
	if err != nil {
		return nil, err
	}
	f, err := scanFriendship(tx.QueryRow(ctx, friendshipSelect+` WHERE f.id = $2`, from, id))
	if err != nil {
		return nil, err
	}
	return f, tx.Commit(ctx)
}

// AcceptFriendRequest accepts a request received by userID.
func (s *PgStore) AcceptFriendRequest(ctx context.Context, userID, id uuid.UUID) (*model.Friendship, error) {
	ct, err := s.pool.Exec(ctx, `UPDATE friendships f SET status = 'accepted', accepted_at = now()
		WHERE f.id = $2 AND f.addressee_id = $1 AND f.status = 'pending' AND `+notBlockedBy("$1", "f.requester_id"), userID, id)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return scanFriendship(s.pool.QueryRow(ctx, friendshipSelect+` WHERE f.id = $2`, userID, id))
}

// DeleteFriendship refuses, cancels or breaks a friendship of userID.
func (s *PgStore) DeleteFriendship(ctx context.Context, userID, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM friendships WHERE id = $2 AND (requester_id = $1 OR addressee_id = $1)`, userID, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Liens d'ami ---

const friendLinkSelect = `SELECT l.id, l.created_at, l.expires_at, l.user_id, u.username, u.display_name, u.avatar
	FROM friend_links l JOIN users u ON u.id = l.user_id AND u.disabled_at IS NULL`

func scanFriendLink(row pgx.Row) (*model.FriendLink, error) {
	var l model.FriendLink
	var a model.ReviewAuthor
	if err := row.Scan(&l.ID, &l.CreatedAt, &l.ExpiresAt, &l.OwnerID, &a.Username, &a.DisplayName, &a.Avatar); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	l.Owner = &a
	return &l, nil
}

// CreateFriendLink records a link of userID; the expired links of everyone go.
func (s *PgStore) CreateFriendLink(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*model.FriendLink, error) {
	if _, err := s.pool.Exec(ctx, `DELETE FROM friend_links WHERE expires_at <= now()`); err != nil {
		return nil, err
	}
	var id uuid.UUID
	if err := s.pool.QueryRow(ctx, `INSERT INTO friend_links (user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING id`,
		userID, tokenHash, expiresAt).Scan(&id); err != nil {
		return nil, err
	}
	return scanFriendLink(s.pool.QueryRow(ctx, friendLinkSelect+` WHERE l.id = $1`, id))
}

// ListFriendLinks lists the links of userID still valid, newest first.
func (s *PgStore) ListFriendLinks(ctx context.Context, userID uuid.UUID) ([]model.FriendLink, error) {
	rows, err := s.pool.Query(ctx, friendLinkSelect+` WHERE l.user_id = $1 AND l.expires_at > now() ORDER BY l.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.FriendLink{}
	for rows.Next() {
		l, err := scanFriendLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

// DeleteFriendLink revokes a link of userID.
func (s *PgStore) DeleteFriendLink(ctx context.Context, userID, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM friend_links WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetFriendLink returns a valid link by the hash of its token, with its owner.
func (s *PgStore) GetFriendLink(ctx context.Context, tokenHash string) (*model.FriendLink, error) {
	return scanFriendLink(s.pool.QueryRow(ctx, friendLinkSelect+` WHERE l.token_hash = $1 AND l.expires_at > now()`, tokenHash))
}

// UseFriendLink makes userID a friend of the link's owner and deletes the
// link, in one transaction: a pending request between them is accepted.
// ErrNotFound when the link is no longer valid, ErrConflict when they are
// already friends (the link is kept).
func (s *PgStore) UseFriendLink(ctx context.Context, tokenHash string, userID uuid.UUID) (*model.Friendship, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var linkID, owner uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id, user_id FROM friend_links WHERE token_hash = $1 AND expires_at > now() FOR UPDATE`,
		tokenHash).Scan(&linkID, &owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var id uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `SELECT id, status FROM friendships
		WHERE LEAST(requester_id, addressee_id) = LEAST($1::uuid, $2::uuid)
		  AND GREATEST(requester_id, addressee_id) = GREATEST($1::uuid, $2::uuid) FOR UPDATE`, owner, userID).Scan(&id, &status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `INSERT INTO friendships (requester_id, addressee_id, status, accepted_at)
			VALUES ($1, $2, 'accepted', now()) RETURNING id`, owner, userID).Scan(&id)
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
	case err != nil:
	case status == model.FriendAccepted:
		return nil, ErrConflict
	default:
		_, err = tx.Exec(ctx, `UPDATE friendships SET status = 'accepted', accepted_at = now() WHERE id = $1`, id)
	}
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM friend_links WHERE id = $1`, linkID); err != nil {
		return nil, err
	}
	f, err := scanFriendship(tx.QueryRow(ctx, friendshipSelect+` WHERE f.id = $2`, userID, id))
	if err != nil {
		return nil, err
	}
	return f, tx.Commit(ctx)
}
