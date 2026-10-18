package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shinsaber/colec/internal/model"
)

// Proposals (P4-3b). Status changes are guarded in SQL (`WHERE status = ANY`):
// a concurrent change makes the second one fail with ErrConflict.

const proposalColumns = `p.id, p.kind, p.status, p.title, p.description, p.payload, p.base, p.item_id,
	p.author_id, a.username, a.display_name, a.avatar, a.disabled_at IS NOT NULL,
	p.reviewer_id, r.username, r.display_name, r.avatar, p.review_comment,
	(SELECT count(*) FROM proposal_comments c WHERE c.proposal_id = p.id),
	p.created_at, p.updated_at, p.submitted_at, p.decided_at`

const proposalFrom = ` FROM proposals p JOIN users a ON a.id = p.author_id LEFT JOIN users r ON r.id = p.reviewer_id`

func scanProposal(row pgx.Row) (*model.Proposal, error) {
	var p model.Proposal
	var payload, base []byte
	var rName *string
	var rDisplay, rAvatar *string
	if err := row.Scan(&p.ID, &p.Kind, &p.Status, &p.Title, &p.Description, &payload, &base, &p.ItemID,
		&p.AuthorID, &p.Author.Username, &p.Author.DisplayName, &p.Author.Avatar, &p.AuthorDisabled,
		&p.ReviewerID, &rName, &rDisplay, &rAvatar, &p.ReviewComment, &p.CommentCount,
		&p.CreatedAt, &p.UpdatedAt, &p.SubmittedAt, &p.DecidedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(payload) > 0 {
		p.Payload = &model.CatalogExport{}
		if err := json.Unmarshal(payload, p.Payload); err != nil {
			return nil, fmt.Errorf("proposal %s payload: %w", p.ID, err)
		}
	}
	if len(base) > 0 {
		p.Base = &model.ProposalBase{}
		if err := json.Unmarshal(base, p.Base); err != nil {
			return nil, fmt.Errorf("proposal %s base: %w", p.ID, err)
		}
	}
	if rName != nil {
		p.Reviewer = &model.Owner{Username: *rName, DisplayName: rDisplay, Avatar: rAvatar}
	}
	return &p, nil
}

func jsonOrNil(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func (s *PgStore) CreateProposal(ctx context.Context, authorID uuid.UUID, in model.ProposalInput) (*model.Proposal, error) {
	var payload []byte
	if in.Payload != nil {
		var err error
		if payload, err = json.Marshal(in.Payload); err != nil {
			return nil, err
		}
	}
	var id uuid.UUID
	err := s.pool.QueryRow(ctx,
		`INSERT INTO proposals (author_id, kind, title, description, payload, item_id)
		 VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6) RETURNING id`,
		authorID, in.Kind, in.Title, in.Description, payload, in.ItemID).Scan(&id)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.GetProposal(ctx, id)
}

func (s *PgStore) GetProposal(ctx context.Context, id uuid.UUID) (*model.Proposal, error) {
	return scanProposal(s.pool.QueryRow(ctx, `SELECT `+proposalColumns+proposalFrom+` WHERE p.id = $1`, id))
}

func (s *PgStore) ListProposals(ctx context.Context, f model.ProposalFilter, pg model.Pagination) ([]model.Proposal, int, error) {
	where := "TRUE"
	args := []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		where += fmt.Sprintf(" AND "+cond, len(args))
	}
	if f.AuthorID != nil {
		add("p.author_id = $%d", *f.AuthorID)
	}
	if f.ExcludeAuthor != nil {
		add("p.author_id <> $%d", *f.ExcludeAuthor)
	}
	if f.Review {
		where += " AND p.status <> 'draft'"
	}
	if f.Status != "" {
		add("p.status = $%d", f.Status)
	}
	order := "p.created_at DESC"
	if f.Review { // pending first, oldest first; then the decided ones, newest first
		order = "(p.status = 'pending') DESC, CASE WHEN p.status = 'pending' THEN p.submitted_at END ASC, p.updated_at DESC"
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM proposals p WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pg.PerPage, pg.Offset())
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`SELECT %s%s WHERE %s ORDER BY %s, p.id LIMIT $%d OFFSET $%d`,
		proposalColumns, proposalFrom, where, order, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Proposal{}
	for rows.Next() {
		p, err := scanProposal(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *p)
	}
	return out, total, rows.Err()
}

// UpdateProposal changes the content of a draft or of a proposal sent back
// for changes (ErrConflict otherwise).
func (s *PgStore) UpdateProposal(ctx context.Context, id uuid.UUID, in model.ProposalInput) (*model.Proposal, error) {
	payload, err := jsonOrNil(in.Payload)
	if err != nil {
		return nil, err
	}
	if in.Payload == nil {
		payload = nil
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE proposals SET title = $2, description = NULLIF($3, ''), payload = $4, item_id = $5, updated_at = now()
		 WHERE id = $1 AND status IN ('draft', 'changes_requested')`,
		id, in.Title, in.Description, payload, in.ItemID)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrConflict
	}
	return s.GetProposal(ctx, id)
}

// SubmitProposal moves a draft or a proposal sent back to pending, with the
// snapshot of the existing entities it changes.
func (s *PgStore) SubmitProposal(ctx context.Context, id uuid.UUID, base *model.ProposalBase) (*model.Proposal, error) {
	b, err := jsonOrNil(base)
	if err != nil {
		return nil, err
	}
	if base == nil {
		b = nil
	}
	return s.moveProposal(ctx, id, []string{model.ProposalDraft, model.ProposalChangesRequested},
		`status = 'pending', base = $3, submitted_at = now(), decided_at = NULL`, b)
}

// WithdrawProposal: the author takes back a proposal not yet decided.
func (s *PgStore) WithdrawProposal(ctx context.Context, id uuid.UUID) (*model.Proposal, error) {
	return s.moveProposal(ctx, id, []string{model.ProposalDraft, model.ProposalPending, model.ProposalChangesRequested},
		`status = 'withdrawn', decided_at = now()`)
}

// ReviewProposal records a moderator decision on a pending proposal without
// touching the catalog: rejected, changes_requested, or accepted (field
// suggestion, report).
func (s *PgStore) ReviewProposal(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID, comment string) (*model.Proposal, error) {
	decided := "decided_at = now()"
	if status == model.ProposalChangesRequested {
		decided = "decided_at = NULL"
	}
	return s.moveProposal(ctx, id, []string{model.ProposalPending},
		`status = $3, reviewer_id = $4, review_comment = NULLIF($5, ''), `+decided, status, reviewerID, comment)
}

func (s *PgStore) moveProposal(ctx context.Context, id uuid.UUID, from []string, set string, args ...any) (*model.Proposal, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE proposals SET `+set+`, updated_at = now()
		WHERE id = $1 AND status = ANY($2)`, append([]any{id, from}, args...)...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.GetProposal(ctx, id); err != nil {
			return nil, err
		}
		return nil, ErrConflict
	}
	return s.GetProposal(ctx, id)
}

// AcceptCatalogProposal applies payload through the import, in one
// transaction with the status change: both or neither. ctx carries the
// author of the proposal as actor (origin `proposal`). The report lists the
// import errors (nothing applied then).
func (s *PgStore) AcceptCatalogProposal(ctx context.Context, id, reviewerID uuid.UUID, payload *model.CatalogExport, comment string) (*model.ImportReport, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return s.importCatalog(ctx, payload, false, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE proposals SET status = 'accepted', reviewer_id = $2, review_comment = NULLIF($3, ''),
			payload = $4, decided_at = now(), updated_at = now() WHERE id = $1 AND status = 'pending'`,
			id, reviewerID, comment, raw)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrConflict
		}
		return nil
	})
}

// ProposalQuotaUsage counts what limits a member: proposals waiting (pending
// or sent back), accepted ones (they raise the quota), created in 24 h.
func (s *PgStore) ProposalQuotaUsage(ctx context.Context, authorID uuid.UUID) (waiting, accepted, today int, err error) {
	err = s.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE status IN ('pending', 'changes_requested')),
		count(*) FILTER (WHERE status = 'accepted'),
		count(*) FILTER (WHERE created_at > now() - interval '24 hours')
		FROM proposals WHERE author_id = $1`, authorID).Scan(&waiting, &accepted, &today)
	return
}

// CountProposalsToReview counts the pending proposals of others (badge).
func (s *PgStore) CountProposalsToReview(ctx context.Context, reviewerID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM proposals WHERE status = 'pending' AND author_id <> $1`, reviewerID).Scan(&n)
	return n, err
}

// EntitySnapshot returns the current attributes of the given items and
// relations that exist (absent ids: created by the proposal, or deleted).
func (s *PgStore) EntitySnapshot(ctx context.Context, itemIDs, relationIDs []uuid.UUID) (*model.ProposalBase, error) {
	out := &model.ProposalBase{Items: map[uuid.UUID]json.RawMessage{}, Relations: map[uuid.UUID]json.RawMessage{}}
	for _, q := range []struct {
		sql string
		ids []uuid.UUID
		dst map[uuid.UUID]json.RawMessage
	}{
		{`SELECT id, attributes FROM items WHERE id = ANY($1)`, itemIDs, out.Items},
		{`SELECT id, attributes FROM item_relations WHERE id = ANY($1)`, relationIDs, out.Relations},
	} {
		if len(q.ids) == 0 {
			continue
		}
		rows, err := s.pool.Query(ctx, q.sql, q.ids)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id uuid.UUID
			var attrs json.RawMessage
			if err := rows.Scan(&id, &attrs); err != nil {
				rows.Close()
				return nil, err
			}
			q.dst[id] = attrs
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// --- Discussion ---

func (s *PgStore) ListProposalComments(ctx context.Context, proposalID uuid.UUID) ([]model.ProposalComment, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.proposal_id, c.author_id, u.username, u.display_name, u.avatar, c.body, c.created_at
		FROM proposal_comments c JOIN users u ON u.id = c.author_id
		WHERE c.proposal_id = $1 ORDER BY c.created_at, c.id`, proposalID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.ProposalComment, error) {
		var c model.ProposalComment
		err := row.Scan(&c.ID, &c.ProposalID, &c.AuthorID, &c.Author.Username, &c.Author.DisplayName,
			&c.Author.Avatar, &c.Body, &c.CreatedAt)
		return c, err
	})
	if out == nil {
		out = []model.ProposalComment{}
	}
	return out, err
}

func (s *PgStore) CreateProposalComment(ctx context.Context, proposalID, authorID uuid.UUID, body string) (*model.ProposalComment, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx,
		`INSERT INTO proposal_comments (proposal_id, author_id, body) VALUES ($1, $2, $3) RETURNING id`,
		proposalID, authorID, body).Scan(&id)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	comments, err := s.ListProposalComments(ctx, proposalID)
	if err != nil {
		return nil, err
	}
	for i := range comments {
		if comments[i].ID == id {
			return &comments[i], nil
		}
	}
	return nil, ErrNotFound
}
