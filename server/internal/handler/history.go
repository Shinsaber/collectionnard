package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/validator"
)

// withActor records the current user as the author of every catalog write of
// the request (history triggers, P4-3a); handlers refine the origin
// (store.WithOrigin) for imports and external sources.
func withActor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := store.WithActor(r.Context(), store.Actor{
			UserID: middleware.GetUserID(r.Context()), Origin: store.OriginDirect,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authors resolves the authors of history entries for the reader, from the
// CURRENT profile visibility: a profile made private later, or a disabled
// account, becomes "a member" in the whole history. Admins always see the
// author (moderation).
type authors struct {
	h      *Handler
	r      *http.Request
	viewer uuid.UUID
	admin  bool
	cache  map[uuid.UUID]model.HistoryAuthor
}

func (h *Handler) newAuthors(r *http.Request) *authors {
	return &authors{
		h: h, r: r, viewer: middleware.GetUserID(r.Context()),
		admin: middleware.GetRole(r.Context()) == model.RoleAdmin,
		cache: map[uuid.UUID]model.HistoryAuthor{},
	}
}

func (a *authors) resolve(userID *uuid.UUID) model.HistoryAuthor {
	if userID == nil {
		return model.HistoryAuthor{Kind: model.AuthorSystem}
	}
	if got, ok := a.cache[*userID]; ok {
		return got
	}
	out := model.HistoryAuthor{Kind: model.AuthorMember}
	if u, err := a.h.Store.GetUserByID(a.r.Context(), *userID); err == nil {
		visible := a.admin || (u.DisabledAt == nil && privacy.CanView(u.ProfileVisibility, u.ID, a.viewer))
		if visible {
			out = model.HistoryAuthor{Kind: model.AuthorUser,
				User: &model.Owner{Username: u.Username, DisplayName: u.DisplayName, Avatar: u.Avatar}}
		}
	}
	a.cache[*userID] = out
	return out
}

func (h *Handler) GetItemHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	hist, err := h.Store.GetItemHistory(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	a := h.newAuthors(r)
	for i := range hist.Entries {
		hist.Entries[i].Author = a.resolve(hist.Entries[i].UserID)
	}
	if hist.Created != nil {
		hist.Created.Author = a.resolve(hist.Created.UserID)
	}
	respondJSON(w, http.StatusOK, hist)
}

// RevertItem restores the attributes of an item version (admin, contributor),
// validated against the current schema; the write is recorded as `revert`.
func (h *Handler) RevertItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.RevertRequest
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Version < 1 {
		respondError(w, http.StatusBadRequest, "bad_request", "version must be at least 1")
		return
	}
	item, err := h.Store.GetItem(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	attrs, err := h.Store.GetItemVersion(r.Context(), id, in.Version)
	if errors.Is(err, store.ErrDeletedVersion) {
		respondError(w, http.StatusBadRequest, "bad_request", "this version is a deletion")
		return
	}
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if item.ItemType != nil {
		if vErr := validator.ValidateAttributes(item.ItemType.AttributesSchema, attrs); vErr != nil {
			respondValidationError(w, "this version does not match the current item type schema", map[string]any{
				"fields": validator.FormatValidationError(vErr),
			})
			return
		}
	}
	ctx := store.WithOrigin(r.Context(), store.OriginRevert, strconv.Itoa(in.Version))
	updated, err := h.Store.UpdateItem(ctx, id, model.ItemUpdate{Attributes: attrs})
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}
