package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
	"github.com/shinsaber/colec/internal/store"
)

// Reports, moderation and reactions (I-1, docs/phases/interactions.md). The
// admins and the moderators handle the reports; every action goes to the
// journal; a reaction never sends a review to the moderation.

const (
	maxReportDetails  = 1000
	maxModerateReason = 500
)

// reviewShownTo tells whether a review appears to the viewer (listed, not
// hidden): what may be reported or reacted to. Their own review is not.
func reviewShownTo(r *model.Review, viewer uuid.UUID) bool {
	if r.HiddenAt != nil || r.UserID == viewer {
		return false
	}
	if r.Visibility == privacy.Anonymous {
		return r.Body != ""
	}
	return slices.Contains(privacy.Levels(viewer), r.Visibility)
}

// CreateReport records a report on a content shown to the reporter.
func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	var in model.ReportCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	in.Details = strings.TrimSpace(in.Details)
	switch {
	case in.TargetKind != model.ReportTargetReview && in.TargetKind != model.ReportTargetComment &&
		in.TargetKind != model.ReportTargetArticle:
		respondValidationError(w, "target_kind must be review, comment or article", map[string]any{"fields": map[string]string{"target_kind": "unsupported"}})
		return
	case !slices.Contains(model.ReportReasons, in.Reason):
		respondValidationError(w, "unknown reason", map[string]any{"fields": map[string]string{"reason": "unknown"}})
		return
	case utf8.RuneCountInString(in.Details) > maxReportDetails:
		respondValidationError(w, "details too long", map[string]any{"fields": map[string]string{"details": "too long"}})
		return
	}
	me := middleware.GetUserID(r.Context())
	author, shown, err := h.reportable(r, in.TargetKind, in.TargetID, me)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if author == me {
		respondError(w, http.StatusForbidden, "own_content", "you cannot report your own content")
		return
	}
	if !shown {
		notFound(w)
		return
	}
	if h.limited(w, r, "reports", rateReports) {
		return
	}
	rep, err := h.Store.CreateReport(r.Context(), me, in)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			respondError(w, http.StatusConflict, "report_exists", "you already reported this content")
			return
		}
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, rep)
}

// reportable returns the author of a content and whether the reader sees it:
// a review listed for them, or a comment of a thread open to them.
func (h *Handler) reportable(r *http.Request, kind string, id, viewer uuid.UUID) (uuid.UUID, bool, error) {
	switch kind {
	case model.ReportTargetArticle:
		a, err := h.Store.GetArticle(r.Context(), id)
		if err != nil {
			return uuid.Nil, false, err
		}
		return a.AuthorID, a.AuthorID != viewer && articleShownTo(a, viewer), nil
	case model.ReportTargetComment:
		c, err := h.Store.GetComment(r.Context(), id)
		if err != nil {
			return uuid.Nil, false, err
		}
		if c.HiddenAt != nil || c.DeletedAt != nil {
			return c.AuthorID, false, nil
		}
		if c.TargetKind == model.CommentTargetArticle {
			a, err := h.Store.GetArticle(r.Context(), c.TargetID)
			if err != nil {
				return uuid.Nil, false, err
			}
			return c.AuthorID, articleShownTo(a, viewer), nil
		}
		rv, err := h.Store.GetReview(r.Context(), c.TargetID)
		if err != nil {
			return uuid.Nil, false, err
		}
		return c.AuthorID, threadVisibleTo(rv, viewer), nil
	}
	rv, err := h.Store.GetReview(r.Context(), id)
	if err != nil {
		return uuid.Nil, false, err
	}
	return rv.UserID, reviewShownTo(rv, viewer), nil
}

func (h *Handler) CountModerationQueue(w http.ResponseWriter, r *http.Request) {
	n, err := h.Store.CountModerationQueue(r.Context())
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	pending, err := h.Store.CountPendingArticles(r.Context())
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]int{"count": n, "pending_articles": pending})
}

func (h *Handler) ListModerationQueue(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, total, err := h.Store.ListModerationQueue(r.Context(), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if items == nil {
		items = []model.ModerationItem{}
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(items, total, pg))
}

func (h *Handler) HideReview(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, model.ReportTargetReview, model.ModerationHide)
}

func (h *Handler) RestoreReview(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, model.ReportTargetReview, model.ModerationRestore)
}

func (h *Handler) DismissReviewReports(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, model.ReportTargetReview, model.ModerationDismiss)
}

func (h *Handler) HideArticle(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, model.ReportTargetArticle, model.ModerationHide)
}

func (h *Handler) RestoreArticle(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, model.ReportTargetArticle, model.ModerationRestore)
}

func (h *Handler) DismissArticleReports(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, model.ReportTargetArticle, model.ModerationDismiss)
}

func (h *Handler) HideComment(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, model.ReportTargetComment, model.ModerationHide)
}

func (h *Handler) RestoreComment(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, model.ReportTargetComment, model.ModerationRestore)
}

func (h *Handler) DismissCommentReports(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, model.ReportTargetComment, model.ModerationDismiss)
}

// moderate applies an action to a review or a comment; a hide needs a reason,
// shown to the author.
func (h *Handler) moderate(w http.ResponseWriter, r *http.Request, kind, action string) {
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
	if action == model.ModerationHide && in.Reason == "" {
		respondValidationError(w, "reason is required", map[string]any{"fields": map[string]string{"reason": "required"}})
		return
	}
	if utf8.RuneCountInString(in.Reason) > maxModerateReason {
		respondValidationError(w, "reason too long", map[string]any{"fields": map[string]string{"reason": "too long"}})
		return
	}
	me := middleware.GetUserID(r.Context())
	author, _, err := h.reportable(r, kind, id, me)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if author == me {
		respondError(w, http.StatusForbidden, "own_content", "nobody moderates their own content")
		return
	}
	res, err := h.Store.Moderate(r.Context(), kind, id, me, action, in.Reason)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			respondError(w, http.StatusConflict, "moderation_state", "this action does not apply to the content any more")
			return
		}
		handleStoreError(w, r, err)
		return
	}
	notify := map[string]map[string]string{
		model.ReportTargetReview:  {model.ModerationHide: model.NotifyReviewHidden, model.ModerationRestore: model.NotifyReviewRestored},
		model.ReportTargetComment: {model.ModerationHide: model.NotifyCommentHidden, model.ModerationRestore: model.NotifyCommentRestore},
		model.ReportTargetArticle: {model.ModerationHide: model.NotifyArticleHidden, model.ModerationRestore: model.NotifyArticleRestored},
	}
	if n := notify[kind][action]; n != "" {
		h.notifyModeration(r, res, n, in.Reason)
	}
	w.WriteHeader(http.StatusNoContent)
}

// notifyModeration tells the author; a failure is only logged.
func (h *Handler) notifyModeration(r *http.Request, res *model.ModerationResult, kind, reason string) {
	payload := map[string]any{"item_id": res.ItemID, "reason": reason}
	if err := h.Store.CreateNotifications(r.Context(), []uuid.UUID{res.AuthorID}, kind, payload); err != nil {
		slog.WarnContext(r.Context(), "notification not recorded", "kind", kind, "err", err)
	}
}

func (h *Handler) ListModerationLog(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	entries, total, err := h.Store.ListModerationLog(r.Context(), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(entries, total, pg))
}

func (h *Handler) GetModerationHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	hist, err := h.Store.ModerationHistory(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, hist)
}

// SetReaction adds (PUT) or removes (DELETE) a reaction to a review shown to the reader.
func (h *Handler) SetReaction(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	kind := chi.URLParam(r, "kind")
	if !slices.Contains(model.ReactionKinds, kind) {
		respondError(w, http.StatusBadRequest, "bad_request", "unknown reaction")
		return
	}
	me := middleware.GetUserID(r.Context())
	rv, err := h.Store.GetReview(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if rv.UserID == me {
		respondError(w, http.StatusForbidden, "own_content", "you cannot react to your own review")
		return
	}
	if !reviewShownTo(rv, me) {
		notFound(w)
		return
	}
	if r.Method == http.MethodPut && h.refuseBlocked(w, r, rv.UserID, privacy.CanView(rv.Visibility, rv.UserID, me)) {
		return
	}
	if h.limited(w, r, "reactions", rateReactions) {
		return
	}
	if err := h.Store.SetReaction(r.Context(), id, me, kind, r.Method == http.MethodPut); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// withReactions adds to each review the count of each reaction and those of the viewer.
func (h *Handler) withReactions(r *http.Request, reviews []model.Review) error {
	ids := make([]uuid.UUID, len(reviews))
	for i := range reviews {
		ids[i] = reviews[i].ID
	}
	counts, mine, err := h.Store.ReactionsOf(r.Context(), ids, middleware.GetUserID(r.Context()))
	if err != nil {
		return err
	}
	for i := range reviews {
		reviews[i].Reactions = counts[reviews[i].ID]
		if reviews[i].Reactions == nil {
			reviews[i].Reactions = map[string]int{}
		}
		reviews[i].MyReactions = mine[reviews[i].ID]
		if reviews[i].MyReactions == nil {
			reviews[i].MyReactions = []string{}
		}
	}
	return nil
}
