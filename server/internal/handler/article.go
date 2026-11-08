package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
	"github.com/shinsaber/colec/internal/review"
	"github.com/shinsaber/colec/internal/store"
)

// Articles (I-3, docs/phases/interactions.md): news about an item, on every
// item type. The article of a member waits for a moderator before it is
// published (and again after an edit); the admins and moderators publish
// directly. The server never fetches the link (no preview: SSRF).

const (
	maxArticleTitle   = 200
	maxArticleURL     = 2000
	maxArticleMessage = 1000
	rateArticles      = 10
)

// checkArticle trims and checks an article; on error, field names the culprit.
func checkArticle(in *model.ArticleWrite) (field, msg string) {
	in.Title, in.URL, in.Message = strings.TrimSpace(in.Title), strings.TrimSpace(in.URL), strings.TrimSpace(in.Message)
	switch {
	case in.Title == "" || utf8.RuneCountInString(in.Title) > maxArticleTitle:
		return "title", "title: 1 to 200 characters"
	case !validArticleURL(in.URL):
		return "url", "url: an http(s) address of 2000 characters at most"
	case utf8.RuneCountInString(in.Message) > maxArticleMessage:
		return "message", "message: 1000 characters at most"
	case !review.ValidLang(in.Lang):
		return "lang", "lang must be a language of the interface"
	}
	return "", ""
}

func validArticleURL(raw string) bool {
	if len(raw) > maxArticleURL {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil
}

// shapeArticles marks the reader's articles and keeps the refusal reason for them only.
func (h *Handler) shapeArticles(list []model.Article, r *http.Request) {
	viewer := h.viewerOrStranger(r)
	for i := range list {
		list[i].Mine = viewer.Is(list[i].AuthorID)
		if !list[i].Mine {
			list[i].RejectReason, list[i].HiddenReason = nil, nil
			if !privacy.CanView(list[i].AuthorProfile, list[i].AuthorID, viewer) {
				list[i].Author = nil // profil non visible du lecteur : « Un membre »
			}
		}
	}
}

// ListItemArticles returns the published articles of an item and the reader's own (optional authentication).
func (h *Handler) ListItemArticles(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := h.Store.GetItem(r.Context(), id); err != nil {
		handleStoreError(w, r, err)
		return
	}
	pg := parsePagination(r)
	list, total, err := h.Store.ListItemArticles(r.Context(), id, middleware.GetUserID(r.Context()), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	h.shapeArticles(list, r)
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

// articleStatus: published directly by the admins and moderators, otherwise waiting.
func articleStatus(r *http.Request) string {
	if model.CanModerate(middleware.GetRole(r.Context())) {
		return model.ArticlePublished
	}
	return model.ArticlePending
}

func (h *Handler) CreateItemArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.ArticleWrite
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if field, msg := checkArticle(&in); field != "" {
		respondValidationError(w, msg, map[string]any{"fields": map[string]string{field: msg}})
		return
	}
	if h.limited(w, r, "articles", rateArticles) {
		return
	}
	a, err := h.Store.CreateArticle(r.Context(), id, middleware.GetUserID(r.Context()), in, articleStatus(r))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	a.Mine = true
	respondJSON(w, http.StatusCreated, a)
}

func (h *Handler) UpdateArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.ArticleWrite
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if field, msg := checkArticle(&in); field != "" {
		respondValidationError(w, msg, map[string]any{"fields": map[string]string{field: msg}})
		return
	}
	if h.limited(w, r, "articles", rateArticles) {
		return
	}
	a, err := h.Store.UpdateArticle(r.Context(), id, middleware.GetUserID(r.Context()), in, articleStatus(r))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	a.Mine = true
	respondJSON(w, http.StatusOK, a)
}

func (h *Handler) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteArticle(r.Context(), id, middleware.GetUserID(r.Context())); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListPendingArticles: the articles waiting for a moderator, oldest first.
func (h *Handler) ListPendingArticles(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	list, total, err := h.Store.ListPendingArticles(r.Context(), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

func (h *Handler) PublishArticle(w http.ResponseWriter, r *http.Request) {
	h.reviewArticle(w, r, true)
}

func (h *Handler) RejectArticle(w http.ResponseWriter, r *http.Request) {
	h.reviewArticle(w, r, false)
}

// reviewArticle publishes or refuses a waiting article (a refusal needs a
// reason, shown to its author); never one's own.
func (h *Handler) reviewArticle(w http.ResponseWriter, r *http.Request, publish bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.ModerationActionBody
	if r.ContentLength != 0 {
		if err := decodeJSON(w, r, &in); err != nil {
			respondDecodeError(w, err)
			return
		}
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if !publish && in.Reason == "" {
		respondValidationError(w, "reason is required", map[string]any{"fields": map[string]string{"reason": "required"}})
		return
	}
	me := middleware.GetUserID(r.Context())
	a, err := h.Store.GetArticle(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if a.AuthorID == me {
		respondError(w, http.StatusForbidden, "own_content", "nobody reviews their own article")
		return
	}
	if a, err = h.Store.ReviewArticle(r.Context(), id, me, publish, in.Reason); err != nil {
		if errors.Is(err, store.ErrConflict) {
			respondError(w, http.StatusConflict, "moderation_state", "this article is not waiting any more")
			return
		}
		handleStoreError(w, r, err)
		return
	}
	kind := model.NotifyArticlePublished
	if !publish {
		kind = model.NotifyArticleRejected
	}
	h.notifyModeration(r, &model.ModerationResult{AuthorID: a.AuthorID, ItemID: a.ItemID}, kind, in.Reason)
	w.WriteHeader(http.StatusNoContent)
}
