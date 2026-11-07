package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shinsaber/colec/internal/model"
)

// Recommendations and blocks (I-4).

const recommendationColumns = `r.id, r.item_id, r.sender_id, r.message, r.created_at, r.read_at,
	u.username, u.display_name, u.avatar, i.item_type_id, i.attributes, i.created_at, i.updated_at,
	it.name, it.description, it.attributes_schema`

const recommendationFrom = ` FROM recommendations r JOIN users u ON u.id = r.sender_id AND u.disabled_at IS NULL
	JOIN items i ON i.id = r.item_id JOIN item_types it ON it.id = i.item_type_id`

func scanRecommendation(row pgx.Row) (*model.Recommendation, error) {
	var rc model.Recommendation
	var a model.ReviewAuthor
	var i model.Item
	var t model.ItemType
	if err := row.Scan(&rc.ID, &rc.ItemID, &rc.SenderID, &rc.Message, &rc.CreatedAt, &rc.ReadAt,
		&a.Username, &a.DisplayName, &a.Avatar, &i.ItemTypeID, &i.Attributes, &i.CreatedAt, &i.UpdatedAt,
		&t.Name, &t.Description, &t.AttributesSchema); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	i.ID, t.ID = rc.ItemID, i.ItemTypeID
	i.ItemType = &t
	rc.Item, rc.Sender = &i, &a
	return &rc, nil
}

// CreateRecommendation records an item sent to a member (ErrConflict when
// the same sender already recommended it to them).
func (s *PgStore) CreateRecommendation(ctx context.Context, senderID, recipientID, itemID uuid.UUID, message string) (*model.Recommendation, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `INSERT INTO recommendations (sender_id, recipient_id, item_id, message)
		VALUES ($1, $2, $3, $4) ON CONFLICT (sender_id, recipient_id, item_id) DO NOTHING RETURNING id`,
		senderID, recipientID, itemID, message).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	return s.GetRecommendation(ctx, id, recipientID)
}

// GetRecommendation returns a recommendation received by recipientID.
func (s *PgStore) GetRecommendation(ctx context.Context, id, recipientID uuid.UUID) (*model.Recommendation, error) {
	return scanRecommendation(s.pool.QueryRow(ctx,
		`SELECT `+recommendationColumns+recommendationFrom+` WHERE r.id = $1 AND r.recipient_id = $2`, id, recipientID))
}

// ListRecommendations returns the recommendations received by a member
// (newest first) and the number of unread ones.
func (s *PgStore) ListRecommendations(ctx context.Context, recipientID uuid.UUID, pg model.Pagination) ([]model.Recommendation, int, int, error) {
	const where = ` WHERE r.recipient_id = $1`
	var total, unread int
	if err := s.pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE r.read_at IS NULL)`+recommendationFrom+where,
		recipientID).Scan(&total, &unread); err != nil {
		return nil, 0, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+recommendationColumns+recommendationFrom+where+
		` ORDER BY r.created_at DESC, r.id LIMIT $2 OFFSET $3`, recipientID, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	out := []model.Recommendation{}
	for rows.Next() {
		rc, err := scanRecommendation(rows)
		if err != nil {
			return nil, 0, 0, err
		}
		out = append(out, *rc)
	}
	return out, total, unread, rows.Err()
}

// MarkRecommendationsRead marks the given recommendations of a member as read
// (all of them when ids is empty).
func (s *PgStore) MarkRecommendationsRead(ctx context.Context, recipientID uuid.UUID, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE recommendations SET read_at = now()
		WHERE recipient_id = $1 AND read_at IS NULL AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR id = ANY($2))`, recipientID, ids)
	return err
}

// DeleteRecommendation dismisses a recommendation received by recipientID.
func (s *PgStore) DeleteRecommendation(ctx context.Context, id, recipientID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM recommendations WHERE id = $1 AND recipient_id = $2`, id, recipientID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const blockColumns = `b.id, b.blocked_id, b.anonymous, b.created_at, u.username, u.display_name, u.avatar, u.profile_visibility`

func scanBlock(row pgx.Row) (*model.Block, error) {
	var b model.Block
	var a model.ReviewAuthor
	if err := row.Scan(&b.ID, &b.BlockedID, &b.Anonymous, &b.CreatedAt, &a.Username, &a.DisplayName, &a.Avatar, &b.BlockedProfile); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	b.User = &a
	return &b, nil
}

// BlockUser blocks a member; anonymous when the blocker does not know who they
// are (a block by name lifts it). The recommendations they sent to the blocker go.
func (s *PgStore) BlockUser(ctx context.Context, blockerID, blockedID uuid.UUID, anonymous bool) (*model.Block, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id, anonymous) VALUES ($1, $2, $3)
		ON CONFLICT (blocker_id, blocked_id) DO UPDATE SET anonymous = user_blocks.anonymous AND EXCLUDED.anonymous`,
		blockerID, blockedID, anonymous); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM recommendations WHERE sender_id = $2 AND recipient_id = $1`, blockerID, blockedID); err != nil {
		return nil, err
	}
	b, err := scanBlock(tx.QueryRow(ctx, `SELECT `+blockColumns+` FROM user_blocks b JOIN users u ON u.id = b.blocked_id
		WHERE b.blocker_id = $1 AND b.blocked_id = $2`, blockerID, blockedID))
	if err != nil {
		return nil, err
	}
	return b, tx.Commit(ctx)
}

// ListBlocks returns the members blocked by blockerID, newest first.
func (s *PgStore) ListBlocks(ctx context.Context, blockerID uuid.UUID, pg model.Pagination) ([]model.Block, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM user_blocks WHERE blocker_id = $1`, blockerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+blockColumns+` FROM user_blocks b JOIN users u ON u.id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC, b.id LIMIT $2 OFFSET $3`, blockerID, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Block{}
	for rows.Next() {
		b, err := scanBlock(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *b)
	}
	return out, total, rows.Err()
}

// DeleteBlock lifts a block of blockerID.
func (s *PgStore) DeleteBlock(ctx context.Context, blockerID, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM user_blocks WHERE id = $1 AND blocker_id = $2`, id, blockerID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// BlockedBy lists the members blockerID blocked.
func (s *PgStore) BlockedBy(ctx context.Context, blockerID uuid.UUID) ([]uuid.UUID, error) {
	if blockerID == uuid.Nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT blocked_id FROM user_blocks WHERE blocker_id = $1`, blockerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}

// Blockers returns those of users who blocked blockedID.
func (s *PgStore) Blockers(ctx context.Context, blockedID uuid.UUID, users []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `SELECT blocker_id FROM user_blocks WHERE blocked_id = $1 AND blocker_id = ANY($2)`,
		blockedID, users)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}

// IsBlocked tells whether blockerID blocked blockedID.
func (s *PgStore) IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2)`,
		blockerID, blockedID).Scan(&ok)
	return ok, err
}

// notBlockedBy: the author column is not a member the viewer (parameter idx) blocked.
func notBlockedBy(idx, author string) string {
	return `NOT EXISTS (SELECT 1 FROM user_blocks ub WHERE ub.blocker_id = ` + idx + ` AND ub.blocked_id = ` + author + `)`
}
