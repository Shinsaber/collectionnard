package handler

import (
	"net/http"

	"github.com/shinsaber/colec/internal/model"
)

func (h *Handler) ListRelationTypes(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	sourceID, ok := parseOptionalUUID(w, r, "source_item_type_id")
	if !ok {
		return
	}
	targetID, ok := parseOptionalUUID(w, r, "target_item_type_id")
	if !ok {
		return
	}

	items, total, err := h.Store.ListRelationTypes(r.Context(), pg, sourceID, targetID)
	if err != nil {
		handleStoreError(w, r, err)
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
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, rt)
}

func (h *Handler) CreateRelationType(w http.ResponseWriter, r *http.Request) {
	var in model.RelationTypeCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Name == "" {
		respondValidationError(w, "name is required", nil)
		return
	}
	if !checkAttributesSchema(w, in.AttributesSchema) {
		return
	}

	rt, err := h.Store.CreateRelationType(r.Context(), in)
	if err != nil {
		handleStoreError(w, r, err)
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
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if len(in.AttributesSchema) > 0 && !checkAttributesSchema(w, in.AttributesSchema) {
		return
	}
	rt, err := h.Store.UpdateRelationType(r.Context(), id, in)
	if err != nil {
		handleStoreError(w, r, err)
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
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
