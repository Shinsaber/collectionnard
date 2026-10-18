package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
	"github.com/shinsaber/colec/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// Accounts, invitations and profiles (Phase 4, P4-0 — social.md).

const (
	defaultInvitationDays = 7
	maxInvitationDays     = 90
	resetLinkTTL          = 24 * time.Hour
	maxDisplayName        = 100
	maxBio                = 2000
	maxNote               = 255
	// bcrypt ignores the bytes past 72 (and GenerateFromPassword refuses them).
	maxPasswordBytes = 72
)

var (
	usernameRE = regexp.MustCompile(`^[A-Za-z0-9_.-]{2,100}$`)
	avatarRE   = regexp.MustCompile(`^/api/v1/uploads/[^/]+$`)
)

// newToken returns a random URL-safe token (invitation, reset link).
func newToken() (string, error) {
	raw := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// checkPassword validates a new password; "" when it is acceptable.
func checkPassword(p string) string {
	switch {
	case utf8.RuneCountInString(p) < model.MinPasswordLength:
		return "password must be at least 8 characters long"
	case len(p) > maxPasswordBytes:
		return "password must be at most 72 bytes long"
	}
	return ""
}

func validEmail(e string) bool {
	at := strings.IndexByte(e, '@')
	return at > 0 && at < len(e)-1 && len(e) <= 255 && !strings.ContainsAny(e, " \t\r\n")
}

// canInvite: admins always, members when MEMBER_INVITES is on.
func (h *Handler) canInvite(r *http.Request) bool {
	return middleware.GetRole(r.Context()) == model.RoleAdmin || h.Config.MemberInvites
}

// ── Public (rate limited like /auth/login) ──────────────────────────

// CheckInvitation answers GET /auth/invitation?token=: is the invitation still usable?
func (h *Handler) CheckInvitation(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondJSON(w, http.StatusOK, model.InvitationCheck{Valid: false})
		return
	}
	exp, err := h.Store.CheckInvitation(r.Context(), hashToken(token))
	if errors.Is(err, store.ErrNotFound) {
		respondJSON(w, http.StatusOK, model.InvitationCheck{Valid: false})
		return
	}
	if err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, model.InvitationCheck{Valid: true, ExpiresAt: &exp})
}

// Register answers POST /auth/register: an account from an invitation, then signed in.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondDecodeError(w, err)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	switch {
	case req.Token == "":
		respondError(w, http.StatusBadRequest, "bad_request", "token is required")
		return
	case !usernameRE.MatchString(req.Username):
		respondError(w, http.StatusBadRequest, "invalid_username", "username: 2 to 100 letters, digits, « _ », « . » or « - »")
		return
	case !validEmail(req.Email):
		respondError(w, http.StatusBadRequest, "invalid_email", "invalid email address")
		return
	}
	if msg := checkPassword(req.Password); msg != "" {
		respondError(w, http.StatusBadRequest, "weak_password", msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}
	user, err := h.Store.RegisterWithInvitation(r.Context(), hashToken(req.Token), req.Username, req.Email, string(hash))
	switch {
	case errors.Is(err, store.ErrNotFound):
		respondError(w, http.StatusUnauthorized, "invalid_invitation", "invitation unknown, used or expired")
		return
	case errors.Is(err, store.ErrConflict):
		respondError(w, http.StatusConflict, "account_exists", "username or email already in use")
		return
	case err != nil:
		respondInternal(w, r, err, "internal error")
		return
	}
	h.issueTokens(w, r, user)
}

// ResetPassword answers POST /auth/reset-password: a new password from a reset link.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req model.ResetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondDecodeError(w, err)
		return
	}
	if req.Token == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "token is required")
		return
	}
	if msg := checkPassword(req.Password); msg != "" {
		respondError(w, http.StatusBadRequest, "weak_password", msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}
	err = h.Store.ResetPassword(r.Context(), hashToken(req.Token), string(hash))
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusUnauthorized, "invalid_reset_link", "reset link unknown, used or expired")
		return
	}
	if err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Current user ────────────────────────────────────────────────────

// UpdateProfile answers PUT /me/profile.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var in model.ProfileUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Bio = strings.TrimSpace(in.Bio)
	switch {
	case utf8.RuneCountInString(in.DisplayName) > maxDisplayName:
		respondError(w, http.StatusBadRequest, "bad_request", "display_name: at most 100 characters")
		return
	case utf8.RuneCountInString(in.Bio) > maxBio:
		respondError(w, http.StatusBadRequest, "bad_request", "bio: at most 2000 characters")
		return
	case in.Avatar != "" && (!avatarRE.MatchString(in.Avatar) || strings.Contains(in.Avatar, "..")):
		respondError(w, http.StatusBadRequest, "bad_request", "avatar must be the url of an upload (/api/v1/uploads/…)")
		return
	case in.ProfileVisibility != "" && !privacy.Valid(in.ProfileVisibility):
		respondError(w, http.StatusBadRequest, "bad_request", "profile_visibility: private, members or public")
		return
	}
	user, err := h.Store.UpdateProfile(r.Context(), middleware.GetUserID(r.Context()), in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, user)
}

// ChangePassword answers PUT /me/password: all sessions are revoked, the current one gets new tokens.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var in model.PasswordChange
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if msg := checkPassword(in.NewPassword); msg != "" {
		respondError(w, http.StatusBadRequest, "weak_password", msg)
		return
	}
	id := middleware.GetUserID(r.Context())
	current, err := h.Store.GetUserPasswordHash(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(current), []byte(in.CurrentPassword)) != nil {
		respondError(w, http.StatusForbidden, "wrong_password", "current password is incorrect")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}
	if err := h.Store.SetPassword(r.Context(), id, string(hash)); err != nil {
		handleStoreError(w, r, err)
		return
	}
	user, err := h.Store.GetUserByID(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	h.issueTokens(w, r, user)
}

// ── Invitations ─────────────────────────────────────────────────────

// ListInvitations answers GET /invitations: mine, or all for an admin.
func (h *Handler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	if !h.canInvite(r) {
		respondError(w, http.StatusForbidden, "forbidden", "invitations are reserved to admins")
		return
	}
	var by *uuid.UUID
	if middleware.GetRole(r.Context()) != model.RoleAdmin {
		id := middleware.GetUserID(r.Context())
		by = &id
	}
	list, err := h.Store.ListInvitations(r.Context(), by)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, list)
}

// CreateInvitation answers POST /invitations: the token is returned this once.
func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	if !h.canInvite(r) {
		respondError(w, http.StatusForbidden, "forbidden", "invitations are reserved to admins")
		return
	}
	var in model.InvitationCreate
	if r.ContentLength != 0 {
		if err := decodeJSON(w, r, &in); err != nil {
			respondDecodeError(w, err)
			return
		}
	}
	if in.Role == "" {
		in.Role = model.RoleUser
	}
	if in.ExpiresInDays == 0 {
		in.ExpiresInDays = defaultInvitationDays
	}
	in.Note = strings.TrimSpace(in.Note)
	switch {
	case in.Role != model.RoleUser && in.Role != model.RoleAdmin:
		respondError(w, http.StatusBadRequest, "bad_request", "role must be admin or user")
		return
	case in.Role == model.RoleAdmin && middleware.GetRole(r.Context()) != model.RoleAdmin:
		respondError(w, http.StatusForbidden, "forbidden", "only an admin may invite an admin")
		return
	case in.ExpiresInDays < 1 || in.ExpiresInDays > maxInvitationDays:
		respondError(w, http.StatusBadRequest, "bad_request", "expires_in_days: 1 to 90")
		return
	case utf8.RuneCountInString(in.Note) > maxNote:
		respondError(w, http.StatusBadRequest, "bad_request", "note: at most 255 characters")
		return
	}
	token, err := newToken()
	if err != nil {
		respondInternal(w, r, err, "token generation failed")
		return
	}
	inv, err := h.Store.CreateInvitation(r.Context(), hashToken(token), middleware.GetUserID(r.Context()), in.Role, in.Note,
		time.Now().Add(time.Duration(in.ExpiresInDays)*24*time.Hour))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, model.InvitationCreated{Invitation: *inv, Token: token})
}

// DeleteInvitation answers DELETE /invitations/{id}: its author or an admin.
func (h *Handler) DeleteInvitation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	inv, err := h.Store.GetInvitation(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	me := middleware.GetUserID(r.Context())
	if middleware.GetRole(r.Context()) != model.RoleAdmin && (inv.CreatedBy == nil || *inv.CreatedBy != me) {
		respondError(w, http.StatusForbidden, "forbidden", "not your invitation")
		return
	}
	if err := h.Store.DeleteInvitation(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrConflict) {
			respondError(w, http.StatusConflict, "invitation_used", "invitation already used")
			return
		}
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Accounts (admin) ────────────────────────────────────────────────

// ListUsers answers GET /users (admin).
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	users, total, err := h.Store.ListUsers(r.Context(), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(users, total, pg))
}

// UpdateUser answers PUT /users/{id} (admin): role and disabled state.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.UserUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Role != nil && *in.Role != model.RoleAdmin && *in.Role != model.RoleUser {
		respondError(w, http.StatusBadRequest, "bad_request", "role must be admin or user")
		return
	}
	if id == middleware.GetUserID(r.Context()) &&
		((in.Role != nil && *in.Role != model.RoleAdmin) || (in.Disabled != nil && *in.Disabled)) {
		respondError(w, http.StatusConflict, "self_lockout", "an admin cannot disable or demote themself")
		return
	}
	user, err := h.Store.UpdateUser(r.Context(), id, in)
	if errors.Is(err, store.ErrLastAdmin) {
		respondError(w, http.StatusConflict, "last_admin", "the instance must keep an active admin")
		return
	}
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, user)
}

// CreateResetLink answers POST /users/{id}/reset-link (admin): a one-time link, shown once.
func (h *Handler) CreateResetLink(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	token, err := newToken()
	if err != nil {
		respondInternal(w, r, err, "token generation failed")
		return
	}
	expires := time.Now().Add(resetLinkTTL)
	if err := h.Store.CreatePasswordReset(r.Context(), id, hashToken(token), middleware.GetUserID(r.Context()), expires); err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, model.ResetLink{Token: token, ExpiresAt: expires})
}
