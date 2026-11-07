package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/review"
)

// checkReviewSchema adds to checkAttributesSchema the shape of the criteria.
func checkReviewSchema(w http.ResponseWriter, schema json.RawMessage) bool {
	if !checkAttributesSchema(w, schema) {
		return false
	}
	if err := review.CheckSchema(schema); err != nil {
		respondValidationError(w, "invalid criteria", map[string]any{
			"fields": map[string]string{"attributes_schema": err.Error()},
		})
		return false
	}
	return true
}

func checkContextLabel(w http.ResponseWriter, label map[string]string) bool {
	if err := review.CheckContextLabel(label); err != nil {
		respondValidationError(w, "invalid context_label", map[string]any{
			"fields": map[string]string{"context_label": err.Error()},
		})
		return false
	}
	return true
}

func (h *Handler) ListReviewTypes(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, total, err := h.Store.ListReviewTypes(r.Context(), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(items, total, pg))
}

func (h *Handler) GetReviewType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	t, err := h.Store.GetReviewType(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, t)
}

func (h *Handler) CreateReviewType(w http.ResponseWriter, r *http.Request) {
	var in model.ReviewTypeCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Name == "" {
		respondValidationError(w, "name is required", nil)
		return
	}
	if !checkReviewSchema(w, in.AttributesSchema) || !checkContextLabel(w, in.ContextLabel) {
		return
	}
	t, err := h.Store.CreateReviewType(r.Context(), in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, t)
}

func (h *Handler) UpdateReviewType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.ReviewTypeUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if len(in.AttributesSchema) > 0 && !checkReviewSchema(w, in.AttributesSchema) {
		return
	}
	if in.ContextLabel != nil && !checkContextLabel(w, *in.ContextLabel) {
		return
	}
	t, err := h.Store.UpdateReviewType(r.Context(), id, in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, t)
}

func (h *Handler) DeleteReviewType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteReviewType(r.Context(), id); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
