package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/validator"
)

func (h *Handler) ListItemRelations(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	relationTypeID, ok := parseOptionalUUID(w, r, "relation_type_id")
	if !ok {
		return
	}
	relations, err := h.Store.ListItemRelations(r.Context(), itemID, relationTypeID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, relations)
}

func (h *Handler) GetRelation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	rel, err := h.Store.GetRelation(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, rel)
}

func (h *Handler) CreateRelation(w http.ResponseWriter, r *http.Request) {
	var in model.ItemRelationCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}

	if in.SourceItemID == in.TargetItemID {
		respondValidationError(w, "Un item ne peut pas être relié à lui-même", nil)
		return
	}

	if len(in.Attributes) == 0 {
		in.Attributes = json.RawMessage(`{}`)
	}

	// Validate type constraints
	rt, err := h.Store.GetRelationType(r.Context(), in.RelationTypeID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}

	sourceItem, err := h.Store.GetItem(r.Context(), in.SourceItemID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if !containsUUID(rt.SourceItemTypeIDs, sourceItem.ItemTypeID) {
		respondValidationError(w, "Le type de l'item source ne correspond pas aux contraintes du RelationType", nil)
		return
	}

	targetItem, err := h.Store.GetItem(r.Context(), in.TargetItemID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if !containsUUID(rt.TargetItemTypeIDs, targetItem.ItemTypeID) {
		respondValidationError(w, "Le type de l'item cible ne correspond pas aux contraintes du RelationType", nil)
		return
	}

	// Validate attributes against relation type schema
	if vErr := validator.ValidateAttributes(rt.AttributesSchema, in.Attributes); vErr != nil {
		respondValidationError(w, "Les attributs ne respectent pas le schéma du type de relation", map[string]any{
			"fields": validator.FormatValidationError(vErr),
		})
		return
	}

	rel, err := h.Store.CreateRelation(r.Context(), in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, rel)
}

func (h *Handler) UpdateRelation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.ItemRelationUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if !hasAttributes(in.Attributes) {
		respondMissingAttributes(w)
		return
	}

	// Get existing relation to validate against its type schema
	existing, err := h.Store.GetRelation(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}

	if vErr := validator.ValidateAttributes(existing.RelationType.AttributesSchema, in.Attributes); vErr != nil {
		respondValidationError(w, "Les attributs ne respectent pas le schéma du type de relation", map[string]any{
			"fields": validator.FormatValidationError(vErr),
		})
		return
	}

	rel, err := h.Store.UpdateRelation(r.Context(), id, in)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, rel)
}

func (h *Handler) DeleteRelation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteRelation(r.Context(), id); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
