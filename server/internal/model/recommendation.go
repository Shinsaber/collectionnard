package model

import (
	"time"

	"github.com/google/uuid"
)

// Notification kind of a recommendation (I-4).
const NotifyRecommendation = "recommendation"

// Recommendation is an item sent by a member to another, with a short word.
// Only its recipient reads it; the sender is named to them.
type Recommendation struct {
	ID        uuid.UUID     `json:"id"`
	ItemID    uuid.UUID     `json:"item_id"`
	Item      *Item         `json:"item,omitempty"`
	SenderID  uuid.UUID     `json:"-"`
	Sender    *ReviewAuthor `json:"sender,omitempty"`
	Message   string        `json:"message"`
	CreatedAt time.Time     `json:"created_at"`
	ReadAt    *time.Time    `json:"read_at,omitempty"`
}

// RecommendationCreate is the body of POST /items/{id}/recommendations.
type RecommendationCreate struct {
	// To: username of the recipient.
	To      string `json:"to"`
	Message string `json:"message"`
}

// RecommendationPage is the response of GET /me/recommendations.
type RecommendationPage struct {
	PaginatedResponse
	Unread int `json:"unread"`
}

// Block: the blocker no longer receives anything from the blocked member and
// no longer sees their contents. User is withheld when the blocker may not see
// the blocked member's profile (blocked from a comment signed « Un membre »).
type Block struct {
	ID        uuid.UUID `json:"id"`
	BlockedID uuid.UUID `json:"-"`
	// BlockedProfile: visibility of the blocked member's profile.
	BlockedProfile string `json:"-"`
	// Anonymous: blocked from a content that did not name them; never named.
	Anonymous bool          `json:"-"`
	User      *ReviewAuthor `json:"user,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}

// BlockCreate is the body of POST /me/blocks: a username, or a content whose
// author is blocked (review, comment, article or received recommendation).
type BlockCreate struct {
	Username    string     `json:"username,omitempty"`
	ContentKind string     `json:"content_kind,omitempty"`
	ContentID   *uuid.UUID `json:"content_id,omitempty"`
}

// Block content kinds besides those of the reports.
const BlockFromRecommendation = "recommendation"
