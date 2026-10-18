package model

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OptionalStrings tells an absent JSON key (Set false) from null (Set true,
// Value nil) and from a list: PUT /collections/{id} `hidden_fields`.
type OptionalStrings struct {
	Set   bool
	Value []string
}

func (o *OptionalStrings) UnmarshalJSON(b []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		o.Value = nil
		return nil
	}
	var v []string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v == nil {
		v = []string{}
	}
	o.Value = v
	return nil
}

// Owner is the owner of a collection as other readers see it.
type Owner struct {
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
	Avatar      *string `json:"avatar,omitempty"`
}

// PublicCollection is a collection read by someone else than its owner: its
// CollectionType schema has lost the hidden fields.
type PublicCollection struct {
	ID             uuid.UUID       `json:"id"`
	Name           string          `json:"name"`
	Description    *string         `json:"description,omitempty"`
	CollectionType *CollectionType `json:"collection_type"`
	ItemCount      int             `json:"item_count"`
	Owner          Owner           `json:"owner"`
	Visibility     string          `json:"visibility"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// PublicProfile is a member profile as other readers see it (no email, no role).
type PublicProfile struct {
	Username          string    `json:"username"`
	DisplayName       *string   `json:"display_name,omitempty"`
	Bio               *string   `json:"bio,omitempty"`
	Avatar            *string   `json:"avatar,omitempty"`
	ProfileVisibility string    `json:"profile_visibility"`
	CreatedAt         time.Time `json:"created_at"`
}

// NewPublicProfile keeps the public part of a user.
func NewPublicProfile(u *User) PublicProfile {
	return PublicProfile{
		Username: u.Username, DisplayName: u.DisplayName, Bio: u.Bio, Avatar: u.Avatar,
		ProfileVisibility: u.ProfileVisibility, CreatedAt: u.CreatedAt,
	}
}

// ShareLink is a secret read-only link to a collection; its token is shown once.
type ShareLink struct {
	ID           uuid.UUID `json:"id"`
	CollectionID uuid.UUID `json:"collection_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// ShareLinkCreated is the response of POST /collections/{id}/share-links.
type ShareLinkCreated struct {
	ShareLink ShareLink `json:"share_link"`
	Token     string    `json:"token"`
}
