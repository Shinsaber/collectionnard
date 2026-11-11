package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Loans (P7-3, docs/phases/exemplaires.md): copies of a lot lent to a
// friend or to a mere name, in a collection whose type is marked `x-lendable`.
const (
	LendableKey = "x-lendable"

	NotifyLoanStarted = "loan_started"
	NotifyLoanOverdue = "loan_overdue"

	MaxLoanNote         = 500
	MaxLoanBorrowerName = 100
)

// IsLendable tells whether a CollectionType schema opens loans (root `x-lendable: true`).
func IsLendable(schema json.RawMessage) bool {
	var root map[string]json.RawMessage
	if json.Unmarshal(schema, &root) != nil {
		return false
	}
	var on bool
	return json.Unmarshal(root[LendableKey], &on) == nil && on
}

// Loan is a loan of copies of a lot. Dates are YYYY-MM-DD. Borrower is the
// friend who borrowed (absent for a mere name, or an account gone);
// BorrowerName is always set, frozen at the time of the loan.
type Loan struct {
	ID           uuid.UUID     `json:"id"`
	LotID        uuid.UUID     `json:"lot_id"`
	CollectionID uuid.UUID     `json:"collection_id"`
	Quantity     int           `json:"quantity"`
	BorrowerID   *uuid.UUID    `json:"-"`
	Borrower     *ReviewAuthor `json:"borrower,omitempty"`
	BorrowerName string        `json:"borrower_name"`
	LentAt       string        `json:"lent_at"`
	DueAt        *string       `json:"due_at,omitempty"`
	ReturnedAt   *string       `json:"returned_at,omitempty"`
	// Overdue: not returned and past its due date.
	Overdue   bool      `json:"overdue"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// OwnerID and Owner: the lender, named to the borrower only.
	OwnerID uuid.UUID     `json:"-"`
	Owner   *ReviewAuthor `json:"owner,omitempty"`
	// Item: what was lent, in the lists of /me/loans and /me/borrowings.
	Item *Item `json:"item,omitempty"`
	// CollectionName: the lender's own lists only.
	CollectionName string `json:"collection_name,omitempty"`
}

// LoanCreate is the body of POST …/lots/{lotId}/loans: a friend by username,
// or a mere name, never both.
type LoanCreate struct {
	Quantity         *int    `json:"quantity,omitempty"` // absent: 1
	BorrowerUsername string  `json:"borrower_username,omitempty"`
	BorrowerName     string  `json:"borrower_name,omitempty"`
	LentAt           string  `json:"lent_at,omitempty"` // absent: today
	DueAt            *string `json:"due_at,omitempty"`
	Note             string  `json:"note,omitempty"`
}

// LoanUpdate is the body of PUT /loans/{id}: the editable fields, replaced
// (the borrower does not change). ReturnedAt set closes the loan.
type LoanUpdate struct {
	Quantity   int     `json:"quantity"`
	LentAt     string  `json:"lent_at"`
	DueAt      *string `json:"due_at,omitempty"`
	ReturnedAt *string `json:"returned_at,omitempty"`
	Note       string  `json:"note"`
}

// LoanInput is a validated loan, as the store writes it.
type LoanInput struct {
	Quantity     int
	BorrowerID   *uuid.UUID
	BorrowerName string
	LentAt       string
	DueAt        *string
	ReturnedAt   *string
	Note         string
}
