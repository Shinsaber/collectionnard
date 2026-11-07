package store

import (
	"context"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// maxExcerpt bounds the copy of a content kept by the journal.
const maxExcerpt = 2000

// GetReview returns a review by id, whatever its visibility or state, with its
// author: for the moderation and the reactions, never sent as is to a reader.
func (s *PgStore) GetReview(ctx context.Context, id uuid.UUID) (*model.Review, error) {
	r, err := scanReview(s.pool.QueryRow(ctx,
		`SELECT `+reviewColumns+` FROM reviews r JOIN users u ON u.id = r.user_id`+reviewContextJoin+`WHERE r.id = $1`, id))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return r, err
}

// CreateReport records a report; a second open report of the same reporter on
// the same target is ErrConflict.
func (s *PgStore) CreateReport(ctx context.Context, reporterID uuid.UUID, in model.ReportCreate) (*model.Report, error) {
	var rep model.Report
	err := s.pool.QueryRow(ctx,
		`INSERT INTO reports (target_kind, target_id, reporter_id, reason, details) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, target_kind, target_id, reason, details, status, reporter_id, created_at`,
		in.TargetKind, in.TargetID, reporterID, in.Reason, in.Details,
	).Scan(&rep.ID, &rep.TargetKind, &rep.TargetID, &rep.Reason, &rep.Details, &rep.Status, &rep.ReporterID, &rep.CreatedAt)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	return &rep, err
}

// CountModerationQueue counts the targets with an open report.
func (s *PgStore) CountModerationQueue(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(DISTINCT (target_kind, target_id)) FROM reports WHERE status = 'open'`).Scan(&n)
	return n, err
}

// ListModerationQueue lists the targets with an open report, oldest first.
func (s *PgStore) ListModerationQueue(ctx context.Context, pg model.Pagination) ([]model.ModerationItem, int, error) {
	total, err := s.CountModerationQueue(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT target_kind, target_id FROM reports WHERE status = 'open'
		 GROUP BY target_kind, target_id ORDER BY min(created_at) LIMIT $1 OFFSET $2`, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	var out []model.ModerationItem
	for rows.Next() {
		var it model.ModerationItem
		if err := rows.Scan(&it.TargetKind, &it.TargetID); err != nil {
			rows.Close()
			return nil, 0, err
		}
		out = append(out, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	for i := range out {
		if err := s.fillModerationItem(ctx, &out[i]); err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

func (s *PgStore) fillModerationItem(ctx context.Context, it *model.ModerationItem) error {
	reviewID := it.TargetID
	if it.TargetKind == model.ReportTargetComment {
		c, err := s.GetComment(ctx, it.TargetID)
		switch {
		case err == ErrNotFound:
			reviewID = uuid.Nil
		case err != nil:
			return err
		default:
			it.Comment, it.AuthorID = c, c.AuthorID
			reviewID = c.TargetID
		}
	}
	if reviewID != uuid.Nil && (it.TargetKind == model.ReportTargetReview || it.TargetKind == model.ReportTargetComment) {
		r, err := s.GetReview(ctx, reviewID)
		switch {
		case err == ErrNotFound: // supprimé par son auteur depuis le signalement
		case err != nil:
			return err
		default:
			it.Review = r
			if it.TargetKind == model.ReportTargetReview {
				it.AuthorID = r.UserID
			}
			if r.ReviewTypeID != nil {
				if rt, err := s.GetReviewType(ctx, *r.ReviewTypeID); err == nil {
					it.ReviewType = rt
				}
			}
			if item, err := s.GetItem(ctx, r.ItemID); err == nil {
				it.Item = item
			}
		}
	}
	reports, err := s.listReports(ctx, `WHERE p.target_kind = $1 AND p.target_id = $2 AND p.status = 'open'`, it.TargetKind, it.TargetID)
	if err != nil {
		return err
	}
	it.Reports = reports
	if it.AuthorID != uuid.Nil {
		recap, err := s.ModerationRecap(ctx, it.AuthorID)
		if err != nil {
			return err
		}
		it.AuthorRecap = *recap
	}
	return nil
}

func (s *PgStore) listReports(ctx context.Context, where string, args ...any) ([]model.Report, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT p.id, p.target_kind, p.target_id, p.reason, p.details, p.status, p.reporter_id, p.created_at,
			u.username, u.display_name, u.avatar,
			(SELECT count(*) FROM reports d WHERE d.reporter_id = p.reporter_id AND d.status = 'dismissed'
				AND d.handled_at > now() - interval '90 days')
		 FROM reports p JOIN users u ON u.id = p.reporter_id `+where+` ORDER BY p.created_at DESC LIMIT 100`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Report{}
	for rows.Next() {
		var rep model.Report
		var a model.ReviewAuthor
		if err := rows.Scan(&rep.ID, &rep.TargetKind, &rep.TargetID, &rep.Reason, &rep.Details, &rep.Status,
			&rep.ReporterID, &rep.CreatedAt, &a.Username, &a.DisplayName, &a.Avatar, &rep.ReporterDismissed); err != nil {
			return nil, err
		}
		rep.Reporter = &a
		out = append(out, rep)
	}
	return out, rows.Err()
}

// ModerationRecap counts the reviews of a member hidden by the moderation, and
// their reports dismissed.
func (s *PgStore) ModerationRecap(ctx context.Context, userID uuid.UUID) (*model.ModerationRecap, error) {
	var r model.ModerationRecap
	err := s.pool.QueryRow(ctx,
		`SELECT
			(SELECT count(*) FROM moderation_log WHERE target_author_id = $1 AND action = 'hide'
				AND created_at > now() - interval '90 days'),
			(SELECT count(*) FROM moderation_log WHERE target_author_id = $1 AND action = 'hide'),
			(SELECT count(*) FROM reports WHERE reporter_id = $1 AND status = 'dismissed'
				AND handled_at > now() - interval '90 days')`, userID).Scan(&r.Hidden90d, &r.HiddenTotal, &r.Dismissed90d)
	return &r, err
}

// excerptOf copies a content for the journal, bounded.
func excerptOf(text string) string {
	if utf8.RuneCountInString(text) > maxExcerpt {
		text = string([]rune(text)[:maxExcerpt]) + "…"
	}
	return text
}

// moderationTables: the table of each moderated kind (its name is part of the SQL).
var moderationTables = map[string]string{model.ReportTargetReview: "reviews", model.ReportTargetComment: "comments"}

// moderationTarget returns the author, the item and the copy of a moderated content.
func (s *PgStore) moderationTarget(ctx context.Context, kind string, id uuid.UUID) (author, item uuid.UUID, excerpt string, err error) {
	switch kind {
	case model.ReportTargetReview:
		r, err := s.GetReview(ctx, id)
		if err != nil {
			return uuid.Nil, uuid.Nil, "", err
		}
		return r.UserID, r.ItemID, excerptOf(r.Body), nil
	case model.ReportTargetComment:
		c, err := s.GetComment(ctx, id)
		if err != nil {
			return uuid.Nil, uuid.Nil, "", err
		}
		if c.TargetKind == model.CommentTargetReview {
			if r, err := s.GetReview(ctx, c.TargetID); err == nil {
				item = r.ItemID
			}
		}
		return c.AuthorID, item, excerptOf(c.Body), nil
	}
	return uuid.Nil, uuid.Nil, "", ErrNotFound
}

// Moderate hides or restores a review or a comment, or dismisses its open
// reports, in one transaction with its journal entry. A hide or a dismiss closes
// the open reports; an action that does not apply (hide a hidden content…) is
// ErrConflict.
func (s *PgStore) Moderate(ctx context.Context, kind string, targetID, moderatorID uuid.UUID, action, reason string) (*model.ModerationResult, error) {
	table, ok := moderationTables[kind]
	if !ok {
		return nil, ErrNotFound
	}
	author, item, excerpt, err := s.moderationTarget(ctx, kind, targetID)
	if err != nil {
		return nil, err
	}
	reviewID := targetID
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // sans effet après Commit

	var sql, status string
	var args []any
	switch action {
	case model.ModerationHide:
		sql, status = `UPDATE `+table+` SET hidden_at = now(), hidden_by = $2, hidden_reason = $3
			WHERE id = $1 AND hidden_at IS NULL`, model.ReportActioned
		args = []any{reviewID, moderatorID, reason}
	case model.ModerationRestore:
		sql = `UPDATE ` + table + ` SET hidden_at = NULL, hidden_by = NULL, hidden_reason = NULL
			WHERE id = $1 AND hidden_at IS NOT NULL`
		args = []any{reviewID}
	case model.ModerationDismiss:
		status = model.ReportDismissed
	default:
		return nil, ErrConflict
	}
	if sql != "" {
		ct, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			return nil, err
		}
		if ct.RowsAffected() == 0 {
			return nil, ErrConflict
		}
	}
	if status != "" {
		ct, err := tx.Exec(ctx,
			`UPDATE reports SET status = $3, handled_by = $2, handled_at = now()
			 WHERE target_kind = $4 AND target_id = $1 AND status = 'open'`, reviewID, moderatorID, status, kind)
		if err != nil {
			return nil, err
		}
		if action == model.ModerationDismiss && ct.RowsAffected() == 0 {
			return nil, ErrConflict
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO moderation_log (action, target_kind, target_id, target_author_id, moderator_id, reason, excerpt)
		 VALUES ($1, $7, $2, $3, $4, $5, $6)`,
		action, reviewID, author, moderatorID, reason, excerpt, kind); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &model.ModerationResult{AuthorID: author, ItemID: item}, nil
}

const logColumns = `l.id, l.action, l.target_kind, l.target_id, l.target_author_id, l.reason, l.excerpt, l.created_at,
	a.username, a.display_name, a.avatar, m.username, m.display_name, m.avatar,
	EXISTS (SELECT 1 FROM reviews r WHERE l.target_kind = 'review' AND r.id = l.target_id AND r.hidden_at IS NOT NULL)
		OR EXISTS (SELECT 1 FROM comments c WHERE l.target_kind = 'comment' AND c.id = l.target_id AND c.hidden_at IS NOT NULL)
	FROM moderation_log l LEFT JOIN users a ON a.id = l.target_author_id LEFT JOIN users m ON m.id = l.moderator_id `

func scanLog(rows pgx.Rows) ([]model.ModerationLogEntry, error) {
	defer rows.Close()
	out := []model.ModerationLogEntry{}
	for rows.Next() {
		var e model.ModerationLogEntry
		var au, mu *string
		var a, m model.ReviewAuthor
		if err := rows.Scan(&e.ID, &e.Action, &e.TargetKind, &e.TargetID, &e.AuthorID, &e.Reason, &e.Excerpt, &e.CreatedAt,
			&au, &a.DisplayName, &a.Avatar, &mu, &m.DisplayName, &m.Avatar, &e.StillHidden); err != nil {
			return nil, err
		}
		if au != nil {
			a.Username = *au
			e.TargetAuthor = &a
		}
		if mu != nil {
			m.Username = *mu
			e.Moderator = &m
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListModerationLog returns the journal, latest first.
func (s *PgStore) ListModerationLog(ctx context.Context, pg model.Pagination) ([]model.ModerationLogEntry, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM moderation_log`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+logColumns+`ORDER BY l.created_at DESC LIMIT $1 OFFSET $2`, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	out, err := scanLog(rows)
	return out, total, err
}

// ModerationHistory returns the record of a member (ErrNotFound if unknown).
func (s *PgStore) ModerationHistory(ctx context.Context, userID uuid.UUID) (*model.ModerationHistory, error) {
	var h model.ModerationHistory
	err := s.pool.QueryRow(ctx, `SELECT username, display_name, avatar FROM users WHERE id = $1`, userID).
		Scan(&h.User.Username, &h.User.DisplayName, &h.User.Avatar)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	recap, err := s.ModerationRecap(ctx, userID)
	if err != nil {
		return nil, err
	}
	h.Recap = *recap
	rows, err := s.pool.Query(ctx, `SELECT `+logColumns+`WHERE l.target_author_id = $1 ORDER BY l.created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	if h.Entries, err = scanLog(rows); err != nil {
		return nil, err
	}
	if h.Dismissed, err = s.listReports(ctx, `WHERE p.reporter_id = $1 AND p.status = 'dismissed'`, userID); err != nil {
		return nil, err
	}
	return &h, nil
}

// SetReaction adds (on) or removes a reaction of a user to a review.
func (s *PgStore) SetReaction(ctx context.Context, reviewID, userID uuid.UUID, kind string, on bool) error {
	sql := `DELETE FROM review_reactions WHERE review_id = $1 AND user_id = $2 AND kind = $3`
	if on {
		sql = `INSERT INTO review_reactions (review_id, user_id, kind) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
	}
	_, err := s.pool.Exec(ctx, sql, reviewID, userID, kind)
	if isForeignKeyViolation(err) {
		return ErrNotFound
	}
	return err
}

// ReactionsOf returns the counts of each reaction of the given reviews, and those of viewer.
func (s *PgStore) ReactionsOf(ctx context.Context, reviewIDs []uuid.UUID, viewer uuid.UUID) (map[uuid.UUID]map[string]int, map[uuid.UUID][]string, error) {
	counts, mine := map[uuid.UUID]map[string]int{}, map[uuid.UUID][]string{}
	if len(reviewIDs) == 0 {
		return counts, mine, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT review_id, kind, count(*), bool_or(user_id = $2) FROM review_reactions
		 WHERE review_id = ANY($1) GROUP BY review_id, kind`, reviewIDs, viewer)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var kind string
		var n int
		var me bool
		if err := rows.Scan(&id, &kind, &n, &me); err != nil {
			return nil, nil, err
		}
		if counts[id] == nil {
			counts[id] = map[string]int{}
		}
		counts[id][kind] = n
		if me {
			mine[id] = append(mine[id], kind)
		}
	}
	return counts, mine, rows.Err()
}
