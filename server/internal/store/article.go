package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

const articleColumns = `a.id, a.item_id, a.author_id, a.title, a.url, a.message, a.lang, a.status, a.reject_reason,
	a.hidden_at, a.hidden_reason, a.comments_closed_at IS NOT NULL,
	(SELECT count(*) FROM comments cc JOIN users cu ON cu.id = cc.author_id AND cu.disabled_at IS NULL
		WHERE cc.target_kind = 'article' AND cc.target_id = a.id AND cc.hidden_at IS NULL AND cc.deleted_at IS NULL),
	a.created_at, a.updated_at, u.username, u.display_name, u.avatar, u.profile_visibility`

func scanArticle(row pgx.Row, extra ...any) (*model.Article, error) {
	var a model.Article
	var au model.ReviewAuthor
	dest := append([]any{&a.ID, &a.ItemID, &a.AuthorID, &a.Title, &a.URL, &a.Message, &a.Lang, &a.Status, &a.RejectReason,
		&a.HiddenAt, &a.HiddenReason, &a.CommentsClosed, &a.CommentCount, &a.CreatedAt, &a.UpdatedAt,
		&au.Username, &au.DisplayName, &au.Avatar, &a.AuthorProfile}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	a.Author = &au
	return &a, nil
}

// ListItemArticles lists the articles of an item, latest first: those
// published and not hidden, and every one of the reader (waiting, refused…).
func (s *PgStore) ListItemArticles(ctx context.Context, itemID, viewer uuid.UUID, pg model.Pagination) ([]model.Article, int, error) {
	where := ` FROM item_articles a JOIN users u ON u.id = a.author_id AND u.disabled_at IS NULL
		WHERE a.item_id = $1 AND ((a.status = 'published' AND a.hidden_at IS NULL) OR a.author_id = $2) AND ` + notBlockedBy("$2", "a.author_id")
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*)`+where, itemID, viewer).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+articleColumns+where+` ORDER BY a.created_at DESC LIMIT $3 OFFSET $4`,
		itemID, viewer, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Article{}
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *a)
	}
	return out, total, rows.Err()
}

// GetArticle returns an article whatever its state.
func (s *PgStore) GetArticle(ctx context.Context, id uuid.UUID) (*model.Article, error) {
	a, err := scanArticle(s.pool.QueryRow(ctx,
		`SELECT `+articleColumns+` FROM item_articles a JOIN users u ON u.id = a.author_id WHERE a.id = $1`, id))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *PgStore) CreateArticle(ctx context.Context, itemID, authorID uuid.UUID, in model.ArticleWrite, status string) (*model.Article, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx,
		`INSERT INTO item_articles (item_id, author_id, title, url, message, lang, status) VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`, itemID, authorID, in.Title, in.URL, in.Message, in.Lang, status).Scan(&id)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.GetArticle(ctx, id)
}

// UpdateArticle replaces an article of authorID and sets its state (back to
// pending for a member: a published article does not change unchecked).
func (s *PgStore) UpdateArticle(ctx context.Context, id, authorID uuid.UUID, in model.ArticleWrite, status string) (*model.Article, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE item_articles SET title = $3, url = $4, message = $5, lang = $6, status = $7,
			reject_reason = NULL, reviewed_by = NULL, reviewed_at = NULL
		 WHERE id = $1 AND author_id = $2`, id, authorID, in.Title, in.URL, in.Message, in.Lang, status)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetArticle(ctx, id)
}

func (s *PgStore) DeleteArticle(ctx context.Context, id, authorID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM item_articles WHERE id = $1 AND author_id = $2`, id, authorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountPendingArticles counts the articles waiting for a moderator.
func (s *PgStore) CountPendingArticles(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM item_articles WHERE status = 'pending'`).Scan(&n)
	return n, err
}

// ListPendingArticles lists the articles waiting for a moderator, oldest first, with their item.
func (s *PgStore) ListPendingArticles(ctx context.Context, pg model.Pagination) ([]model.Article, int, error) {
	total, err := s.CountPendingArticles(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+articleColumns+`, i.item_type_id, i.attributes, it.name, it.attributes_schema
		 FROM item_articles a JOIN users u ON u.id = a.author_id
		 JOIN items i ON i.id = a.item_id JOIN item_types it ON it.id = i.item_type_id
		 WHERE a.status = 'pending' ORDER BY a.created_at LIMIT $1 OFFSET $2`, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Article{}
	for rows.Next() {
		var i model.Item
		var t model.ItemType
		a, err := scanArticle(rows, &i.ItemTypeID, &i.Attributes, &t.Name, &t.AttributesSchema)
		if err != nil {
			return nil, 0, err
		}
		i.ID, t.ID = a.ItemID, i.ItemTypeID
		i.ItemType, a.Item = &t, &i
		out = append(out, *a)
	}
	return out, total, rows.Err()
}

// ReviewArticle publishes or refuses a waiting article (ErrConflict otherwise).
func (s *PgStore) ReviewArticle(ctx context.Context, id, moderatorID uuid.UUID, publish bool, reason string) (*model.Article, error) {
	status, rejectReason := model.ArticleRejected, any(reason)
	if publish {
		status, rejectReason = model.ArticlePublished, nil
	}
	ct, err := s.pool.Exec(ctx,
		`UPDATE item_articles SET status = $3, reviewed_by = $2, reviewed_at = now(),
			reject_reason = $4
		 WHERE id = $1 AND status = 'pending'`, id, moderatorID, status, rejectReason)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		if _, err := s.GetArticle(ctx, id); err != nil {
			return nil, err
		}
		return nil, ErrConflict
	}
	return s.GetArticle(ctx, id)
}

// SetArticleCommentsClosed closes or reopens the thread of an article of userID.
func (s *PgStore) SetArticleCommentsClosed(ctx context.Context, id, userID uuid.UUID, closed bool) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE item_articles SET comments_closed_at = CASE WHEN $3 THEN COALESCE(comments_closed_at, now()) END
		 WHERE id = $1 AND author_id = $2`, id, userID, closed)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
