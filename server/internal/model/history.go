package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RoleContributor writes directly in the catalogue (no item deletion) and
// moderates proposals (P4-3).
const RoleContributor = "contributor"

// CanEditCatalog: admins and contributors write in the catalogue.
func CanEditCatalog(role string) bool {
	return role == RoleAdmin || role == RoleContributor
}

// History author kinds (HistoryAuthor.Kind).
const (
	AuthorUser   = "user"   // profile visible to the reader
	AuthorMember = "member" // profile hidden from the reader, or account disabled
	AuthorSystem = "system" // seed, SQL
)

// HistoryAuthor is resolved at read time from the current profile visibility.
type HistoryAuthor struct {
	Kind string `json:"kind"`
	User *Owner `json:"user,omitempty"`
}

// HistoryRelation identifies the relation of a `relation` entry.
type HistoryRelation struct {
	RelationTypeID uuid.UUID `json:"relation_type_id"`
	SourceItemID   uuid.UUID `json:"source_item_id"`
	TargetItemID   uuid.UUID `json:"target_item_id"`
}

// HistoryEntry is one write of an item or of one of its relations.
type HistoryEntry struct {
	Kind      string           `json:"kind"` // item | relation
	EntityID  uuid.UUID        `json:"entity_id"`
	Version   int              `json:"version"`
	Origin    string           `json:"origin"`
	OriginRef *string          `json:"origin_ref,omitempty"`
	UserID    *uuid.UUID       `json:"-"`
	Author    HistoryAuthor    `json:"author"`
	Before    json.RawMessage  `json:"before"`
	After     json.RawMessage  `json:"after"`
	Relation  *HistoryRelation `json:"relation,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// MaxHistory caps ItemHistory.Entries.
const MaxHistory = 500

// ItemHistory is the response of GET /items/{id}/history.
type ItemHistory struct {
	Entries   []HistoryEntry `json:"entries"`
	Truncated bool           `json:"truncated"`
	Created   *HistoryEntry  `json:"created,omitempty"`
}

// RevertRequest is the body of POST /items/{id}/revert.
type RevertRequest struct {
	Version int `json:"version"`
}
