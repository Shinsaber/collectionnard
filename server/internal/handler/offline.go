package handler

import (
	"net/http"

	"github.com/shinsaber/colec/internal/middleware"
)

// GetOfflineSnapshot answers GET /me/offline: the current user's collections with
// their items and every meta-type, for the offline copy of the mobile app.
// Never cached by intermediaries (personal data).
func (h *Handler) GetOfflineSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := h.Store.GetOfflineSnapshot(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	respondJSON(w, http.StatusOK, snap)
}
