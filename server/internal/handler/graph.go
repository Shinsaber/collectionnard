package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func (h *Handler) GetItemGraph(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	q := r.URL.Query()
	maxDepth := 3
	if v := q.Get("max_depth"); v != "" {
		d, err := strconv.Atoi(v)
		if err == nil && d >= 1 && d <= 10 {
			maxDepth = d
		}
	}

	var relationTypeIDs []uuid.UUID
	if v := q.Get("relation_type_ids"); v != "" {
		for _, raw := range strings.Split(v, ",") {
			id, err := uuid.Parse(strings.TrimSpace(raw))
			if err == nil {
				relationTypeIDs = append(relationTypeIDs, id)
			}
		}
	}

	graph, err := h.Store.GetItemGraph(r.Context(), itemID, maxDepth, relationTypeIDs)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, graph)
}
