package model

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ReviewType is the fourth meta-type (I-0): its JSON Schema lists the criteria
// of a review, each one an integer from 0 to 10. No property: overall rating only.
type ReviewType struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	AttributesSchema json.RawMessage `json:"attributes_schema"`
	// ContextRelationTypeIDs: relations from the reviewed item to the edition a
	// review may name (Game → GameRelease); empty = no edition to choose.
	ContextRelationTypeIDs []uuid.UUID `json:"context_relation_type_ids"`
	// ContextLabel: label of that choice by language ("Joué sur", "Played on").
	ContextLabel map[string]string `json:"context_label,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type ReviewTypeCreate struct {
	Name                   string            `json:"name"`
	Description            *string           `json:"description,omitempty"`
	AttributesSchema       json.RawMessage   `json:"attributes_schema,omitempty"`
	ContextRelationTypeIDs []uuid.UUID       `json:"context_relation_type_ids,omitempty"`
	ContextLabel           map[string]string `json:"context_label,omitempty"`
}

type ReviewTypeUpdate struct {
	Name             *string         `json:"name,omitempty"`
	Description      *string         `json:"description,omitempty"`
	AttributesSchema json.RawMessage `json:"attributes_schema,omitempty"`
	// Absent keeps the stored value; an empty array / object clears it.
	ContextRelationTypeIDs *[]uuid.UUID       `json:"context_relation_type_ids,omitempty"`
	ContextLabel           *map[string]string `json:"context_label,omitempty"`
}

// OptionalUUID tells an absent field (keep the stored value) from an explicit
// null (clear it).
type OptionalUUID struct {
	Set   bool
	Value *uuid.UUID
}

// UnmarshalJSON is called for a present field only, null included.
func (o *OptionalUUID) UnmarshalJSON(raw []byte) error {
	o.Set = true
	o.Value = nil
	if bytes.Equal(raw, []byte("null")) {
		return nil
	}
	var id uuid.UUID
	if err := json.Unmarshal(raw, &id); err != nil {
		return err
	}
	o.Value = &id
	return nil
}

// ReviewAuthor is the public part of the author of a signed review.
type ReviewAuthor struct {
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
	Avatar      *string `json:"avatar,omitempty"`
}

// Review is the opinion of a user on an item. UserID never leaves the server:
// Author is the only link to the person, and stays nil unless the review is
// signed for the reader.
type Review struct {
	ID           uuid.UUID       `json:"id"`
	ItemID       uuid.UUID       `json:"item_id"`
	UserID       uuid.UUID       `json:"-"`
	ReviewTypeID *uuid.UUID      `json:"review_type_id,omitempty"`
	Criteria     json.RawMessage `json:"criteria,omitempty"`
	// ContextItemID: edition used for the review; left out with the author
	// when the review is not signed for the reader.
	ContextItemID *uuid.UUID    `json:"context_item_id,omitempty"`
	ContextItem   *Item         `json:"context_item,omitempty"`
	Rating        float64       `json:"rating"`
	Body          string        `json:"body"`
	Spoiler       bool          `json:"spoiler"`
	Lang          string        `json:"lang"`
	Visibility    string        `json:"visibility"`
	Author        *ReviewAuthor `json:"author,omitempty"`
	Item          *Item         `json:"item,omitempty"`
	// HiddenAt, HiddenReason: hidden by the moderation (I-1); a hidden review
	// reaches only its author and the moderators.
	HiddenAt     *time.Time `json:"hidden_at,omitempty"`
	HiddenReason *string    `json:"hidden_reason,omitempty"`
	// Reactions: count of each kind; MyReactions: those of the reader.
	Reactions   map[string]int `json:"reactions,omitempty"`
	MyReactions []string       `json:"my_reactions,omitempty"`
	// CommentCount, CommentsClosed: the thread of the review (I-2).
	CommentCount   int       `json:"comment_count"`
	CommentsClosed bool      `json:"comments_closed"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ReviewPut is the body of PUT /items/{id}/reviews/mine: an overall rating or
// the rating of each criterion, never both.
type ReviewPut struct {
	Rating   *int            `json:"rating,omitempty"`
	Criteria json.RawMessage `json:"criteria,omitempty"`
	// ContextItemID: optional edition, linked to the item by a context relation of its review type.
	ContextItemID *uuid.UUID `json:"context_item_id,omitempty"`
	Body          string     `json:"body"`
	Spoiler       bool       `json:"spoiler"`
	Lang          string     `json:"lang"`
	Visibility    string     `json:"visibility"`
}

// ReviewWrite is a checked review, as stored (Rating computed by the server).
type ReviewWrite struct {
	ReviewTypeID  *uuid.UUID
	Criteria      json.RawMessage
	ContextItemID *uuid.UUID
	Rating        float64
	Body          string
	Spoiler       bool
	Lang          string
	Visibility    string
}

// ReviewListParams filters and sorts GET /items/{id}/reviews.
type ReviewListParams struct {
	Viewer uuid.UUID // uuid.Nil: anonymous
	Levels []string  // visibilities signed for the reader
	Sort   string    // recent | rating | interesting
	Lang   string
	// ContextItemID: only the reviews of this edition, signed for the reader.
	ContextItemID *uuid.UUID
}

// ReviewCriterionStats is the raw aggregate of a criterion.
type ReviewCriterionStats struct {
	Count int
	Avg   float64
}

// ReviewStats is the raw aggregate of the reviews of an item.
type ReviewStats struct {
	Count    int
	Avg      float64
	NoteOnly int // not listed for the reader (their own review excluded)
	Criteria map[string]ReviewCriterionStats
}

// ReviewCriterionSummary is the average of a criterion (nil below the threshold).
type ReviewCriterionSummary struct {
	Key    string   `json:"key"`
	Count  int      `json:"count"`
	Rating *float64 `json:"rating"`
}

// ReviewSummary is the response of GET /items/{id}/reviews/summary: the same
// figures for every reader, except note_only_count. Rating stays nil below
// MinRatings, so that one note cannot be read in the average.
type ReviewSummary struct {
	Enabled       bool                     `json:"enabled"`
	ReviewType    *ReviewType              `json:"review_type,omitempty"`
	Count         int                      `json:"count"`
	Rating        *float64                 `json:"rating"`
	MinRatings    int                      `json:"min_ratings"`
	NoteOnlyCount int                      `json:"note_only_count"`
	Criteria      []ReviewCriterionSummary `json:"criteria"`
	// ContextItems: editions a review of this item may name (form, filter).
	ContextItems []Item `json:"context_items"`
}
