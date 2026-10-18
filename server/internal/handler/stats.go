package handler

import (
	"net/http"

	"github.com/shinsaber/colec/internal/middleware"
)

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Store.GetStats(r.Context())
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

// GetMyStats sums up the collections of the current user (P4-1).
func (h *Handler) GetMyStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Store.GetUserStats(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

// GetCollectionStats returns the statistics of one of the current user's
// collections (404 for someone else's).
func (h *Handler) GetCollectionStats(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	col, err := h.Store.GetCollection(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}
	stats, err := h.Store.GetCollectionStats(r.Context(), id, nil)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, stats)
}
