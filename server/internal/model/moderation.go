package model

import (
	"time"

	"github.com/google/uuid"
)

// RoleModerator handles the reports with the admins (I-1); it does not write
// in the catalogue (one role per account).
const RoleModerator = "moderator"

// CanModerate: admins and moderators handle the reports.
func CanModerate(role string) bool {
	return role == RoleAdmin || role == RoleModerator
}

// Report targets, reasons, states and moderation actions (I-1).
const (
	ReportTargetReview = "review"

	ReportOpen      = "open"
	ReportActioned  = "actioned"
	ReportDismissed = "dismissed"

	ModerationHide    = "hide"
	ModerationRestore = "restore"
	ModerationDismiss = "dismiss"
)

// ReportReasons are the reasons a report may give.
var ReportReasons = []string{"abusive", "harassment", "spoiler", "spam", "off_topic", "other"}

// ReactionKinds are the reactions to a review: an intention or an emotion,
// never a report.
var ReactionKinds = []string{"interesting", "tempting", "funny", "disapprove"}

// Notification kinds of the moderation.
const (
	NotifyReviewHidden   = "review_hidden"
	NotifyReviewRestored = "review_restored"
)

// ReportCreate is the body of POST /reports.
type ReportCreate struct {
	TargetKind string    `json:"target_kind"`
	TargetID   uuid.UUID `json:"target_id"`
	Reason     string    `json:"reason"`
	Details    string    `json:"details"`
}

// Report is a report as the moderators see it.
type Report struct {
	ID         uuid.UUID     `json:"id"`
	TargetKind string        `json:"target_kind"`
	TargetID   uuid.UUID     `json:"target_id"`
	Reason     string        `json:"reason"`
	Details    string        `json:"details"`
	Status     string        `json:"status"`
	Reporter   *ReviewAuthor `json:"reporter,omitempty"`
	ReporterID uuid.UUID     `json:"reporter_id"`
	// ReporterDismissed: reports of the same reporter dismissed in the last 90 days.
	ReporterDismissed int       `json:"reporter_dismissed"`
	CreatedAt         time.Time `json:"created_at"`
}

// ModerationRecap sums up how often a member was moderated, for the moderators:
// no sanction follows from it automatically.
type ModerationRecap struct {
	Hidden90d    int `json:"hidden_90d"`
	HiddenTotal  int `json:"hidden_total"`
	Dismissed90d int `json:"dismissed_reports_90d"`
}

// ModerationItem is a target of the queue with its open reports. The review is
// shown whole (author of an anonymous review included) to the moderators only.
type ModerationItem struct {
	TargetKind string    `json:"target_kind"`
	TargetID   uuid.UUID `json:"target_id"`
	AuthorID   uuid.UUID `json:"author_id"`
	Review     *Review   `json:"review,omitempty"`
	// Comment: the reported comment (target_kind comment), Review being its thread's review.
	Comment *Comment `json:"comment,omitempty"`
	// Article: the reported article (target_kind article).
	Article *Article `json:"article,omitempty"`
	// ReviewType: labels of its criteria.
	ReviewType  *ReviewType     `json:"review_type,omitempty"`
	Item        *Item           `json:"item,omitempty"`
	Reports     []Report        `json:"reports"`
	AuthorRecap ModerationRecap `json:"author_recap"`
}

// ModerationActionBody is the body of a moderation action.
type ModerationActionBody struct {
	Reason string `json:"reason"`
}

// ModerationResult tells the handler whom to notify.
type ModerationResult struct {
	AuthorID uuid.UUID
	ItemID   uuid.UUID
}

// ModerationLogEntry is one line of the journal.
type ModerationLogEntry struct {
	ID           uuid.UUID     `json:"id"`
	Action       string        `json:"action"`
	TargetKind   string        `json:"target_kind"`
	TargetID     uuid.UUID     `json:"target_id"`
	TargetAuthor *ReviewAuthor `json:"target_author,omitempty"`
	AuthorID     *uuid.UUID    `json:"author_id,omitempty"`
	Moderator    *ReviewAuthor `json:"moderator,omitempty"`
	Reason       string        `json:"reason"`
	Excerpt      string        `json:"excerpt"`
	// StillHidden: the target is hidden now (a hide may be undone).
	StillHidden bool      `json:"still_hidden"`
	CreatedAt   time.Time `json:"created_at"`
}

// ModerationHistory is the record of a member: what was moderated, and the
// reports they made that were dismissed.
type ModerationHistory struct {
	User      ReviewAuthor         `json:"user"`
	Recap     ModerationRecap      `json:"recap"`
	Entries   []ModerationLogEntry `json:"entries"`
	Dismissed []Report             `json:"dismissed_reports"`
}
