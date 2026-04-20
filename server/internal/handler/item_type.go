package handler

import (
	"net/http"

	"github.com/shinsaber/colec/internal/model"
)

func (h *Handler) ListItemTypes(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, total, err := h.Store.ListItemTypes(r.Context(), pg)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(items, total, pg))
}

func (h *Handler) GetItemType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	it, err := h.Store.GetItemType(r.Context(), id)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, it)
}

func (h *Handler) CreateItemType(w http.ResponseWriter, r *http.Request) {
	var in model.ItemTypeCreate
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "JSON invalide")
		return
	}
	if in.Name == "" {
		respondValidationError(w, "Le nom est obligatoire", nil)
		return
	}
	if len(in.AttributesSchema) == 0 {
		in.AttributesSchema = []byte(`{}`)
	}

	it, err := h.Store.CreateItemType(r.Context(), in)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, it)
}

func (h *Handler) UpdateItemType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.ItemTypeUpdate
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "JSON invalide")
		return
	}
	it, err := h.Store.UpdateItemType(r.Context(), id, in)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, it)
}

func (h *Handler) DeleteItemType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteItemType(r.Context(), id); err != nil {
		handleStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
