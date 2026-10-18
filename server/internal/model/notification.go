package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Notification kinds (P4-3c).
const (
	NotifyProposalAccepted         = "proposal_accepted"
	NotifyProposalRejected         = "proposal_rejected"
	NotifyProposalChangesRequested = "proposal_changes_requested"
	NotifyProposalComment          = "proposal_comment"
)

// Notification is an event for one account.
type Notification struct {
	ID        uuid.UUID       `json:"id"`
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
	ReadAt    *time.Time      `json:"read_at,omitempty"`
}

// NotificationPage is the response of GET /me/notifications.
type NotificationPage struct {
	PaginatedResponse
	Unread int `json:"unread"`
}
