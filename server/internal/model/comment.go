package model

import (
	"time"

	"github.com/google/uuid"
)

// Comment targets and notification kinds (I-2).
const (
	CommentTargetReview  = "review"
	ReportTargetComment  = "comment"
	NotifyReviewComment  = "review_comment"
	NotifyCommentHidden  = "comment_hidden"
	NotifyCommentRestore = "comment_restored"
)

// Comment is a message of the thread of a review (later of an article). Its
// author is shown, except the author of an anonymous review: TargetAuthor
// then tells it is them, without their name.
type Comment struct {
	ID         uuid.UUID `json:"id"`
	TargetKind string    `json:"target_kind"`
	TargetID   uuid.UUID `json:"target_id"`
	AuthorID   uuid.UUID `json:"-"`
	// AuthorProfile: visibility of the author's profile; their name reaches
	// only the readers who may see that profile.
	AuthorProfile string        `json:"-"`
	Author        *ReviewAuthor `json:"author,omitempty"`
	// TargetAuthor: written by the author of the review or article commented.
	TargetAuthor bool `json:"target_author"`
	// Mine: written by the reader (edit, delete).
	Mine     bool       `json:"mine"`
	Body     string     `json:"body"`
	HiddenAt *time.Time `json:"hidden_at,omitempty"`
	// DeletedAt: deleted by its author, a trace without text nor author.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	// EditedAt: changed after it was posted.
	EditedAt     *time.Time `json:"edited_at,omitempty"`
	HiddenReason *string    `json:"hidden_reason,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CommentBody is the body of POST /reviews/{id}/comments and PUT /comments/{id}.
type CommentBody struct {
	Body string `json:"body"`
}
