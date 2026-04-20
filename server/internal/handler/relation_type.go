package handler

import (
	"net/http"

	"github.com/shinsaber/colec/internal/model"
)

func (h *Handler) ListRelationTypes(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	q := r.URL.Query()
	sourceID := parseOptionalUUID(q.Get("source_item_type_id"))
	targetID := parseOptionalUUID(q.Get("target_item_type_id"))

	items, total, err := h.Store.ListRelationTypes(r.Context(), pg, sourceID, targetID)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(items, total, pg))
}

func (h *Handler) GetRelationType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	rt, err := h.Store.GetRelationType(r.Context(), id)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, rt)
}

func (h *Handler) CreateRelationType(w http.ResponseWriter, r *http.Request) {
	var in model.RelationTypeCreate
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "JSON invalide")
		return
	}
	if in.Name == "" {
		respondValidationError(w, "Le nom est obligatoire", nil)
		return
	}

	rt, err := h.Store.CreateRelationType(r.Context(), in)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, rt)
}

func (h *Handler) UpdateRelationType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.RelationTypeUpdate
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "JSON invalide")
		return
	}
	rt, err := h.Store.UpdateRelationType(r.Context(), id, in)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, rt)
}

func (h *Handler) DeleteRelationType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteRelationType(r.Context(), id); err != nil {
		handleStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
