package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Proposal kinds and states (P4-3b).
const (
	ProposalCatalog          = "catalog"
	ProposalSchemaSuggestion = "schema_suggestion"
	ProposalReport           = "report"

	ProposalDraft            = "draft"
	ProposalPending          = "pending"
	ProposalChangesRequested = "changes_requested"
	ProposalAccepted         = "accepted"
	ProposalRejected         = "rejected"
	ProposalWithdrawn        = "withdrawn"
)

// ProposalBase is the snapshot of the existing entities a catalog proposal
// changes, taken at submission (id → attributes).
type ProposalBase struct {
	Items     map[uuid.UUID]json.RawMessage `json:"items"`
	Relations map[uuid.UUID]json.RawMessage `json:"relations"`
}

// Proposal is a member's proposal: catalog change, field suggestion or report.
type Proposal struct {
	ID             uuid.UUID      `json:"id"`
	Kind           string         `json:"kind"`
	Status         string         `json:"status"`
	Title          string         `json:"title"`
	Description    *string        `json:"description,omitempty"`
	Payload        *CatalogExport `json:"payload,omitempty"`
	Base           *ProposalBase  `json:"base,omitempty"`
	ItemID         *uuid.UUID     `json:"item_id,omitempty"`
	AuthorID       uuid.UUID      `json:"-"`
	Author         Owner          `json:"author"`
	AuthorDisabled bool           `json:"author_disabled,omitempty"`
	ReviewerID     *uuid.UUID     `json:"-"`
	Reviewer       *Owner         `json:"reviewer,omitempty"`
	ReviewComment  *string        `json:"review_comment,omitempty"`
	CommentCount   int            `json:"comment_count"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	SubmittedAt    *time.Time     `json:"submitted_at,omitempty"`
	DecidedAt      *time.Time     `json:"decided_at,omitempty"`
}

// Final tells whether the proposal can no longer change.
func (p *Proposal) Final() bool {
	return p.Status == ProposalAccepted || p.Status == ProposalRejected || p.Status == ProposalWithdrawn
}

// ProposalInput creates (Kind, Submit) or updates a proposal.
type ProposalInput struct {
	Kind        string         `json:"kind"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Payload     *CatalogExport `json:"payload"`
	ItemID      *uuid.UUID     `json:"item_id"`
	Submit      bool           `json:"submit"`
}

// ProposalDecision is the body of reject / request-changes.
type ProposalDecision struct {
	Comment string `json:"comment"`
}

// ProposalAccept is the body of accept.
type ProposalAccept struct {
	Payload           *CatalogExport `json:"payload"`
	ConflictsResolved bool           `json:"conflicts_resolved"`
	Comment           string         `json:"comment"`
}

// ProposalConflict: an entity changed since the proposal was submitted.
type ProposalConflict struct {
	Kind     string          `json:"kind"` // item | relation
	ID       uuid.UUID       `json:"id"`
	Base     json.RawMessage `json:"base"`
	Current  json.RawMessage `json:"current"`
	Proposed json.RawMessage `json:"proposed"`
}

// ProposalComment is one message of a proposal discussion.
type ProposalComment struct {
	ID         uuid.UUID `json:"id"`
	ProposalID uuid.UUID `json:"-"`
	AuthorID   uuid.UUID `json:"-"`
	Author     Owner     `json:"author"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// ProposalFilter selects proposals: an author (mine) or the review queue.
type ProposalFilter struct {
	AuthorID      *uuid.UUID
	ExcludeAuthor *uuid.UUID // review queue: never one's own
	Review        bool       // no drafts, pending first
	Status        string
}

// DuplicateCheck is the body of POST /items/duplicates.
type DuplicateCheck struct {
	ItemTypeID uuid.UUID       `json:"item_type_id"`
	Attributes json.RawMessage `json:"attributes"`
	ExcludeID  *uuid.UUID      `json:"exclude_id"`
}

// Duplicate is an existing item carrying the same identifier value.
type Duplicate struct {
	Field string `json:"field"`
	Value string `json:"value"`
	Item  Item   `json:"item"`
}
