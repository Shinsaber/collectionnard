package model

import (
	"time"

	"github.com/google/uuid"
)

// Article states, report target and notification kinds (I-3).
const (
	ArticlePending   = "pending"
	ArticlePublished = "published"
	ArticleRejected  = "rejected"

	ReportTargetArticle  = "article"
	CommentTargetArticle = "article"

	NotifyArticlePublished = "article_published"
	NotifyArticleRejected  = "article_rejected"
	NotifyArticleHidden    = "article_hidden"
	NotifyArticleRestored  = "article_restored"
	NotifyArticleComment   = "article_comment"
)

// Article relays a piece of news about an item: a title, a link, a word. The
// article of a member waits for a moderator before it is published.
type Article struct {
	ID       uuid.UUID `json:"id"`
	ItemID   uuid.UUID `json:"item_id"`
	AuthorID uuid.UUID `json:"-"`
	// AuthorProfile: visibility of the author's profile (name shown accordingly).
	AuthorProfile string        `json:"-"`
	Author        *ReviewAuthor `json:"author,omitempty"`
	Mine          bool          `json:"mine"`
	Title         string        `json:"title"`
	URL           string        `json:"url"`
	Message       string        `json:"message"`
	Lang          string        `json:"lang"`
	Status        string        `json:"status"`
	// RejectReason: why a moderator refused it (its author only).
	RejectReason   *string    `json:"reject_reason,omitempty"`
	HiddenAt       *time.Time `json:"hidden_at,omitempty"`
	HiddenReason   *string    `json:"hidden_reason,omitempty"`
	CommentCount   int        `json:"comment_count"`
	CommentsClosed bool       `json:"comments_closed"`
	Item           *Item      `json:"item,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ArticleWrite is the body of POST /items/{id}/articles and PUT /articles/{id}.
type ArticleWrite struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Message string `json:"message"`
	Lang    string `json:"lang"`
}
