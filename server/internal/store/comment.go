package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

const commentColumns = `c.id, c.target_kind, c.target_id, c.author_id, c.body, c.hidden_at, c.hidden_reason,
	c.created_at, c.updated_at, u.username, u.display_name, u.avatar`

func scanComment(row pgx.Row) (*model.Comment, error) {
	var c model.Comment
	var a model.ReviewAuthor
	if err := row.Scan(&c.ID, &c.TargetKind, &c.TargetID, &c.AuthorID, &c.Body, &c.HiddenAt, &c.HiddenReason,
		&c.CreatedAt, &c.UpdatedAt, &a.Username, &a.DisplayName, &a.Avatar); err != nil {
		return nil, err
	}
	c.Author = &a
	return &c, nil
}

// ListComments lists the thread of a target, oldest first: the hidden comments
// only for their author; none of a disabled account.
func (s *PgStore) ListComments(ctx context.Context, kind string, targetID, viewer uuid.UUID, pg model.Pagination) ([]model.Comment, int, error) {
	where := ` FROM comments c JOIN users u ON u.id = c.author_id AND u.disabled_at IS NULL
		WHERE c.target_kind = $1 AND c.target_id = $2 AND (c.hidden_at IS NULL OR c.author_id = $3)`
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*)`+where, kind, targetID, viewer).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+commentColumns+where+` ORDER BY c.created_at, c.id LIMIT $4 OFFSET $5`,
		kind, targetID, viewer, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Comment{}
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *c)
	}
	return out, total, rows.Err()
}

// GetComment returns a comment with its author, whatever its state.
func (s *PgStore) GetComment(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	c, err := scanComment(s.pool.QueryRow(ctx,
		`SELECT `+commentColumns+` FROM comments c JOIN users u ON u.id = c.author_id WHERE c.id = $1`, id))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return c, err
}

func (s *PgStore) CreateComment(ctx context.Context, kind string, targetID, authorID uuid.UUID, body string) (*model.Comment, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx,
		`INSERT INTO comments (target_kind, target_id, author_id, body) VALUES ($1, $2, $3, $4) RETURNING id`,
		kind, targetID, authorID, body).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetComment(ctx, id)
}

// UpdateComment replaces the text of a comment of authorID (ErrNotFound otherwise).
func (s *PgStore) UpdateComment(ctx context.Context, id, authorID uuid.UUID, body string) (*model.Comment, error) {
	ct, err := s.pool.Exec(ctx, `UPDATE comments SET body = $3 WHERE id = $1 AND author_id = $2`, id, authorID, body)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetComment(ctx, id)
}

// DeleteComment removes a comment of authorID (ErrNotFound otherwise).
func (s *PgStore) DeleteComment(ctx context.Context, id, authorID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM comments WHERE id = $1 AND author_id = $2`, id, authorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CommentParticipants lists the authors of a thread.
func (s *PgStore) CommentParticipants(ctx context.Context, kind string, targetID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT author_id FROM comments WHERE target_kind = $1 AND target_id = $2`, kind, targetID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}

// SetReviewCommentsClosed closes or reopens the thread of a review of userID.
func (s *PgStore) SetReviewCommentsClosed(ctx context.Context, reviewID, userID uuid.UUID, closed bool) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE reviews SET comments_closed_at = CASE WHEN $3 THEN COALESCE(comments_closed_at, now()) END
		 WHERE id = $1 AND user_id = $2`, reviewID, userID, closed)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
