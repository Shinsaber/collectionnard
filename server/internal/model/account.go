package model

import (
	"time"

	"github.com/google/uuid"
)

// MinPasswordLength is the minimum length of a password (create-user, register, reset, change).
const MinPasswordLength = 8

// Roles of an account.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// UserUpdate is the change of an account by an admin (PUT /users/{id}); nil = unchanged.
type UserUpdate struct {
	Role     *string `json:"role,omitempty"`
	Disabled *bool   `json:"disabled,omitempty"`
}

// ProfileUpdate replaces the profile of the current user (PUT /me/profile); "" clears a field.
type ProfileUpdate struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	Avatar      string `json:"avatar"`
	// ProfileVisibility: empty = private.
	ProfileVisibility string `json:"profile_visibility"`
}

// PasswordChange is the body of PUT /me/password.
type PasswordChange struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// Invitation states (Invitation.Status).
const (
	InvitationPending = "pending"
	InvitationUsed    = "used"
	InvitationExpired = "expired"
)

// Invitation lets someone create an account (social.md, P4-0). Its token is
// only returned once, at creation (InvitationCreated).
type Invitation struct {
	ID                uuid.UUID  `json:"id"`
	Role              string     `json:"role"`
	Note              *string    `json:"note,omitempty"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	CreatedByUsername string     `json:"created_by_username,omitempty"`
	ExpiresAt         time.Time  `json:"expires_at"`
	Status            string     `json:"status"`
	UsedAt            *time.Time `json:"used_at,omitempty"`
	UsedByUsername    string     `json:"used_by_username,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// InvitationCreate is the body of POST /invitations.
type InvitationCreate struct {
	Role          string `json:"role"`
	Note          string `json:"note"`
	ExpiresInDays int    `json:"expires_in_days"`
}

// InvitationCreated answers POST /invitations: the token is shown this once.
type InvitationCreated struct {
	Invitation Invitation `json:"invitation"`
	Token      string     `json:"token"`
}

// InvitationCheck answers GET /auth/invitation.
type InvitationCheck struct {
	Valid     bool       `json:"valid"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// RegisterRequest is the body of POST /auth/register.
type RegisterRequest struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ResetLink answers POST /users/{id}/reset-link: the token is shown this once.
type ResetLink struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ResetPasswordRequest is the body of POST /auth/reset-password.
type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}
