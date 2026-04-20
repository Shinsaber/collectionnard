package handler

import "net/http"

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	tags, err := h.Store.ListTags(r.Context(), search)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, tags)
}
