package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/validator"
)

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	q := r.URL.Query()
	itemTypeID, ok := parseOptionalUUID(w, r, "item_type_id")
	if !ok {
		return
	}
	if len(q.Get("search")) > maxSearchLength || len(q.Get("tag")) > maxSearchLength {
		respondError(w, http.StatusBadRequest, "bad_request",
			fmt.Sprintf("search et tag sont limités à %d caractères", maxSearchLength))
		return
	}
	filters := store.ItemFilters{
		ItemTypeID: itemTypeID,
		Search:     q.Get("search"),
		Tag:        q.Get("tag"),
		Sort:       q.Get("sort"),
	}

	items, total, err := h.Store.ListItems(r.Context(), pg, filters)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(items, total, pg))
}

func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	item, err := h.Store.GetItem(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var in model.ItemCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.ItemTypeID.String() == "00000000-0000-0000-0000-000000000000" {
		respondValidationError(w, "item_type_id est obligatoire", nil)
		return
	}
	if len(in.Attributes) == 0 {
		in.Attributes = json.RawMessage(`{}`)
	}

	// Validate attributes against item type schema
	itemType, err := h.Store.GetItemType(r.Context(), in.ItemTypeID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}

	if vErr := validator.ValidateAttributes(itemType.AttributesSchema, in.Attributes); vErr != nil {
		respondValidationError(w, "Les attributs ne respectent pas le schéma du type", map[string]any{
			"fields": validator.FormatValidationError(vErr),
		})
		return
	}

	item, err := h.Store.CreateItem(r.Context(), in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.ItemUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}

	// Validate attributes if provided
	if len(in.Attributes) > 0 {
		existing, err := h.Store.GetItem(r.Context(), id)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		if vErr := validator.ValidateAttributes(existing.ItemType.AttributesSchema, in.Attributes); vErr != nil {
			respondValidationError(w, "Les attributs ne respectent pas le schéma du type", map[string]any{
				"fields": validator.FormatValidationError(vErr),
			})
			return
		}
	}

	item, err := h.Store.UpdateItem(r.Context(), id, in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteItem(r.Context(), id); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListItemCollections(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	userID := middleware.GetUserID(r.Context())
	memberships, err := h.Store.ListItemCollections(r.Context(), userID, id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if memberships == nil {
		memberships = []model.ItemCollectionMembership{}
	}
	respondJSON(w, http.StatusOK, memberships)
}
