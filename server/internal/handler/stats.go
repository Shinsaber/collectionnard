package handler

import "net/http"

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Store.GetStats(r.Context())
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, stats)
}
