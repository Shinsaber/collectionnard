package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
	"github.com/shinsaber/colec/internal/store"
)

// Friends (P7-2, docs/phases/exemplaires-prets.md). A member asks another
// whose profile they see, by username (any other name answers 404, as an
// unknown one: usernames cannot be probed); the other accepts or refuses. A
// private profile hands out a secret single-use link instead (friend links).
// The friendship opens the `friends` visibility of collections, profiles and
// reviews. A member blocked by the addressee gets no answer: their request
// stays pending, hidden from the addressee and never notified.

const (
	rateFriendRequests = 20 // par jour
	rateFriendLinks    = 20 // par jour
	maxFriendLinks     = 10 // liens valides à la fois
	friendLinkDays     = 7
)

// viewer loads the reader of the request with their friends.
func (h *Handler) viewer(r *http.Request) (privacy.Viewer, error) {
	me := middleware.GetUserID(r.Context())
	friends, err := h.Store.FriendIDs(r.Context(), me)
	if err != nil {
		return privacy.Viewer{ID: me}, err
	}
	return privacy.NewViewer(me, friends), nil
}

// viewerOrStranger is viewer for the reads that only name people: when the
// friends cannot be loaded, the reader is treated as a stranger to everyone
// (less shown, nothing disclosed).
func (h *Handler) viewerOrStranger(r *http.Request) privacy.Viewer {
	v, err := h.viewer(r)
	if err != nil {
		slog.WarnContext(r.Context(), "friends not loaded", "err", err)
	}
	return v
}

// shapeFriendship tells whether the reader (me) may open the other member's
// profile, and withholds what they may not know yet of a member who did not
// accept them: a sent request names only the username they typed.
func shapeFriendship(f *model.Friendship, me uuid.UUID) {
	var friends []uuid.UUID
	if f.Status == model.FriendAccepted {
		friends = []uuid.UUID{f.OtherID}
	}
	f.ProfileVisible = privacy.CanView(f.OtherProfile, f.OtherID, privacy.NewViewer(me, friends))
	if f.Status == model.FriendPending && !f.Incoming && !f.ProfileVisible {
		f.User = &model.ReviewAuthor{Username: f.User.Username}
	}
}

func (h *Handler) listFriendships(w http.ResponseWriter, r *http.Request, status string) {
	me := middleware.GetUserID(r.Context())
	pg := parsePagination(r)
	list, total, err := h.Store.ListFriendships(r.Context(), me, status, pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	for i := range list {
		shapeFriendship(&list[i], me)
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

// ListFriends answers GET /me/friends.
func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {
	h.listFriendships(w, r, model.FriendAccepted)
}

// ListFriendRequests answers GET /me/friend-requests: received, then sent.
func (h *Handler) ListFriendRequests(w http.ResponseWriter, r *http.Request) {
	h.listFriendships(w, r, model.FriendPending)
}

// RequestFriend answers POST /me/friends: a request, or the acceptance of the
// request the other member already sent.
func (h *Handler) RequestFriend(w http.ResponseWriter, r *http.Request) {
	var in model.FriendRequestCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Username = strings.TrimSpace(in.Username); in.Username == "" {
		respondValidationError(w, "username: required", map[string]any{"fields": map[string]string{"username": "required"}})
		return
	}
	viewer, err := h.viewer(r)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	me := viewer.ID
	to, _, err := h.Store.GetUserByUsername(r.Context(), in.Username)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if to.DisabledAt != nil || !privacy.CanView(to.ProfileVisibility, to.ID, viewer) {
		notFound(w) // profil qui ne m'est pas visible : comme inconnu
		return
	}
	if to.ID == me {
		respondValidationError(w, "username: you cannot be your own friend",
			map[string]any{"fields": map[string]string{"username": "yourself"}})
		return
	}
	iBlocked, err := h.Store.IsBlocked(r.Context(), me, to.ID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if iBlocked {
		respondError(w, http.StatusConflict, "member_blocked", "you blocked this member: lift the block first")
		return
	}
	muted, err := h.Store.IsBlocked(r.Context(), to.ID, me)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if h.limitedIn(w, r, "friend_requests", rateFriendRequests, rateDay, "day") {
		return
	}
	f, err := h.Store.RequestFriend(r.Context(), me, to.ID)
	if errors.Is(err, store.ErrConflict) {
		respondError(w, http.StatusConflict, "friendship_exists", "already friends, or request already sent")
		return
	}
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if !muted {
		kind := model.NotifyFriendRequest
		if f.Status == model.FriendAccepted {
			kind = model.NotifyFriendAccepted
		}
		h.notifyFriend(r, to.ID, kind, f.ID)
	}
	shapeFriendship(f, me)
	respondJSON(w, http.StatusCreated, f)
}

// AcceptFriendRequest answers POST /me/friend-requests/{id}/accept.
func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	me := middleware.GetUserID(r.Context())
	f, err := h.Store.AcceptFriendRequest(r.Context(), me, id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	shapeFriendship(f, me)
	h.notifyFriend(r, f.OtherID, model.NotifyFriendAccepted, f.ID)
	respondJSON(w, http.StatusOK, f)
}

// DeleteFriendship answers DELETE /me/friends/{id}: refusal, cancellation or
// break, never notified.
func (h *Handler) DeleteFriendship(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteFriendship(r.Context(), middleware.GetUserID(r.Context()), id); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// notifyFriend tells the other member; they know the emitter by name, since
// the emitter asked them or accepted their request.
func (h *Handler) notifyFriend(r *http.Request, to uuid.UUID, kind string, id uuid.UUID) {
	payload := map[string]any{"friendship_id": id}
	if u, err := h.Store.GetUserByID(r.Context(), middleware.GetUserID(r.Context())); err == nil {
		payload["by"] = u.Username
	}
	if err := h.Store.CreateNotifications(r.Context(), []uuid.UUID{to}, kind, payload); err != nil {
		slog.WarnContext(r.Context(), "notification not recorded", "kind", kind, "err", err)
	}
}

// --- Liens d'ami ---

// ListFriendLinks answers GET /me/friend-links (valid links, without their token).
func (h *Handler) ListFriendLinks(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListFriendLinks(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	for i := range list {
		list[i].Owner = nil
	}
	respondJSON(w, http.StatusOK, list)
}

// CreateFriendLink answers POST /me/friend-links: the token is shown this once.
func (h *Handler) CreateFriendLink(w http.ResponseWriter, r *http.Request) {
	me := middleware.GetUserID(r.Context())
	list, err := h.Store.ListFriendLinks(r.Context(), me)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if len(list) >= maxFriendLinks {
		respondError(w, http.StatusConflict, "friend_links_limit", "too many valid friend links: revoke one first")
		return
	}
	if h.limitedIn(w, r, "friend_links", rateFriendLinks, rateDay, "day") {
		return
	}
	token, err := newToken()
	if err != nil {
		respondInternal(w, r, err, "token generation failed")
		return
	}
	l, err := h.Store.CreateFriendLink(r.Context(), me, hashToken(token), time.Now().Add(friendLinkDays*24*time.Hour))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	l.Token, l.Owner = token, nil
	respondJSON(w, http.StatusCreated, l)
}

// DeleteFriendLink answers DELETE /me/friend-links/{id}.
func (h *Handler) DeleteFriendLink(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteFriendLink(r.Context(), middleware.GetUserID(r.Context()), id); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// openFriendLink loads the valid link of the route's token; a link of a member
// who blocked the reader answers 404, as an invalid one.
func (h *Handler) openFriendLink(w http.ResponseWriter, r *http.Request) (*model.FriendLink, string, bool) {
	hash := hashToken(chi.URLParam(r, "token"))
	l, err := h.Store.GetFriendLink(r.Context(), hash)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, "", false
	}
	blocked, err := h.Store.IsBlocked(r.Context(), l.OwnerID, middleware.GetUserID(r.Context()))
	if err != nil {
		handleStoreError(w, r, err)
		return nil, "", false
	}
	if blocked {
		notFound(w)
		return nil, "", false
	}
	return l, hash, true
}

// GetFriendLink answers GET /friend-links/{token}: who invites the reader.
func (h *Handler) GetFriendLink(w http.ResponseWriter, r *http.Request) {
	if l, _, ok := h.openFriendLink(w, r); ok {
		respondJSON(w, http.StatusOK, l)
	}
}

// AcceptFriendLink answers POST /friend-links/{token}/accept: the reader
// becomes a friend of the link's owner, who is notified; the link is used up.
func (h *Handler) AcceptFriendLink(w http.ResponseWriter, r *http.Request) {
	l, hash, ok := h.openFriendLink(w, r)
	if !ok {
		return
	}
	me := middleware.GetUserID(r.Context())
	if l.OwnerID == me {
		respondValidationError(w, "this is your own link", map[string]any{"fields": map[string]string{"token": "yourself"}})
		return
	}
	iBlocked, err := h.Store.IsBlocked(r.Context(), me, l.OwnerID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if iBlocked {
		respondError(w, http.StatusConflict, "member_blocked", "you blocked this member: lift the block first")
		return
	}
	f, err := h.Store.UseFriendLink(r.Context(), hash, me)
	if errors.Is(err, store.ErrConflict) {
		respondError(w, http.StatusConflict, "friendship_exists", "already friends")
		return
	}
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	shapeFriendship(f, me)
	h.notifyFriend(r, l.OwnerID, model.NotifyFriendAccepted, f.ID)
	respondJSON(w, http.StatusOK, f)
}
