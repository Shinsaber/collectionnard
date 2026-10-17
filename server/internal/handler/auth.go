package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const refreshTokenBytes = 32

// dummyHash is compared against the submitted password when the username is
// unknown, so that a login attempt costs the same time whether or not the user
// exists (no username enumeration through response timing).
var dummyHash = func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("collectionnard-dummy-password"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return h
}()

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondDecodeError(w, err)
		return
	}
	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "username and password are required")
		return
	}

	user, passwordHash, err := h.Store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
			respondError(w, http.StatusUnauthorized, "unauthorized", "invalid username or password")
			return
		}
		respondInternal(w, r, err, "internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", "invalid username or password")
		return
	}
	if user.DisabledAt != nil {
		respondError(w, http.StatusUnauthorized, "account_disabled", "this account is disabled")
		return
	}

	h.issueTokens(w, r, user)
}

// Refresh exchanges a valid refresh token for a new access token and a new
// refresh token (rotation: the presented token is invalidated).
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondDecodeError(w, err)
		return
	}
	if req.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "refresh_token is required")
		return
	}

	userID, err := h.Store.ConsumeRefreshToken(r.Context(), hashToken(req.RefreshToken))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired refresh token")
			return
		}
		respondInternal(w, r, err, "internal error")
		return
	}

	// The user is re-read so that a deleted account or a changed role takes
	// effect at the next refresh at the latest.
	user, err := h.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusUnauthorized, "unauthorized", "unknown user")
			return
		}
		respondInternal(w, r, err, "internal error")
		return
	}
	if user.DisabledAt != nil {
		respondError(w, http.StatusUnauthorized, "account_disabled", "this account is disabled")
		return
	}

	h.issueTokens(w, r, user)
}

// Logout revokes a refresh token. The access token stays valid until it expires.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondDecodeError(w, err)
		return
	}
	if req.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "refresh_token is required")
		return
	}
	if err := h.Store.DeleteRefreshToken(r.Context(), hashToken(req.RefreshToken)); err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	user, err := h.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	user.CanInvite = h.canInvite(r)
	respondJSON(w, http.StatusOK, user)
}

// issueTokens generates an access token and a fresh refresh token for user.
func (h *Handler) issueTokens(w http.ResponseWriter, r *http.Request, user *model.User) {
	access, err := h.Auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		respondInternal(w, r, err, "token generation failed")
		return
	}

	raw := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		respondInternal(w, r, err, "token generation failed")
		return
	}
	refresh := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().Add(h.Config.RefreshTokenTTL)
	if err := h.Store.CreateRefreshToken(r.Context(), user.ID, hashToken(refresh), expiresAt); err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, model.LoginResponse{
		Token:        access,
		RefreshToken: refresh,
		ExpiresIn:    int(h.Auth.AccessTTL().Seconds()),
		User:         *user,
	})
}

// hashToken returns the hex SHA-256 of a refresh token: only this value is stored.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
