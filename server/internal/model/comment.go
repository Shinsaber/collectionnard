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
// author is shown, except the author of an anonymous review: ReviewAuthor
// then tells it is them, without their name.
type Comment struct {
	ID         uuid.UUID     `json:"id"`
	TargetKind string        `json:"target_kind"`
	TargetID   uuid.UUID     `json:"target_id"`
	AuthorID   uuid.UUID     `json:"-"`
	Author     *ReviewAuthor `json:"author,omitempty"`
	// ReviewAuthor: written by the author of the review commented.
	ReviewAuthor bool `json:"review_author"`
	// Mine: written by the reader (edit, delete).
	Mine         bool       `json:"mine"`
	Body         string     `json:"body"`
	HiddenAt     *time.Time `json:"hidden_at,omitempty"`
	HiddenReason *string    `json:"hidden_reason,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CommentBody is the body of POST /reviews/{id}/comments and PUT /comments/{id}.
type CommentBody struct {
	Body string `json:"body"`
}
