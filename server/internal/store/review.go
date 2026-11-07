package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// reviewColumns: a review with its author, whom the handler keeps only when the
// review is signed for the reader. Reviews of disabled accounts count nowhere.
const reviewColumns = `r.id, r.item_id, r.user_id, r.review_type_id, r.criteria, r.rating::float8, r.body, r.spoiler,
	r.lang, r.visibility, r.created_at, r.updated_at, u.username, u.display_name, u.avatar,
	r.context_item_id, ci.item_type_id, ci.attributes, cit.name, cit.attributes_schema, r.hidden_at, r.hidden_reason,
	r.comments_closed_at IS NOT NULL, (SELECT count(*) FROM comments cc JOIN users cu ON cu.id = cc.author_id
		AND cu.disabled_at IS NULL WHERE cc.target_kind = 'review' AND cc.target_id = r.id AND cc.hidden_at IS NULL AND cc.deleted_at IS NULL)`

// reviewContextJoin: the edition used, with what its label needs.
const reviewContextJoin = ` LEFT JOIN items ci ON ci.id = r.context_item_id
	LEFT JOIN item_types cit ON cit.id = ci.item_type_id `

const reviewFrom = ` FROM reviews r JOIN users u ON u.id = r.user_id AND u.disabled_at IS NULL ` + reviewContextJoin

// reviewListed: the review appears in the list for the reader (signed levels in
// parameter idx); otherwise only its note counts, in the average. An anonymous
// review without text has nothing to show.
func reviewListed(idx string) string {
	return `(r.visibility = ANY(` + idx + `) OR (r.visibility = 'anonymous' AND r.body <> ''))`
}

// reviewScan holds the destinations of reviewColumns, extra ones appended.
type reviewScan struct {
	r         model.Review
	a         model.ReviewAuthor
	ctxType   *uuid.UUID
	ctxAttrs  []byte
	ctxName   *string
	ctxSchema []byte
}

func (v *reviewScan) dest(extra ...any) []any {
	r := &v.r
	return append([]any{&r.ID, &r.ItemID, &r.UserID, &r.ReviewTypeID, &r.Criteria, &r.Rating, &r.Body, &r.Spoiler,
		&r.Lang, &r.Visibility, &r.CreatedAt, &r.UpdatedAt, &v.a.Username, &v.a.DisplayName, &v.a.Avatar,
		&r.ContextItemID, &v.ctxType, &v.ctxAttrs, &v.ctxName, &v.ctxSchema, &r.HiddenAt, &r.HiddenReason,
		&r.CommentsClosed, &r.CommentCount}, extra...)
}

func (v *reviewScan) review() *model.Review {
	r := v.r
	r.Author = &v.a
	if r.ContextItemID != nil && v.ctxType != nil {
		t := model.ItemType{ID: *v.ctxType, AttributesSchema: v.ctxSchema}
		if v.ctxName != nil {
			t.Name = *v.ctxName
		}
		r.ContextItem = &model.Item{ID: *r.ContextItemID, ItemTypeID: t.ID, ItemType: &t, Attributes: v.ctxAttrs}
	}
	return &r
}

func scanReview(row pgx.Row) (*model.Review, error) {
	var v reviewScan
	if err := row.Scan(v.dest()...); err != nil {
		return nil, err
	}
	return v.review(), nil
}

// GetItemReviewType returns the review type of an item (nil: reviews closed on
// its type), ErrNotFound when the item does not exist.
func (s *PgStore) GetItemReviewType(ctx context.Context, itemID uuid.UUID) (*model.ReviewType, error) {
	var id *uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT it.review_type_id FROM items i JOIN item_types it ON it.id = i.item_type_id WHERE i.id = $1`, itemID,
	).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil || id == nil {
		return nil, err
	}
	return s.GetReviewType(ctx, *id)
}

// ListReviewContextItems lists the editions a review of an item may name: the
// targets of the item by a context relation of the review type (100 at most).
func (s *PgStore) ListReviewContextItems(ctx context.Context, reviewTypeID, itemID uuid.UUID) ([]model.Item, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT i.id, i.item_type_id, i.attributes, i.created_at, i.updated_at, it.name, it.attributes_schema
		 FROM item_relations ir
		 JOIN review_type_context_relations c ON c.relation_type_id = ir.relation_type_id AND c.review_type_id = $1
		 JOIN items i ON i.id = ir.target_item_id JOIN item_types it ON it.id = i.item_type_id
		 WHERE ir.source_item_id = $2 ORDER BY i.created_at, i.id LIMIT 100`, reviewTypeID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Item{}
	for rows.Next() {
		var i model.Item
		var t model.ItemType
		if err := rows.Scan(&i.ID, &i.ItemTypeID, &i.Attributes, &i.CreatedAt, &i.UpdatedAt, &t.Name, &t.AttributesSchema); err != nil {
			return nil, err
		}
		t.ID = i.ItemTypeID
		i.ItemType = &t
		out = append(out, i)
	}
	return out, rows.Err()
}

// IsReviewContext tells whether contextID is an edition a review of itemID may name.
func (s *PgStore) IsReviewContext(ctx context.Context, reviewTypeID, itemID, contextID uuid.UUID) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM item_relations ir
		 JOIN review_type_context_relations c ON c.relation_type_id = ir.relation_type_id AND c.review_type_id = $1
		 WHERE ir.source_item_id = $2 AND ir.target_item_id = $3)`, reviewTypeID, itemID, contextID).Scan(&ok)
	return ok, err
}

// ListItemReviews lists the reviews of an item shown to the reader, their own excluded.
func (s *PgStore) ListItemReviews(ctx context.Context, itemID uuid.UUID, p model.ReviewListParams, pg model.Pagination) ([]model.Review, int, error) {
	// Filtre par édition : seuls les avis signés pour le lecteur la montrent.
	// Membres bloqués : leurs avis signés sortent ; un avis anonyme reste, le retirer nommerait son auteur.
	where := `WHERE r.item_id = $1 AND r.user_id <> $2 AND r.hidden_at IS NULL
		AND (r.visibility = 'anonymous' OR ` + notBlockedBy("$2", "r.user_id") + `)` +
		` AND ` + reviewListed("$3") + ` AND ($4 = '' OR r.lang = $4)
		AND ($5::uuid IS NULL OR (r.context_item_id = $5 AND r.visibility = ANY($3)))`
	var total int
	err := s.pool.QueryRow(ctx, `SELECT count(*)`+reviewFrom+where, itemID, p.Viewer, p.Levels, p.Lang, p.ContextItemID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	order := `r.created_at DESC`
	switch p.Sort {
	case "rating":
		order = `r.rating DESC, r.created_at DESC`
	case "interesting":
		order = `(SELECT count(*) FROM review_reactions rr WHERE rr.review_id = r.id AND rr.kind = 'interesting') DESC,
			r.created_at DESC`
	}
	rows, err := s.pool.Query(ctx, `SELECT `+reviewColumns+reviewFrom+where+` ORDER BY `+order+` LIMIT $6 OFFSET $7`,
		itemID, p.Viewer, p.Levels, p.Lang, p.ContextItemID, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Review{}
	for rows.Next() {
		r, err := scanReview(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *r)
	}
	return out, total, rows.Err()
}

// GetReviewStats aggregates the reviews of an item, whatever their visibility.
func (s *PgStore) GetReviewStats(ctx context.Context, itemID, viewer uuid.UUID, levels []string) (*model.ReviewStats, error) {
	st := model.ReviewStats{Criteria: map[string]model.ReviewCriterionStats{}}
	err := s.pool.QueryRow(ctx,
		`SELECT count(*), COALESCE(avg(r.rating), 0)::float8,
			count(*) FILTER (WHERE r.user_id <> $2 AND NOT `+reviewListed("$3")+`)`+reviewFrom+`WHERE r.item_id = $1 AND r.hidden_at IS NULL`,
		itemID, viewer, levels).Scan(&st.Count, &st.Avg, &st.NoteOnly)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT c.key, count(*), avg(c.value::numeric)::float8`+reviewFrom+`
		 CROSS JOIN LATERAL jsonb_each_text(r.criteria) c
		 WHERE r.item_id = $1 AND r.criteria IS NOT NULL AND r.hidden_at IS NULL GROUP BY c.key`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var c model.ReviewCriterionStats
		if err := rows.Scan(&key, &c.Count, &c.Avg); err != nil {
			return nil, err
		}
		st.Criteria[key] = c
	}
	return &st, rows.Err()
}

// GetMyReview returns the review of a user on an item.
func (s *PgStore) GetMyReview(ctx context.Context, itemID, userID uuid.UUID) (*model.Review, error) {
	r, err := scanReview(s.pool.QueryRow(ctx,
		`SELECT `+reviewColumns+` FROM reviews r JOIN users u ON u.id = r.user_id`+reviewContextJoin+`
		 WHERE r.item_id = $1 AND r.user_id = $2`, itemID, userID))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return r, err
}

// PutReview creates or replaces the review of a user on an item.
func (s *PgStore) PutReview(ctx context.Context, itemID, userID uuid.UUID, in model.ReviewWrite) (*model.Review, error) {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO reviews (user_id, item_id, review_type_id, criteria, rating, body, spoiler, lang, visibility, context_item_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (user_id, item_id) DO UPDATE SET
			review_type_id = EXCLUDED.review_type_id, criteria = EXCLUDED.criteria, rating = EXCLUDED.rating,
			context_item_id = EXCLUDED.context_item_id,
			body = EXCLUDED.body, spoiler = EXCLUDED.spoiler, lang = EXCLUDED.lang, visibility = EXCLUDED.visibility`,
		userID, itemID, in.ReviewTypeID, nullableJSON(in.Criteria), in.Rating, in.Body, in.Spoiler, in.Lang, in.Visibility, in.ContextItemID)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.GetMyReview(ctx, itemID, userID)
}

// DeleteReview removes the review of a user on an item.
func (s *PgStore) DeleteReview(ctx context.Context, itemID, userID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM reviews WHERE item_id = $1 AND user_id = $2`, itemID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListUserReviews lists the reviews of a user with their item, latest first:
// every one when levels is nil (their own page), else those signed at these levels.
func (s *PgStore) ListUserReviews(ctx context.Context, userID uuid.UUID, levels []string, pg model.Pagination) ([]model.Review, int, error) {
	// Sa propre liste montre aussi les avis masqués, avec leur motif.
	where := `WHERE r.user_id = $1 AND ($2::text[] IS NULL OR (r.visibility = ANY($2) AND r.hidden_at IS NULL))`
	var total int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM reviews r `+where, userID, levels).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+reviewColumns+`, i.item_type_id, i.attributes, i.created_at, i.updated_at,
			it.name, it.description, it.attributes_schema`+reviewFrom+`
		 JOIN items i ON i.id = r.item_id JOIN item_types it ON it.id = i.item_type_id
		 `+where+` ORDER BY r.created_at DESC LIMIT $3 OFFSET $4`, userID, levels, pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Review{}
	for rows.Next() {
		var v reviewScan
		var i model.Item
		var t model.ItemType
		if err := rows.Scan(v.dest(&i.ItemTypeID, &i.Attributes, &i.CreatedAt, &i.UpdatedAt, &t.Name, &t.Description, &t.AttributesSchema)...); err != nil {
			return nil, 0, err
		}
		r := v.review()
		i.ID, t.ID = r.ItemID, i.ItemTypeID
		i.ItemType, r.Item = &t, &i
		out = append(out, *r)
	}
	return out, total, rows.Err()
}
