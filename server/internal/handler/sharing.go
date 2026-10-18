package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
)

// Visibility and sharing (P4-2). Every read below is made "as others see it":
// the hidden fields of the collection are stripped for every reader, the owner
// included (preview of the public page) and admins included. The owner's full
// view stays GET /collections/{id}.

// readable is a collection read through a public route, with its owner and
// the fields to hide.
type readable struct {
	col    *model.Collection
	owner  *model.User
	hidden []string
}

func notFound(w http.ResponseWriter) {
	respondError(w, http.StatusNotFound, "not_found", "resource not found")
}

// loadReadable loads collection id for a public read; shared skips the
// visibility check (secret link). Writes the error response and returns false
// when the reader may not see it (404, never 403: nothing is disclosed).
func (h *Handler) loadReadable(w http.ResponseWriter, r *http.Request, id uuid.UUID, shared bool) (*readable, bool) {
	col, err := h.Store.GetCollection(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	if !shared && !privacy.CanView(col.Visibility, col.UserID, middleware.GetUserID(r.Context())) {
		notFound(w)
		return nil, false
	}
	owner, err := h.Store.GetUserByID(r.Context(), col.UserID)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	if owner.DisabledAt != nil {
		notFound(w)
		return nil, false
	}
	var schema []byte
	if col.CollectionType != nil {
		schema = col.CollectionType.AttributesSchema
	}
	return &readable{col: col, owner: owner, hidden: privacy.HiddenFields(schema, col.HiddenFields)}, true
}

// public converts to the response: CollectionType schema without the hidden fields.
func (rd *readable) public() model.PublicCollection {
	var ct *model.CollectionType
	if rd.col.CollectionType != nil {
		c := *rd.col.CollectionType
		c.AttributesSchema = privacy.StripSchema(c.AttributesSchema, rd.hidden)
		ct = &c
	}
	return model.PublicCollection{
		ID: rd.col.ID, Name: rd.col.Name, Description: rd.col.Description, CollectionType: ct,
		ItemCount: rd.col.ItemCount, Visibility: rd.col.Visibility,
		Owner:     model.Owner{Username: rd.owner.Username, DisplayName: rd.owner.DisplayName, Avatar: rd.owner.Avatar},
		CreatedAt: rd.col.CreatedAt, UpdatedAt: rd.col.UpdatedAt,
	}
}

func (h *Handler) respondReadableItems(w http.ResponseWriter, r *http.Request, rd *readable) {
	pg := parsePagination(r)
	items, total, err := h.Store.ListCollectionItems(r.Context(), rd.col.ID, pg, nil, "")
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	for i := range items {
		items[i].Attributes = privacy.StripAttributes(items[i].Attributes, rd.hidden)
	}
	if items == nil {
		items = []model.CollectionItem{}
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(items, total, pg))
}

func (h *Handler) respondReadableStats(w http.ResponseWriter, r *http.Request, rd *readable) {
	stats, err := h.Store.GetCollectionStats(r.Context(), rd.col.ID, rd.hidden)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

// --- /public/collections/{id} ---

func (h *Handler) publicCollection(w http.ResponseWriter, r *http.Request) (*readable, bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return nil, false
	}
	return h.loadReadable(w, r, id, false)
}

func (h *Handler) GetPublicCollection(w http.ResponseWriter, r *http.Request) {
	if rd, ok := h.publicCollection(w, r); ok {
		respondJSON(w, http.StatusOK, rd.public())
	}
}

func (h *Handler) ListPublicCollectionItems(w http.ResponseWriter, r *http.Request) {
	if rd, ok := h.publicCollection(w, r); ok {
		h.respondReadableItems(w, r, rd)
	}
}

func (h *Handler) GetPublicCollectionStats(w http.ResponseWriter, r *http.Request) {
	if rd, ok := h.publicCollection(w, r); ok {
		h.respondReadableStats(w, r, rd)
	}
}

// --- /share?token= ---

func (h *Handler) sharedCollection(w http.ResponseWriter, r *http.Request) (*readable, bool) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "token is required")
		return nil, false
	}
	id, err := h.Store.GetSharedCollectionID(r.Context(), hashToken(token))
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	return h.loadReadable(w, r, id, true)
}

func (h *Handler) GetSharedCollection(w http.ResponseWriter, r *http.Request) {
	if rd, ok := h.sharedCollection(w, r); ok {
		respondJSON(w, http.StatusOK, rd.public())
	}
}

func (h *Handler) ListSharedCollectionItems(w http.ResponseWriter, r *http.Request) {
	if rd, ok := h.sharedCollection(w, r); ok {
		h.respondReadableItems(w, r, rd)
	}
}

func (h *Handler) GetSharedCollectionStats(w http.ResponseWriter, r *http.Request) {
	if rd, ok := h.sharedCollection(w, r); ok {
		h.respondReadableStats(w, r, rd)
	}
}

// --- /public/users/{username} ---

// visibleProfile loads an active user whose profile the reader may see.
func (h *Handler) visibleProfile(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	u, _, err := h.Store.GetUserByUsername(r.Context(), chi.URLParam(r, "username"))
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	if u.DisabledAt != nil || !privacy.CanView(u.ProfileVisibility, u.ID, middleware.GetUserID(r.Context())) {
		notFound(w)
		return nil, false
	}
	return u, true
}

func (h *Handler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	if u, ok := h.visibleProfile(w, r); ok {
		respondJSON(w, http.StatusOK, model.NewPublicProfile(u))
	}
}

// ListPublicCollections lists the collections of a visible profile that the
// reader may see (the owner sees the list as members do: preview).
func (h *Handler) ListPublicCollections(w http.ResponseWriter, r *http.Request) {
	u, ok := h.visibleProfile(w, r)
	if !ok {
		return
	}
	levels := privacy.Levels(middleware.GetUserID(r.Context()))
	cols, err := h.Store.ListVisibleCollections(r.Context(), u.ID, levels)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	out := make([]model.PublicCollection, 0, len(cols))
	for i := range cols {
		c := &cols[i]
		var schema []byte
		if c.CollectionType != nil {
			schema = c.CollectionType.AttributesSchema
		}
		rd := readable{col: c, owner: u, hidden: privacy.HiddenFields(schema, c.HiddenFields)}
		out = append(out, rd.public())
	}
	respondJSON(w, http.StatusOK, out)
}

// ListMembers is the members directory (signed-in readers).
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	users, total, err := h.Store.ListMembers(r.Context(), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	out := make([]model.PublicProfile, 0, len(users))
	for i := range users {
		out = append(out, model.NewPublicProfile(&users[i]))
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(out, total, pg))
}

// --- Liens de partage (propriétaire) ---

// ownCollection loads a collection of the current user (404 otherwise).
func (h *Handler) ownCollection(w http.ResponseWriter, r *http.Request) (*model.Collection, bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return nil, false
	}
	col, err := h.Store.GetCollection(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	if col.UserID != middleware.GetUserID(r.Context()) {
		notFound(w)
		return nil, false
	}
	return col, true
}

func (h *Handler) ListShareLinks(w http.ResponseWriter, r *http.Request) {
	col, ok := h.ownCollection(w, r)
	if !ok {
		return
	}
	links, err := h.Store.ListShareLinks(r.Context(), col.ID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, links)
}

func (h *Handler) CreateShareLink(w http.ResponseWriter, r *http.Request) {
	col, ok := h.ownCollection(w, r)
	if !ok {
		return
	}
	token, err := newToken()
	if err != nil {
		respondInternal(w, r, err, "token generation failed")
		return
	}
	link, err := h.Store.CreateShareLink(r.Context(), col.ID, hashToken(token))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, model.ShareLinkCreated{ShareLink: *link, Token: token})
}

func (h *Handler) RevokeShareLink(w http.ResponseWriter, r *http.Request) {
	col, ok := h.ownCollection(w, r)
	if !ok {
		return
	}
	linkID, ok := parseUUID(w, r, "linkId")
	if !ok {
		return
	}
	if err := h.Store.RevokeShareLink(r.Context(), col.ID, linkID); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
