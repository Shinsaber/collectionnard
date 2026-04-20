package handler

import (
	"net/http"

	"github.com/shinsaber/colec/internal/model"
)

func (h *Handler) ListCollectionTypes(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, total, err := h.Store.ListCollectionTypes(r.Context(), pg)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(items, total, pg))
}

func (h *Handler) GetCollectionType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	ct, err := h.Store.GetCollectionType(r.Context(), id)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, ct)
}

func (h *Handler) CreateCollectionType(w http.ResponseWriter, r *http.Request) {
	var in model.CollectionTypeCreate
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "JSON invalide")
		return
	}
	if in.Name == "" {
		respondValidationError(w, "Le nom est obligatoire", nil)
		return
	}

	ct, err := h.Store.CreateCollectionType(r.Context(), in)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, ct)
}

func (h *Handler) UpdateCollectionType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.CollectionTypeUpdate
	if err := decodeJSON(r, &in); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "JSON invalide")
		return
	}
	ct, err := h.Store.UpdateCollectionType(r.Context(), id, in)
	if err != nil {
		handleStoreError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, ct)
}

func (h *Handler) DeleteCollectionType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteCollectionType(r.Context(), id); err != nil {
		handleStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
