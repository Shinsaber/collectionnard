package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
	"github.com/shinsaber/colec/internal/store"
)

// Recommendations and blocks (I-4, docs/phases/interactions.md). A member
// recommends an item to a member of the directory, who may refuse them all or
// block the sender: both answer the same 403, the sender cannot tell them apart.
// A blocked member can no longer comment on or react to the blocker's signed
// contents; the blocker no longer sees theirs. On a content that does not name
// its author (anonymous review), a block only mutes: refusing would name them.

const (
	maxRecommendationMessage = 500
	rateRecommendations      = 20 // par jour
)

// CreateRecommendation answers POST /items/{id}/recommendations.
func (h *Handler) CreateRecommendation(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.RecommendationCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	in.To, in.Message = strings.TrimSpace(in.To), strings.TrimSpace(in.Message)
	switch {
	case in.To == "":
		respondValidationError(w, "to: required", map[string]any{"fields": map[string]string{"to": "required"}})
		return
	case utf8.RuneCountInString(in.Message) > maxRecommendationMessage:
		respondValidationError(w, "message: at most 500 characters",
			map[string]any{"fields": map[string]string{"message": "at most 500 characters"}})
		return
	}
	if _, err := h.Store.GetItem(r.Context(), itemID); err != nil {
		handleStoreError(w, r, err)
		return
	}
	viewer, err := h.viewer(r)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	me := viewer.ID
	to, _, err := h.Store.GetUserByUsername(r.Context(), in.To)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if to.ID == me {
		respondValidationError(w, "to: you cannot recommend an item to yourself",
			map[string]any{"fields": map[string]string{"to": "yourself"}})
		return
	}
	if to.DisabledAt != nil || !privacy.CanView(to.ProfileVisibility, to.ID, viewer) {
		notFound(w) // hors de l'annuaire : comme inconnu
		return
	}
	blocked, err := h.Store.IsBlocked(r.Context(), to.ID, me)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if blocked || !to.AcceptsRecommendations {
		respondError(w, http.StatusForbidden, "recommendations_refused", "this member does not accept recommendations")
		return
	}
	if h.limitedIn(w, r, "recommendations", rateRecommendations, rateDay, "day") {
		return
	}
	rc, err := h.Store.CreateRecommendation(r.Context(), me, to.ID, itemID, in.Message)
	if errors.Is(err, store.ErrConflict) {
		respondError(w, http.StatusConflict, "already_recommended", "you already recommended this item to this member")
		return
	}
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	payload := map[string]any{"recommendation_id": rc.ID, "item_id": itemID}
	if rc.Sender != nil {
		payload["by"] = rc.Sender.Username
	}
	if err := h.Store.CreateNotifications(r.Context(), []uuid.UUID{to.ID}, model.NotifyRecommendation, payload); err != nil {
		slog.WarnContext(r.Context(), "notification not recorded", "kind", model.NotifyRecommendation, "err", err)
	}
	respondJSON(w, http.StatusCreated, rc)
}

// ListMyRecommendations answers GET /me/recommendations (received, newest first).
func (h *Handler) ListMyRecommendations(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	list, total, unread, err := h.Store.ListRecommendations(r.Context(), middleware.GetUserID(r.Context()), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.RecommendationPage{
		PaginatedResponse: model.NewPaginatedResponse(list, total, pg), Unread: unread,
	})
}

// MarkRecommendationsRead answers POST /me/recommendations/read (all when ids is empty).
func (h *Handler) MarkRecommendationsRead(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(w, r, &in); err != nil {
			respondDecodeError(w, err)
			return
		}
	}
	if err := h.Store.MarkRecommendationsRead(r.Context(), middleware.GetUserID(r.Context()), in.IDs); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteMyRecommendation dismisses a received recommendation.
func (h *Handler) DeleteMyRecommendation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteRecommendation(r.Context(), id, middleware.GetUserID(r.Context())); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Blocks ---

// shapeBlock withholds the name of a member the blocker may not know.
func shapeBlock(b *model.Block, viewer privacy.Viewer) {
	if b.Anonymous || !privacy.CanView(b.BlockedProfile, b.BlockedID, viewer) {
		b.User = nil
	}
}

// ListBlocks answers GET /me/blocks.
func (h *Handler) ListBlocks(w http.ResponseWriter, r *http.Request) {
	viewer, err := h.viewer(r)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	pg := parsePagination(r)
	list, total, err := h.Store.ListBlocks(r.Context(), viewer.ID, pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	for i := range list {
		shapeBlock(&list[i], viewer)
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

// CreateBlock answers POST /me/blocks: a member named, or the author of a
// content shown to the reader.
func (h *Handler) CreateBlock(w http.ResponseWriter, r *http.Request) {
	var in model.BlockCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	viewer, err := h.viewer(r)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	me := viewer.ID
	var target uuid.UUID
	var named bool
	switch in.Username = strings.TrimSpace(in.Username); {
	case in.Username != "" && in.ContentID == nil:
		u, _, err := h.Store.GetUserByUsername(r.Context(), in.Username)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		if u.DisabledAt != nil || !privacy.CanView(u.ProfileVisibility, u.ID, viewer) {
			notFound(w)
			return
		}
		target, named = u.ID, true
	case in.Username == "" && in.ContentID != nil:
		var shown bool
		var err error
		target, named, shown, err = h.blockTarget(r, in.ContentKind, *in.ContentID, viewer)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		if !shown {
			notFound(w)
			return
		}
	default:
		respondError(w, http.StatusBadRequest, "bad_request", "username, or content_kind and content_id")
		return
	}
	if target == me {
		respondValidationError(w, "you cannot block yourself", map[string]any{"fields": map[string]string{"username": "yourself"}})
		return
	}
	b, err := h.Store.BlockUser(r.Context(), me, target, !named)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	shapeBlock(b, viewer)
	respondJSON(w, http.StatusCreated, b)
}

// blockTarget returns the author of a content, whether the reader knows who
// they are, and whether the content is shown to the reader.
func (h *Handler) blockTarget(r *http.Request, kind string, id uuid.UUID, viewer privacy.Viewer) (uuid.UUID, bool, bool, error) {
	ctx := r.Context()
	switch kind {
	case model.BlockFromRecommendation:
		rc, err := h.Store.GetRecommendation(ctx, id, viewer.ID)
		if err != nil {
			return uuid.Nil, false, false, err
		}
		return rc.SenderID, true, true, nil
	case model.ReportTargetReview:
		rv, err := h.Store.GetReview(ctx, id)
		if err != nil {
			return uuid.Nil, false, false, err
		}
		return rv.UserID, privacy.CanView(rv.Visibility, rv.UserID, viewer), reviewShownTo(rv, viewer), nil
	case model.ReportTargetArticle:
		a, err := h.Store.GetArticle(ctx, id)
		if err != nil {
			return uuid.Nil, false, false, err
		}
		return a.AuthorID, privacy.CanView(a.AuthorProfile, a.AuthorID, viewer), articleShownTo(a, viewer), nil
	case model.ReportTargetComment:
		c, err := h.Store.GetComment(ctx, id)
		if err != nil {
			return uuid.Nil, false, false, err
		}
		author, shown, err := h.reportable(r, kind, id, viewer)
		if err != nil || !shown {
			return author, false, false, err
		}
		named := privacy.CanView(c.AuthorProfile, c.AuthorID, viewer)
		if c.TargetKind == model.CommentTargetReview && named {
			// L'auteur d'un avis anonyme qui répond dans son fil n'est pas nommé.
			rv, err := h.Store.GetReview(ctx, c.TargetID)
			if err != nil {
				return uuid.Nil, false, false, err
			}
			named = rv.UserID != c.AuthorID || rv.Visibility != privacy.Anonymous
		}
		return author, named, true, nil
	}
	return uuid.Nil, false, false, store.ErrNotFound
}

// DeleteBlock answers DELETE /me/blocks/{id}.
func (h *Handler) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteBlock(r.Context(), middleware.GetUserID(r.Context()), id); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// refuseBlocked answers 403 blocked when author blocked the reader and the
// content names its author to them (otherwise the block only mutes).
func (h *Handler) refuseBlocked(w http.ResponseWriter, r *http.Request, author uuid.UUID, named bool) bool {
	if !named {
		return false
	}
	blocked, err := h.Store.IsBlocked(r.Context(), author, middleware.GetUserID(r.Context()))
	if err != nil {
		handleStoreError(w, r, err)
		return true
	}
	if blocked {
		respondError(w, http.StatusForbidden, "blocked", "the author of this content does not accept your interactions")
		return true
	}
	return false
}

// blockedSet returns the members the reader blocked.
func (h *Handler) blockedSet(r *http.Request, viewer uuid.UUID) (map[uuid.UUID]bool, error) {
	ids, err := h.Store.BlockedBy(r.Context(), viewer)
	if err != nil {
		return nil, err
	}
	set := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// withoutBlockers removes from users those who blocked the emitter (no
// notification from them); on error the list is kept and the error logged.
func (h *Handler) withoutBlockers(r *http.Request, emitter uuid.UUID, users []uuid.UUID) []uuid.UUID {
	blockers, err := h.Store.Blockers(r.Context(), emitter, users)
	if err != nil {
		slog.WarnContext(r.Context(), "blockers unknown", "err", err)
		return users
	}
	if len(blockers) == 0 {
		return users
	}
	out := users[:0:0]
	for _, u := range users {
		if !slices.Contains(blockers, u) {
			out = append(out, u)
		}
	}
	return out
}
