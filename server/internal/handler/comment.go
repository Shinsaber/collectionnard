package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
)

// Comments (I-2, docs/phases/interactions.md): the thread of a review follows
// its visibility; a rating only has none. In the thread of an anonymous
// review, its author is "the author of the review", never named.

const (
	maxCommentBody = 5000
	rateComments   = 120
)

// threadVisibleTo tells whether the thread of a review is open to the reader.
func threadVisibleTo(rv *model.Review, viewer uuid.UUID) bool {
	switch {
	case rv.Visibility == privacy.Private, rv.Visibility == privacy.Anonymous && rv.Body == "":
		return false // une note seule n'a pas de fil
	case viewer != uuid.Nil && rv.UserID == viewer:
		return true
	}
	return reviewShownTo(rv, viewer)
}

// shapeComments sets what the reader may see of each comment.
func shapeComments(list []model.Comment, rv *model.Review, viewer uuid.UUID) {
	for i := range list {
		c := &list[i]
		c.Mine = viewer != uuid.Nil && c.AuthorID == viewer
		c.ReviewAuthor = c.AuthorID == rv.UserID
		if c.ReviewAuthor && rv.Visibility == privacy.Anonymous && !c.Mine {
			c.Author = nil
		}
	}
}

// loadThread loads the review of a thread visible to the reader (404 otherwise).
func (h *Handler) loadThread(w http.ResponseWriter, r *http.Request) (*model.Review, bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return nil, false
	}
	rv, err := h.Store.GetReview(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	if !threadVisibleTo(rv, middleware.GetUserID(r.Context())) {
		notFound(w)
		return nil, false
	}
	return rv, true
}

func checkCommentBody(w http.ResponseWriter, in *model.CommentBody) bool {
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" || utf8.RuneCountInString(in.Body) > maxCommentBody {
		respondValidationError(w, "body: 1 to 5000 characters", map[string]any{"fields": map[string]string{"body": "1 to 5000 characters"}})
		return false
	}
	return true
}

// ListReviewComments returns the thread of a review, oldest first (optional authentication).
func (h *Handler) ListReviewComments(w http.ResponseWriter, r *http.Request) {
	rv, ok := h.loadThread(w, r)
	if !ok {
		return
	}
	viewer := middleware.GetUserID(r.Context())
	pg := parsePaginationDefault(r, 50)
	list, total, err := h.Store.ListComments(r.Context(), model.CommentTargetReview, rv.ID, viewer, pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	shapeComments(list, rv, viewer)
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

// CreateReviewComment adds a message to the thread of a review.
func (h *Handler) CreateReviewComment(w http.ResponseWriter, r *http.Request) {
	rv, ok := h.loadThread(w, r)
	if !ok {
		return
	}
	var in model.CommentBody
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if !checkCommentBody(w, &in) {
		return
	}
	if rv.CommentsClosed || rv.HiddenAt != nil {
		respondError(w, http.StatusConflict, "comments_closed", "the thread of this review is closed")
		return
	}
	if h.limited(w, r, "comments", rateComments) {
		return
	}
	me := middleware.GetUserID(r.Context())
	c, err := h.Store.CreateComment(r.Context(), model.CommentTargetReview, rv.ID, me, in.Body)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	one := []model.Comment{*c}
	shapeComments(one, rv, me)
	h.notifyThread(r, rv)
	respondJSON(w, http.StatusCreated, one[0])
}

// notifyThread tells the author of the review and the participants of its
// thread, never the emitter; the author of an anonymous review stays unnamed.
func (h *Handler) notifyThread(r *http.Request, rv *model.Review) {
	me := middleware.GetUserID(r.Context())
	users, err := h.Store.CommentParticipants(r.Context(), model.CommentTargetReview, rv.ID)
	if err != nil {
		slog.WarnContext(r.Context(), "notification recipients unknown", "review", rv.ID, "err", err)
	}
	seen := map[uuid.UUID]bool{me: true}
	var to []uuid.UUID
	for _, u := range append([]uuid.UUID{rv.UserID}, users...) {
		if !seen[u] {
			seen[u] = true
			to = append(to, u)
		}
	}
	if len(to) == 0 {
		return
	}
	payload := map[string]any{"item_id": rv.ItemID, "review_id": rv.ID}
	if me == rv.UserID && rv.Visibility == privacy.Anonymous {
		payload["by_review_author"] = true
	} else if u, err := h.Store.GetUserByID(r.Context(), me); err == nil {
		payload["by"] = u.Username
	}
	if err := h.Store.CreateNotifications(r.Context(), to, model.NotifyReviewComment, payload); err != nil {
		slog.WarnContext(r.Context(), "notification not recorded", "kind", model.NotifyReviewComment, "err", err)
	}
}

// UpdateComment replaces the text of one's own comment.
func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.CommentBody
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if !checkCommentBody(w, &in) {
		return
	}
	me := middleware.GetUserID(r.Context())
	c, err := h.Store.UpdateComment(r.Context(), id, me, in.Body)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	c.Mine = true
	respondJSON(w, http.StatusOK, c)
}

// DeleteComment removes one's own comment.
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteComment(r.Context(), id, middleware.GetUserID(r.Context())); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CloseReviewComments(w http.ResponseWriter, r *http.Request) {
	h.setCommentsClosed(w, r, true)
}

func (h *Handler) OpenReviewComments(w http.ResponseWriter, r *http.Request) {
	h.setCommentsClosed(w, r, false)
}

// setCommentsClosed: only the author of the review closes or reopens its thread.
func (h *Handler) setCommentsClosed(w http.ResponseWriter, r *http.Request, closed bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.SetReviewCommentsClosed(r.Context(), id, middleware.GetUserID(r.Context()), closed); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
