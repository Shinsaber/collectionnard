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

// Comments (I-2, I-3, docs/phases/interactions.md): one thread per review or
// article. The thread of a review follows its visibility (a rating only has
// none); in the thread of an anonymous review, its author is "the author of
// the review", never named. The thread of an article exists once published.

const (
	maxCommentBody = 5000
	rateComments   = 120
)

// thread is what a comment handler needs of the commented content.
type thread struct {
	Kind     string
	ID       uuid.UUID
	AuthorID uuid.UUID
	ItemID   uuid.UUID
	Closed   bool
	// Hidden: hidden by the moderation (no new comment).
	Hidden bool
	// AnonymousAuthor: its author is not named in the thread.
	AnonymousAuthor bool
	// AuthorNamed: the reader knows who its author is (a block then refuses
	// their comments; otherwise it only mutes them).
	AuthorNamed bool
}

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

// articleShownTo tells whether an article (and its thread) is open to the reader.
func articleShownTo(a *model.Article, viewer uuid.UUID) bool {
	if viewer != uuid.Nil && a.AuthorID == viewer {
		return a.Status == model.ArticlePublished
	}
	return a.Status == model.ArticlePublished && a.HiddenAt == nil
}

// loadThread loads the content of a thread visible to the reader (404 otherwise).
func (h *Handler) loadThread(w http.ResponseWriter, r *http.Request, kind string) (*thread, bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return nil, false
	}
	viewer := middleware.GetUserID(r.Context())
	if kind == model.CommentTargetArticle {
		a, err := h.Store.GetArticle(r.Context(), id)
		if err != nil {
			handleStoreError(w, r, err)
			return nil, false
		}
		if !articleShownTo(a, viewer) {
			notFound(w)
			return nil, false
		}
		return &thread{Kind: kind, ID: a.ID, AuthorID: a.AuthorID, ItemID: a.ItemID, Closed: a.CommentsClosed,
			Hidden: a.HiddenAt != nil, AuthorNamed: privacy.CanView(a.AuthorProfile, a.AuthorID, viewer)}, true
	}
	rv, err := h.Store.GetReview(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	if !threadVisibleTo(rv, viewer) {
		notFound(w)
		return nil, false
	}
	return &thread{Kind: kind, ID: rv.ID, AuthorID: rv.UserID, ItemID: rv.ItemID, Closed: rv.CommentsClosed,
		Hidden: rv.HiddenAt != nil, AnonymousAuthor: rv.Visibility == privacy.Anonymous,
		AuthorNamed: privacy.CanView(rv.Visibility, rv.UserID, viewer)}, true
}

// shapeComments sets what the reader may see of each comment: a deleted
// comment, one hidden by the moderation for anyone but its author, or one of a
// member the reader blocked is a trace. The author of an anonymous review is
// never marked blocked: it would name them.
func shapeComments(list []model.Comment, t *thread, viewer uuid.UUID, blocked map[uuid.UUID]bool) {
	for i := range list {
		c := &list[i]
		c.Mine = viewer != uuid.Nil && c.AuthorID == viewer
		c.TargetAuthor = c.AuthorID == t.AuthorID
		// Traces : ni texte ni auteur, pour que les réponses gardent leur sens.
		if c.DeletedAt != nil || (c.HiddenAt != nil && !c.Mine) {
			c.Body, c.Author, c.HiddenReason, c.EditedAt = "", nil, nil, nil
			c.Mine, c.TargetAuthor = false, false
			continue
		}
		if !c.Mine && blocked[c.AuthorID] && (!c.TargetAuthor || !t.AnonymousAuthor) {
			c.Body, c.Author, c.HiddenAt, c.HiddenReason, c.EditedAt = "", nil, nil, nil, nil
			c.TargetAuthor, c.Blocked = false, true
			continue
		}
		switch {
		case c.Mine:
		case c.TargetAuthor && t.AnonymousAuthor:
			c.Author = nil
		case !privacy.CanView(c.AuthorProfile, c.AuthorID, viewer):
			c.Author = nil // profil non visible du lecteur : « Un membre »
		}
	}
}

func checkCommentBody(w http.ResponseWriter, in *model.CommentBody) bool {
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" || utf8.RuneCountInString(in.Body) > maxCommentBody {
		respondValidationError(w, "body: 1 to 5000 characters", map[string]any{"fields": map[string]string{"body": "1 to 5000 characters"}})
		return false
	}
	return true
}

func (h *Handler) ListReviewComments(w http.ResponseWriter, r *http.Request) {
	h.listComments(w, r, model.CommentTargetReview)
}

func (h *Handler) ListArticleComments(w http.ResponseWriter, r *http.Request) {
	h.listComments(w, r, model.CommentTargetArticle)
}

// listComments returns a thread, oldest first (optional authentication).
func (h *Handler) listComments(w http.ResponseWriter, r *http.Request, kind string) {
	t, ok := h.loadThread(w, r, kind)
	if !ok {
		return
	}
	viewer := middleware.GetUserID(r.Context())
	pg := parsePaginationDefault(r, 50)
	list, total, err := h.Store.ListComments(r.Context(), kind, t.ID, viewer, pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	blocked, err := h.blockedSet(r, viewer)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	shapeComments(list, t, viewer, blocked)
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

func (h *Handler) CreateReviewComment(w http.ResponseWriter, r *http.Request) {
	h.createComment(w, r, model.CommentTargetReview)
}

func (h *Handler) CreateArticleComment(w http.ResponseWriter, r *http.Request) {
	h.createComment(w, r, model.CommentTargetArticle)
}

// createComment adds a message to a thread.
func (h *Handler) createComment(w http.ResponseWriter, r *http.Request, kind string) {
	t, ok := h.loadThread(w, r, kind)
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
	if t.Closed || t.Hidden {
		respondError(w, http.StatusConflict, "comments_closed", "this thread is closed")
		return
	}
	me := middleware.GetUserID(r.Context())
	if t.AuthorID != me && h.refuseBlocked(w, r, t.AuthorID, t.AuthorNamed) {
		return
	}
	if h.limited(w, r, "comments", rateComments) {
		return
	}
	c, err := h.Store.CreateComment(r.Context(), kind, t.ID, me, in.Body)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	one := []model.Comment{*c}
	shapeComments(one, t, me, nil)
	h.notifyThread(r, t)
	respondJSON(w, http.StatusCreated, one[0])
}

// notifyThread tells the author of the content and the participants of its
// thread, never the emitter; the author of an anonymous review stays unnamed.
func (h *Handler) notifyThread(r *http.Request, t *thread) {
	me := middleware.GetUserID(r.Context())
	users, err := h.Store.CommentParticipants(r.Context(), t.Kind, t.ID)
	if err != nil {
		slog.WarnContext(r.Context(), "notification recipients unknown", "target", t.ID, "err", err)
	}
	seen := map[uuid.UUID]bool{me: true}
	var to []uuid.UUID
	for _, u := range append([]uuid.UUID{t.AuthorID}, users...) {
		if !seen[u] {
			seen[u] = true
			to = append(to, u)
		}
	}
	to = h.withoutBlockers(r, me, to)
	if len(to) == 0 {
		return
	}
	kind, payload := model.NotifyReviewComment, map[string]any{"item_id": t.ItemID, "review_id": t.ID}
	if t.Kind == model.CommentTargetArticle {
		kind, payload = model.NotifyArticleComment, map[string]any{"item_id": t.ItemID, "article_id": t.ID}
	}
	if me == t.AuthorID && t.AnonymousAuthor {
		payload["by_review_author"] = true
	} else if u, err := h.Store.GetUserByID(r.Context(), me); err == nil && u.ProfileVisibility != privacy.Private {
		payload["by"] = u.Username // les destinataires sont des membres : profil « membres » ou public
	} else {
		payload["by_member"] = true
	}
	if err := h.Store.CreateNotifications(r.Context(), to, kind, payload); err != nil {
		slog.WarnContext(r.Context(), "notification not recorded", "kind", kind, "err", err)
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
	h.setCommentsClosed(w, r, model.CommentTargetReview, true)
}

func (h *Handler) OpenReviewComments(w http.ResponseWriter, r *http.Request) {
	h.setCommentsClosed(w, r, model.CommentTargetReview, false)
}

func (h *Handler) CloseArticleComments(w http.ResponseWriter, r *http.Request) {
	h.setCommentsClosed(w, r, model.CommentTargetArticle, true)
}

func (h *Handler) OpenArticleComments(w http.ResponseWriter, r *http.Request) {
	h.setCommentsClosed(w, r, model.CommentTargetArticle, false)
}

// setCommentsClosed: only the author of the content closes or reopens its thread.
func (h *Handler) setCommentsClosed(w http.ResponseWriter, r *http.Request, kind string, closed bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	me := middleware.GetUserID(r.Context())
	var err error
	if kind == model.CommentTargetArticle {
		err = h.Store.SetArticleCommentsClosed(r.Context(), id, me, closed)
	} else {
		err = h.Store.SetReviewCommentsClosed(r.Context(), id, me, closed)
	}
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
