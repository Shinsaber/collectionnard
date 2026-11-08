package model

import (
	"time"

	"github.com/google/uuid"
)

// Friends (P7-2, docs/phases/exemplaires-prets.md): a reciprocal relation, a
// request then its acceptance, that opens the `friends` visibility.
const (
	FriendPending  = "pending"
	FriendAccepted = "accepted"

	NotifyFriendRequest  = "friend_request"
	NotifyFriendAccepted = "friend_accepted"
)

// Friendship is a friendship or a request, as seen by one of the two members:
// User is the other one.
type Friendship struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
	// Incoming: a request received (false: sent, or a friendship).
	Incoming bool          `json:"incoming"`
	User     *ReviewAuthor `json:"user"`
	// ProfileVisible: the reader may open the other member's public profile.
	ProfileVisible bool `json:"profile_visible"`
	// OtherID and OtherProfile: the other member and the visibility of their
	// profile, to decide what the reader may know of a member who did not
	// accept them yet.
	OtherID      uuid.UUID  `json:"-"`
	OtherProfile string     `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`
}

// FriendRequestCreate is the body of POST /me/friends.
type FriendRequestCreate struct {
	Username string `json:"username"`
}

// FriendLink is a secret, single-use link a member hands out to become friends
// with whoever opens it; its token is shown once, at creation.
type FriendLink struct {
	ID        uuid.UUID `json:"id"`
	Token     string    `json:"token,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	// OwnerID and Owner: the member who created it; Owner is shown to the
	// one who opens it.
	OwnerID uuid.UUID     `json:"-"`
	Owner   *ReviewAuthor `json:"user,omitempty"`
}
