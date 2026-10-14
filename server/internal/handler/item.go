package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/shinsaber/colec/internal/itemquery"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/validator"
)

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	filters, ok := h.parseItemFilters(w, r)
	if !ok {
		return
	}
	items, total, err := h.Store.ListItems(r.Context(), pg, filters)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(items, total, pg))
}

// parseItemFilters reads the search parameters of GET /items (also used by
// GET /items/export). It answers 400 and returns ok=false on any invalid one.
func (h *Handler) parseItemFilters(w http.ResponseWriter, r *http.Request) (store.ItemFilters, bool) {
	q := r.URL.Query()
	itemTypeID, ok := parseOptionalUUID(w, r, "item_type_id")
	if !ok {
		return store.ItemFilters{}, false
	}
	relatedTo, ok := parseOptionalUUID(w, r, "related_to")
	if !ok {
		return store.ItemFilters{}, false
	}
	relationTypeID, ok := parseOptionalUUID(w, r, "relation_type_id")
	if !ok {
		return store.ItemFilters{}, false
	}
	if relationTypeID != nil && relatedTo == nil {
		respondError(w, http.StatusBadRequest, "bad_request", "relation_type_id requires related_to")
		return store.ItemFilters{}, false
	}
	if len(q.Get("search")) > maxSearchLength || len(q.Get("tag")) > maxSearchLength {
		respondError(w, http.StatusBadRequest, "bad_request",
			fmt.Sprintf("search and tag are limited to %d characters", maxSearchLength))
		return store.ItemFilters{}, false
	}
	filters := store.ItemFilters{
		ItemTypeID:     itemTypeID,
		Search:         q.Get("search"),
		Tag:            q.Get("tag"),
		RelatedTo:      relatedTo,
		RelationTypeID: relationTypeID,
	}
	return filters, h.parseItemSearch(w, r, &filters)
}

// itemSortFields are the non-attribute sorts of GET /items.
var itemSortFields = map[string]bool{"created_at": true, "updated_at": true, "relevance": true}

// parseItemSearch validates `sort` and the attribute filters (`attr.<key>[.<op>]`)
// against the schema of item_type_id, which they require. It answers 400 and
// returns false on any invalid parameter.
func (h *Handler) parseItemSearch(w http.ResponseWriter, r *http.Request, f *store.ItemFilters) bool {
	q := r.URL.Query()
	sort := q.Get("sort")
	if !itemquery.HasAttrParams(q) {
		if sort != "" && !itemSortFields[strings.TrimPrefix(sort, "-")] {
			respondError(w, http.StatusBadRequest, "bad_request",
				"invalid sort: accepted values are created_at, updated_at, relevance, attr.<key> (prefix with - for descending)")
			return false
		}
		f.Sort = sort
		return true
	}
	if f.ItemTypeID == nil {
		respondError(w, http.StatusBadRequest, "bad_request",
			"attribute filters and attribute sort require item_type_id")
		return false
	}
	it, err := h.Store.GetItemType(r.Context(), *f.ItemTypeID)
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusBadRequest, "bad_request", "unknown item_type_id")
		return false
	}
	if err != nil {
		handleStoreError(w, r, err)
		return false
	}
	props, err := itemquery.Properties(it.AttributesSchema)
	if err != nil {
		respondInternal(w, r, err, "internal error")
		return false
	}
	if f.Attrs, err = itemquery.ParseFilters(q, props); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return false
	}
	attrSort, isAttr, err := itemquery.ParseSort(sort, props)
	switch {
	case err != nil:
		respondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return false
	case isAttr:
		f.SortAttr = &attrSort
	case sort != "" && !itemSortFields[strings.TrimPrefix(sort, "-")]:
		respondError(w, http.StatusBadRequest, "bad_request",
			"invalid sort: accepted values are created_at, updated_at, relevance, attr.<key> (prefix with - for descending)")
		return false
	default:
		f.Sort = sort
	}
	return true
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
		respondValidationError(w, "item_type_id is required", nil)
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
		respondValidationError(w, "attributes do not match the item type schema", map[string]any{
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
			respondValidationError(w, "attributes do not match the item type schema", map[string]any{
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
	// 404 if the item does not exist (the membership query alone would return []).
	if _, err := h.Store.GetItem(r.Context(), id); err != nil {
		handleStoreError(w, r, err)
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
