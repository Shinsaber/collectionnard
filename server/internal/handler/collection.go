package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/itemquery"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/validator"
)

func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	pg := parsePagination(r)
	collectionTypeID, ok := parseOptionalUUID(w, r, "collection_type_id")
	if !ok {
		return
	}

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
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
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
		respondValidationError(w, "name is required", nil)
		return
	}
	if len(in.ItemTypeIDs) > 0 || in.DefaultSort != "" {
		ct, err := h.Store.GetCollectionType(r.Context(), in.CollectionTypeID)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		in.ItemTypeIDs = uniqueIDs(in.ItemTypeIDs)
		draft := &model.Collection{CollectionType: ct, ItemTypeIDs: in.ItemTypeIDs}
		if !h.checkCollectionSettings(w, r, draft, in.DefaultSort) {
			return
		}
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
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	var in model.CollectionUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Visibility != nil && !privacy.Valid(*in.Visibility) {
		respondError(w, http.StatusBadRequest, "bad_request", "visibility: private, members or public")
		return
	}
	if in.HiddenFields.Set && in.HiddenFields.Value != nil {
		var schema []byte
		if col.CollectionType != nil {
			schema = col.CollectionType.AttributesSchema
		}
		if unknown := privacy.UnknownFields(schema, in.HiddenFields.Value); len(unknown) > 0 {
			respondValidationError(w, "hidden_fields: unknown fields", map[string]any{"unknown": unknown})
			return
		}
		in.HiddenFields.Value = slices.Compact(slices.Sorted(slices.Values(in.HiddenFields.Value)))
	}
	if in.ItemTypeIDs != nil || in.DefaultSort != nil {
		draft := *col
		if in.ItemTypeIDs != nil {
			ids := uniqueIDs(*in.ItemTypeIDs)
			in.ItemTypeIDs, draft.ItemTypeIDs = &ids, ids
		}
		// Without a new default sort, the current one is kept even if it no longer
		// applies: the list then falls back to the latest added.
		sort := ""
		if in.DefaultSort != nil {
			sort = *in.DefaultSort
		}
		if !h.checkCollectionSettings(w, r, &draft, sort) {
			return
		}
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
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
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
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	pg := parsePagination(r)
	f, ok := h.parseCollectionItemFilters(w, r, col)
	if !ok {
		return
	}

	items, total, err := h.Store.ListCollectionItems(r.Context(), colID, pg, f)
	if err == nil {
		err = h.withActiveLoans(r, items)
	}
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
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
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
	if in.Quantity != nil && !validQuantity(w, *in.Quantity) {
		return
	}

	// Validate attributes against collection type schema
	ct, err := h.Store.GetCollectionType(r.Context(), col.CollectionTypeID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if vErr := validator.ValidateAttributes(ct.AttributesSchema, in.Attributes); vErr != nil {
		respondValidationError(w, "attributes do not match the collection type schema", map[string]any{
			"fields": validator.FormatValidationError(vErr),
		})
		return
	}
	// Allowed item types (empty = all); items already in the collection are not rechecked.
	// Accepted types: those of the collection within those of its type (P7-1b).
	limited := *col
	limited.CollectionType = ct
	if accepted := limited.AcceptedItemTypes(); len(accepted) > 0 {
		item, err := h.Store.GetItem(r.Context(), in.ItemID)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		if !slices.Contains(accepted, item.ItemTypeID) {
			respondError(w, http.StatusUnprocessableEntity, "item_type_not_allowed",
				"the collection does not accept items of type "+item.ItemTypeID.String())
			return
		}
	}

	ci, err := h.Store.AddCollectionItem(r.Context(), colID, in)
	if err != nil {
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
	lotID, ok := parseUUID(w, r, "lotId")
	if !ok {
		return
	}

	col, err := h.Store.GetCollection(r.Context(), colID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
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
	if in.Quantity != nil && !validQuantity(w, *in.Quantity) {
		return
	}

	// Validate attributes
	ct, err := h.Store.GetCollectionType(r.Context(), col.CollectionTypeID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if vErr := validator.ValidateAttributes(ct.AttributesSchema, in.Attributes); vErr != nil {
		respondValidationError(w, "attributes do not match the collection type schema", map[string]any{
			"fields": validator.FormatValidationError(vErr),
		})
		return
	}

	ci, err := h.Store.UpdateCollectionItem(r.Context(), colID, lotID, in)
	if err != nil {
		respondLotError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, ci)
}

func (h *Handler) RemoveItemFromCollection(w http.ResponseWriter, r *http.Request) {
	colID, ok := parseUUID(w, r, "collectionId")
	if !ok {
		return
	}
	lotID, ok := parseUUID(w, r, "lotId")
	if !ok {
		return
	}

	col, err := h.Store.GetCollection(r.Context(), colID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	if err := h.Store.RemoveCollectionItem(r.Context(), colID, lotID); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// validQuantity answers 422 invalid_quantity outside 1..model.MaxQuantity.
func validQuantity(w http.ResponseWriter, quantity int) bool {
	if quantity < 1 || quantity > model.MaxQuantity {
		respondError(w, http.StatusUnprocessableEntity, "invalid_quantity",
			fmt.Sprintf("quantity must be between 1 and %d", model.MaxQuantity))
		return false
	}
	return true
}

// respondLotError translates the errors of a split or a merge.
func respondLotError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrInvalidQuantity):
		respondError(w, http.StatusUnprocessableEntity, "invalid_quantity",
			"a split leaves at least one copy, a merge at most "+strconv.Itoa(model.MaxQuantity)+
				", and a lot keeps at least its copies lent")
	case errors.Is(err, store.ErrNotMergeable):
		respondError(w, http.StatusUnprocessableEntity, "lots_not_mergeable",
			"only two different lots of the same item can be merged")
	default:
		handleStoreError(w, r, err)
	}
}

// SplitCollectionLot moves copies of a lot to a new lot (P7-1).
func (h *Handler) SplitCollectionLot(w http.ResponseWriter, r *http.Request) {
	col, ok := h.ownCollection(w, r, "collectionId")
	if !ok {
		return
	}
	lotID, ok := parseUUID(w, r, "lotId")
	if !ok {
		return
	}
	var in model.CollectionLotSplit
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if !validQuantity(w, in.Quantity) {
		return
	}
	if len(in.Attributes) > 0 {
		ct, err := h.Store.GetCollectionType(r.Context(), col.CollectionTypeID)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		if vErr := validator.ValidateAttributes(ct.AttributesSchema, in.Attributes); vErr != nil {
			respondValidationError(w, "attributes do not match the collection type schema", map[string]any{
				"fields": validator.FormatValidationError(vErr),
			})
			return
		}
	}
	lot, err := h.Store.SplitCollectionLot(r.Context(), col.ID, lotID, in.Quantity, in.Attributes)
	if err != nil {
		respondLotError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, lot)
}

// MergeCollectionLot adds the copies of a lot to another lot of the same item (P7-1).
func (h *Handler) MergeCollectionLot(w http.ResponseWriter, r *http.Request) {
	col, ok := h.ownCollection(w, r, "collectionId")
	if !ok {
		return
	}
	lotID, ok := parseUUID(w, r, "lotId")
	if !ok {
		return
	}
	var in model.CollectionLotMerge
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Into == uuid.Nil {
		respondValidationError(w, "into is required", nil)
		return
	}
	var galleries []string
	if col.CollectionType != nil {
		galleries = galleryKeys(col.CollectionType.AttributesSchema)
	}
	lot, err := h.Store.MergeCollectionLots(r.Context(), col.ID, lotID, in.Into, galleries)
	if err != nil {
		respondLotError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, lot)
}

// lotSortFields are the non-attribute sorts of GET /collections/{id}/items.
var lotSortFields = map[string]bool{"created_at": true, "updated_at": true, "quantity": true, "relevance": true}

// parseCollectionItemFilters reads the search of the lots of a collection
// (P7-1): the catalogue search (type, text, attributes, attribute sort) and
// `multiple`. It answers 400 and returns ok=false on any invalid parameter.
func (h *Handler) parseCollectionItemFilters(w http.ResponseWriter, r *http.Request, col *model.Collection) (store.CollectionItemFilters, bool) {
	q := r.URL.Query()
	var f store.CollectionItemFilters
	itemTypeID, ok := parseOptionalUUID(w, r, "item_type_id")
	if !ok {
		return f, false
	}
	if len(q.Get("search")) > maxSearchLength {
		respondError(w, http.StatusBadRequest, "bad_request",
			fmt.Sprintf("search is limited to %d characters", maxSearchLength))
		return f, false
	}
	switch q.Get("multiple") {
	case "", "false":
	case "true":
		f.Multiple = true
	default:
		respondError(w, http.StatusBadRequest, "bad_request", "multiple must be true or false")
		return f, false
	}
	f.ItemTypeID, f.Search = itemTypeID, q.Get("search")
	// A specialized collection sorts and filters by the attributes of its only
	// item type without filtering by type (P7-1b).
	if !h.parseItemSearch(w, r, &f.ItemFilters, lotSortFields, col.SpecializedType()) {
		return f, false
	}
	if q.Get("sort") == "" {
		h.applyDefaultSort(r, &f, col)
	}
	return f, true
}

// galleryKeys lists the image galleries (`x-widget: image-gallery`) of a
// schema, whose photos follow the copies of a merged lot.
func galleryKeys(schema json.RawMessage) []string {
	var root struct {
		Properties map[string]struct {
			Widget string `json:"x-widget"`
		} `json:"properties"`
	}
	if json.Unmarshal(schema, &root) != nil {
		return nil
	}
	var keys []string
	for k, p := range root.Properties {
		if p.Widget == "image-gallery" {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

// defaultSortFields are the non-attribute default sorts of a collection.
var defaultSortFields = map[string]bool{"created_at": true, "updated_at": true, "quantity": true}

// checkCollectionSettings validates the item types a collection accepts and its
// default sort (P7-1b) against draft (the collection as it would be): the types
// must be allowed by the CollectionType (422 item_type_not_allowed), an
// attribute sort requires a specialized collection and an attribute of its item
// type (422 invalid_sort).
func (h *Handler) checkCollectionSettings(w http.ResponseWriter, r *http.Request, draft *model.Collection, defaultSort string) bool {
	if draft.CollectionType != nil {
		for _, id := range draft.ItemTypeIDs {
			if limit := draft.CollectionType.ItemTypeIDs; len(limit) > 0 && !slices.Contains(limit, id) {
				respondError(w, http.StatusUnprocessableEntity, "item_type_not_allowed",
					"the collection type does not allow items of type "+id.String())
				return false
			}
		}
	}
	if defaultSort == "" {
		return true
	}
	invalid := func(msg string) bool {
		respondError(w, http.StatusUnprocessableEntity, "invalid_sort", msg)
		return false
	}
	if !strings.Contains(defaultSort, itemquery.Prefix) {
		if !defaultSortFields[strings.TrimPrefix(defaultSort, "-")] {
			return invalid("default_sort: created_at, updated_at, quantity or attr.<key> (prefix with - for descending)")
		}
		return true
	}
	typeID := draft.SpecializedType()
	if typeID == nil {
		return invalid("an attribute sort requires a collection that accepts a single item type")
	}
	props, err := h.itemTypeProps(r, *typeID)
	if err != nil {
		handleStoreError(w, r, err)
		return false
	}
	if _, ok, err := itemquery.ParseSort(defaultSort, props); !ok || err != nil {
		return invalid("default_sort: unknown or unsortable attribute")
	}
	return true
}

// applyDefaultSort sorts by the default sort of col when the request gives
// none; a default sort that no longer applies (attribute removed, accepted
// types changed) is ignored: the latest added come first.
func (h *Handler) applyDefaultSort(r *http.Request, f *store.CollectionItemFilters, col *model.Collection) {
	sort := col.DefaultSort
	if sort == "" {
		return
	}
	if !strings.Contains(sort, itemquery.Prefix) {
		if defaultSortFields[strings.TrimPrefix(sort, "-")] {
			f.Sort = sort
		}
		return
	}
	typeID := col.SpecializedType()
	if typeID == nil {
		return
	}
	props, err := h.itemTypeProps(r, *typeID)
	if err != nil {
		return
	}
	if s, ok, err := itemquery.ParseSort(sort, props); ok && err == nil {
		f.SortAttr = &s
	}
}

// itemTypeProps reads the properties of an item type (filters and sorts).
func (h *Handler) itemTypeProps(r *http.Request, id uuid.UUID) (map[string]itemquery.Property, error) {
	it, err := h.Store.GetItemType(r.Context(), id)
	if err != nil {
		return nil, err
	}
	return itemquery.Properties(it.AttributesSchema)
}

// uniqueIDs drops the duplicates of ids, keeping their order.
func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}
