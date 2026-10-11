package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/validator"
)

func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	pg := parsePagination(r)
	collectionTypeID := parseOptionalUUID(r.URL.Query().Get("collection_type_id"))

	cols, total, err := h.Store.ListCollections(r.Context(), userID, pg, collectionTypeID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(cols, total, pg))
}

func (h *Handler) GetCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	col, err := h.Store.GetCollection(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	// Ensure user can only see their own collections
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "Ressource non trouvée")
		return
	}
	respondJSON(w, http.StatusOK, col)
}

func (h *Handler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var in model.CollectionCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Name == "" {
		respondValidationError(w, "Le nom est obligatoire", nil)
		return
	}

	userID := middleware.GetUserID(r.Context())
	col, err := h.Store.CreateCollection(r.Context(), userID, in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, col)
}

func (h *Handler) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	// Check ownership
	col, err := h.Store.GetCollection(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "Ressource non trouvée")
		return
	}

	var in model.CollectionUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	updated, err := h.Store.UpdateCollection(r.Context(), id, in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	col, err := h.Store.GetCollection(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "Ressource non trouvée")
		return
	}

	if err := h.Store.DeleteCollection(r.Context(), id); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Collection Items ---

func (h *Handler) ListCollectionItems(w http.ResponseWriter, r *http.Request) {
	colID, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	col, err := h.Store.GetCollection(r.Context(), colID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "Ressource non trouvée")
		return
	}

	pg := parsePagination(r)
	q := r.URL.Query()
	itemTypeID := parseOptionalUUID(q.Get("item_type_id"))
	sort := q.Get("sort")

	items, total, err := h.Store.ListCollectionItems(r.Context(), colID, pg, itemTypeID, sort)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(items, total, pg))
}

func (h *Handler) AddItemToCollection(w http.ResponseWriter, r *http.Request) {
	colID, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	col, err := h.Store.GetCollection(r.Context(), colID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "Ressource non trouvée")
		return
	}

	var in model.CollectionItemAdd
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if len(in.Attributes) == 0 {
		in.Attributes = json.RawMessage(`{}`)
	}

	// Validate attributes against collection type schema
	ct, err := h.Store.GetCollectionType(r.Context(), col.CollectionTypeID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if vErr := validator.ValidateAttributes(ct.AttributesSchema, in.Attributes); vErr != nil {
		respondValidationError(w, "Les attributs ne respectent pas le schéma du type de collection", map[string]any{
			"fields": validator.FormatValidationError(vErr),
		})
		return
	}

	ci, err := h.Store.AddCollectionItem(r.Context(), colID, in)
	if err != nil {
		if err == store.ErrConflict {
			respondError(w, http.StatusConflict, "conflict", "Cet item est déjà dans la collection")
			return
		}
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, ci)
}

func (h *Handler) UpdateCollectionItem(w http.ResponseWriter, r *http.Request) {
	colID, ok := parseUUID(w, r, "collectionId")
	if !ok {
		return
	}
	itemID, ok := parseUUID(w, r, "itemId")
	if !ok {
		return
	}

	col, err := h.Store.GetCollection(r.Context(), colID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "Ressource non trouvée")
		return
	}

	var in model.CollectionItemUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if !hasAttributes(in.Attributes) {
		respondMissingAttributes(w)
		return
	}

	// Validate attributes
	ct, err := h.Store.GetCollectionType(r.Context(), col.CollectionTypeID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if vErr := validator.ValidateAttributes(ct.AttributesSchema, in.Attributes); vErr != nil {
		respondValidationError(w, "Les attributs ne respectent pas le schéma du type de collection", map[string]any{
			"fields": validator.FormatValidationError(vErr),
		})
		return
	}

	ci, err := h.Store.UpdateCollectionItem(r.Context(), colID, itemID, in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, ci)
}

func (h *Handler) RemoveItemFromCollection(w http.ResponseWriter, r *http.Request) {
	colID, ok := parseUUID(w, r, "collectionId")
	if !ok {
		return
	}
	itemID, ok := parseUUID(w, r, "itemId")
	if !ok {
		return
	}

	col, err := h.Store.GetCollection(r.Context(), colID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "Ressource non trouvée")
		return
	}

	if err := h.Store.RemoveCollectionItem(r.Context(), colID, itemID); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
