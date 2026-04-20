package handler

import (
	"errors"
	"net/http"

	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "JSON invalide")
		return
	}
	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "username et password requis")
		return
	}

	user, passwordHash, err := h.Store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusUnauthorized, "unauthorized", "Nom d'utilisateur ou mot de passe incorrect")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal", "Erreur interne")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", "Nom d'utilisateur ou mot de passe incorrect")
		return
	}

	token, err := h.Auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal", "Erreur génération token")
		return
	}

	respondJSON(w, http.StatusOK, model.LoginResponse{Token: token, User: *user})
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	user, err := h.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, user)
}
