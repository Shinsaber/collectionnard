package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// Loans (P7-3): the copies lent of a lot never exceed its quantity; every
// write that could break it locks the lot (FOR UPDATE), as CreateLoan does.

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// lentCopies sums the loans in progress of a lot, except the loan except.
func lentCopies(ctx context.Context, q rowQuerier, lotID uuid.UUID, except *uuid.UUID) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT COALESCE(sum(quantity), 0) FROM loans
		WHERE lot_id = $1 AND returned_at IS NULL AND ($2::uuid IS NULL OR id <> $2)`, lotID, except).Scan(&n)
	return n, err
}

// loanSelect: a loan with its lot's collection and owner, its borrower when a
// member still active, and its item.
const loanSelect = `SELECT l.id, l.lot_id, ci.collection_id, l.quantity, l.borrower_id,
	bu.username, bu.display_name, bu.avatar, l.borrower_name,
	l.lent_at::text, l.due_at::text, l.returned_at::text, COALESCE(l.returned_at IS NULL AND l.due_at < current_date, false),
	l.note, l.created_at, l.updated_at,
	c.user_id, ou.username, ou.display_name, ou.avatar, c.name,
	i.id, i.item_type_id, i.attributes, it.name, it.attributes_schema
	FROM loans l
	JOIN collection_items ci ON ci.id = l.lot_id
	JOIN collections c ON c.id = ci.collection_id
	JOIN users ou ON ou.id = c.user_id
	JOIN items i ON i.id = ci.item_id
	JOIN item_types it ON it.id = i.item_type_id
	LEFT JOIN users bu ON bu.id = l.borrower_id AND bu.disabled_at IS NULL`

func scanLoan(row pgx.Row) (*model.Loan, error) {
	var l model.Loan
	var bName *string
	var b model.ReviewAuthor
	var o model.ReviewAuthor
	var i model.Item
	var t model.ItemType
	if err := row.Scan(&l.ID, &l.LotID, &l.CollectionID, &l.Quantity, &l.BorrowerID,
		&bName, &b.DisplayName, &b.Avatar, &l.BorrowerName,
		&l.LentAt, &l.DueAt, &l.ReturnedAt, &l.Overdue,
		&l.Note, &l.CreatedAt, &l.UpdatedAt,
		&l.OwnerID, &o.Username, &o.DisplayName, &o.Avatar, &l.CollectionName,
		&i.ID, &i.ItemTypeID, &i.Attributes, &t.Name, &t.AttributesSchema); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if bName != nil {
		b.Username = *bName
		l.Borrower = &b
	}
	l.Owner = &o
	t.ID = i.ItemTypeID
	i.ItemType = &t
	l.Item = &i
	return &l, nil
}

func collectLoans(rows pgx.Rows, err error) ([]model.Loan, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Loan{}
	for rows.Next() {
		l, err := scanLoan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

// CreateLoan lends copies of a lot (ErrInvalidQuantity beyond the copies not lent yet).
func (s *PgStore) CreateLoan(ctx context.Context, lotID uuid.UUID, in model.LoanInput) (*model.Loan, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var quantity int
	err = tx.QueryRow(ctx, `SELECT quantity FROM collection_items WHERE id = $1 FOR UPDATE`, lotID).Scan(&quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	lent, err := lentCopies(ctx, tx, lotID, nil)
	if err != nil {
		return nil, err
	}
	if lent+in.Quantity > quantity {
		return nil, ErrInvalidQuantity
	}
	var id uuid.UUID
	if err := tx.QueryRow(ctx, `INSERT INTO loans (lot_id, quantity, borrower_id, borrower_name, lent_at, due_at, note)
		VALUES ($1, $2, $3, $4, COALESCE($5::date, current_date), $6::date, $7) RETURNING id`,
		lotID, in.Quantity, in.BorrowerID, in.BorrowerName, nullIfEmpty(in.LentAt), in.DueAt, in.Note).Scan(&id); err != nil {
		return nil, err
	}
	l, err := scanLoan(tx.QueryRow(ctx, loanSelect+` WHERE l.id = $1`, id))
	if err != nil {
		return nil, err
	}
	return l, tx.Commit(ctx)
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// GetLoan returns a loan with its lot's owner (the handler checks who reads).
func (s *PgStore) GetLoan(ctx context.Context, id uuid.UUID) (*model.Loan, error) {
	return scanLoan(s.pool.QueryRow(ctx, loanSelect+` WHERE l.id = $1`, id))
}

// UpdateLoan replaces the editable fields of a loan (not its borrower). A
// loan reopened or enlarged must still fit in its lot; a new due date allows a
// new reminder.
func (s *PgStore) UpdateLoan(ctx context.Context, id uuid.UUID, in model.LoanInput) (*model.Loan, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var lotID uuid.UUID
	var quantity int
	err = tx.QueryRow(ctx, `SELECT ci.id, ci.quantity FROM loans l JOIN collection_items ci ON ci.id = l.lot_id
		WHERE l.id = $1 FOR UPDATE OF ci`, id).Scan(&lotID, &quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if in.ReturnedAt == nil {
		lent, err := lentCopies(ctx, tx, lotID, &id)
		if err != nil {
			return nil, err
		}
		if lent+in.Quantity > quantity {
			return nil, ErrInvalidQuantity
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE loans SET quantity = $2, lent_at = $3::date, due_at = $4::date, returned_at = $5::date,
			note = $6, reminded_at = CASE WHEN due_at IS DISTINCT FROM $4::date THEN NULL ELSE reminded_at END,
			updated_at = now()
		WHERE id = $1`, id, in.Quantity, in.LentAt, in.DueAt, in.ReturnedAt, in.Note); err != nil {
		return nil, err
	}
	l, err := scanLoan(tx.QueryRow(ctx, loanSelect+` WHERE l.id = $1`, id))
	if err != nil {
		return nil, err
	}
	return l, tx.Commit(ctx)
}

// DeleteLoan removes a loan (entered by mistake: a returned loan stays as history).
func (s *PgStore) DeleteLoan(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM loans WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListLotLoans lists the loans of a lot: in progress first, then the history, latest first.
func (s *PgStore) ListLotLoans(ctx context.Context, lotID uuid.UUID) ([]model.Loan, error) {
	return collectLoans(s.pool.Query(ctx, loanSelect+` WHERE l.lot_id = $1
		ORDER BY l.returned_at IS NOT NULL, l.lent_at DESC, l.created_at DESC`, lotID))
}

// ActiveLoans returns the loans in progress of the given lots, by lot.
func (s *PgStore) ActiveLoans(ctx context.Context, lotIDs []uuid.UUID) (map[uuid.UUID][]model.Loan, error) {
	out := map[uuid.UUID][]model.Loan{}
	if len(lotIDs) == 0 {
		return out, nil
	}
	list, err := collectLoans(s.pool.Query(ctx, loanSelect+` WHERE l.lot_id = ANY($1) AND l.returned_at IS NULL
		ORDER BY l.lent_at, l.created_at`, lotIDs))
	if err != nil {
		return nil, err
	}
	for _, l := range list {
		out[l.LotID] = append(out[l.LotID], l)
	}
	return out, nil
}

// loanStatus: "active" (in progress), "returned", or "" (all).
func loanStatus(status string) string {
	switch status {
	case "active":
		return ` AND l.returned_at IS NULL`
	case "returned":
		return ` AND l.returned_at IS NOT NULL`
	}
	return ``
}

func (s *PgStore) listLoansWhere(ctx context.Context, where string, userID uuid.UUID, pg model.Pagination) ([]model.Loan, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM (`+loanSelect+where+`) t`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	list, err := collectLoans(s.pool.Query(ctx, loanSelect+where+`
		ORDER BY l.returned_at IS NOT NULL, l.due_at NULLS LAST, l.lent_at DESC, l.id LIMIT $2 OFFSET $3`,
		userID, pg.PerPage, pg.Offset()))
	return list, total, err
}

// ListLoans lists the loans of ownerID's lots: in progress first, the closest due date first.
func (s *PgStore) ListLoans(ctx context.Context, ownerID uuid.UUID, status string, pg model.Pagination) ([]model.Loan, int, error) {
	return s.listLoansWhere(ctx, ` WHERE c.user_id = $1`+loanStatus(status), ownerID, pg)
}

// ListBorrowings lists what borrowerID borrowed, lenders of disabled accounts excepted.
func (s *PgStore) ListBorrowings(ctx context.Context, borrowerID uuid.UUID, status string, pg model.Pagination) ([]model.Loan, int, error) {
	return s.listLoansWhere(ctx, ` WHERE l.borrower_id = $1 AND ou.disabled_at IS NULL`+loanStatus(status), borrowerID, pg)
}

// OverdueLoans lists the loans past their due date not reminded yet.
func (s *PgStore) OverdueLoans(ctx context.Context) ([]model.Loan, error) {
	return collectLoans(s.pool.Query(ctx, loanSelect+` WHERE l.returned_at IS NULL AND l.due_at < current_date
		AND l.reminded_at IS NULL AND ou.disabled_at IS NULL ORDER BY l.due_at`))
}

// MarkLoansReminded records the reminder of the current due date.
func (s *PgStore) MarkLoansReminded(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE loans SET reminded_at = now() WHERE id = ANY($1)`, ids)
	return err
}

// GetCollectionLot returns a lot of a collection.
func (s *PgStore) GetCollectionLot(ctx context.Context, collectionID, lotID uuid.UUID) (*model.CollectionItem, error) {
	return scanLot(s.pool.QueryRow(ctx, `SELECT `+lotColumns+` FROM collection_items WHERE collection_id = $1 AND id = $2`,
		collectionID, lotID))
}
