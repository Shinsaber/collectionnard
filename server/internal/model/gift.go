package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Gift reservations (P7-4, docs/phases/exemplaires-prets.md): a friend of the
// owner of a wish list (type marked `x-reservable`) reserves a share of a lot
// to give it. Never returned to the owner.
const (
	ReservableKey = "x-reservable"

	GiftReserved = "reserved"
	GiftGiven    = "given"

	NotifyGiftLeftList = "gift_left_list"
	NotifyGiftExpiring = "gift_expiring"

	// A reservation not given lasts GiftDefaultDays unless told otherwise, at
	// most GiftMaxDays from today; its giver is reminded GiftExpiryNoticeDays before.
	GiftDefaultDays      = 60
	GiftMaxDays          = 365
	GiftExpiryNoticeDays = 7
)

// IsReservable tells whether a CollectionType schema opens gift reservations (root `x-reservable: true`).
func IsReservable(schema json.RawMessage) bool {
	var root map[string]json.RawMessage
	if json.Unmarshal(schema, &root) != nil {
		return false
	}
	var on bool
	return json.Unmarshal(root[ReservableKey], &on) == nil && on
}

// GiftReservation is a share of a lot reserved by a giver. Seen by the
// owner's friends (giver named, for them to agree), never by the owner; the
// note is the giver's own.
type GiftReservation struct {
	ID uuid.UUID `json:"id"`
	// LotID: absent once the lot left the list (LeftList).
	LotID    *uuid.UUID `json:"lot_id,omitempty"`
	LeftList bool       `json:"left_list"`
	ItemID   uuid.UUID  `json:"item_id"`
	Quantity int        `json:"quantity"`
	Status   string     `json:"status"`
	// ExpiresAt: last day (YYYY-MM-DD) a reservation not given holds; past it,
	// Expired: it no longer counts and only its giver sees it.
	ExpiresAt string `json:"expires_at"`
	Expired   bool   `json:"expired"`
	// Mine: reserved by the reader.
	Mine      bool          `json:"mine"`
	GiverID   uuid.UUID     `json:"-"`
	Giver     *ReviewAuthor `json:"giver,omitempty"`
	Note      string        `json:"note,omitempty"`
	OwnerID   uuid.UUID     `json:"-"`
	Owner     *ReviewAuthor `json:"owner,omitempty"`
	Item      *Item         `json:"item,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// ReservationCreate is the body of POST …/lots/{lotId}/reservations.
type ReservationCreate struct {
	Quantity  *int   `json:"quantity,omitempty"`   // absent: 1
	ExpiresAt string `json:"expires_at,omitempty"` // absent: in GiftDefaultDays
	Note      string `json:"note,omitempty"`
}

// ReservationUpdate is the body of PUT /reservations/{id}: fields replaced.
type ReservationUpdate struct {
	Quantity  int    `json:"quantity"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at,omitempty"` // absent: unchanged
	Note      string `json:"note"`
}
