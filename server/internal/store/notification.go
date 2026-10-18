package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shinsaber/colec/internal/model"
)

// Notifications (P4-3c).

// CreateNotifications records one event for each user (duplicates ignored).
func (s *PgStore) CreateNotifications(ctx context.Context, userIDs []uuid.UUID, kind string, payload any) error {
	if len(userIDs) == 0 {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO notifications (user_id, kind, payload)
		SELECT DISTINCT u, $2, $3::jsonb FROM unnest($1::uuid[]) u`, userIDs, kind, raw)
	return err
}

// ListNotifications returns the notifications of a user (newest first) and
// the number of unread ones.
func (s *PgStore) ListNotifications(ctx context.Context, userID uuid.UUID, pg model.Pagination) ([]model.Notification, int, int, error) {
	var total, unread int
	if err := s.pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE read_at IS NULL)
		FROM notifications WHERE user_id = $1`, userID).Scan(&total, &unread); err != nil {
		return nil, 0, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, kind, payload, created_at, read_at FROM notifications
		WHERE user_id = $1 ORDER BY created_at DESC, id LIMIT $2 OFFSET $3`, userID, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, 0, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Notification, error) {
		var n model.Notification
		err := row.Scan(&n.ID, &n.Kind, &n.Payload, &n.CreatedAt, &n.ReadAt)
		return n, err
	})
	if out == nil {
		out = []model.Notification{}
	}
	return out, total, unread, err
}

// MarkNotificationsRead marks the given notifications of a user as read (all
// of them when ids is empty).
func (s *PgStore) MarkNotificationsRead(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		_, err := s.pool.Exec(ctx, `UPDATE notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL`, userID)
		return err
	}
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL AND id = ANY($2)`, userID, ids)
	return err
}
