package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
)

const (
	graphDefaultDepth     = 3
	graphMaxDepthUser     = 5
	graphMaxDepthAdmin    = 10
	graphMaxRelationTypes = 20
	graphTimeout          = 5 * time.Second
)

func (h *Handler) GetItemGraph(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	q := r.URL.Query()
	depthLimit := graphMaxDepthUser
	if middleware.GetRole(r.Context()) == "admin" {
		depthLimit = graphMaxDepthAdmin
	}
	maxDepth := graphDefaultDepth
	if v := q.Get("max_depth"); v != "" {
		d, err := strconv.Atoi(v)
		if err != nil || d < 1 || d > depthLimit {
			respondError(w, http.StatusBadRequest, "bad_request",
				fmt.Sprintf("max_depth doit être un entier entre 1 et %d", depthLimit))
			return
		}
		maxDepth = d
	}

	var relationTypeIDs []uuid.UUID
	if v := q.Get("relation_type_ids"); v != "" {
		for _, raw := range strings.Split(v, ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			id, err := uuid.Parse(raw)
			if err != nil {
				respondError(w, http.StatusBadRequest, "bad_request", "relation_type_ids: UUID invalide: "+raw)
				return
			}
			relationTypeIDs = append(relationTypeIDs, id)
		}
		if len(relationTypeIDs) > graphMaxRelationTypes {
			respondError(w, http.StatusBadRequest, "bad_request",
				fmt.Sprintf("relation_type_ids: %d identifiants maximum", graphMaxRelationTypes))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), graphTimeout)
	defer cancel()

	graph, err := h.Store.GetItemGraph(ctx, itemID, maxDepth, h.Config.GraphMaxNodes, relationTypeIDs)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			respondError(w, http.StatusServiceUnavailable, "timeout",
				"Exploration du graphe trop longue : réduisez max_depth ou filtrez par relation_type_ids")
			return
		}
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, graph)
}
