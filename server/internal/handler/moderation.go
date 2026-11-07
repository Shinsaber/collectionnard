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
	case in.TargetKind != model.ReportTargetReview:
		respondValidationError(w, "target_kind must be review", map[string]any{"fields": map[string]string{"target_kind": "unsupported"}})
		return
	case !slices.Contains(model.ReportReasons, in.Reason):
		respondValidationError(w, "unknown reason", map[string]any{"fields": map[string]string{"reason": "unknown"}})
		return
	case utf8.RuneCountInString(in.Details) > maxReportDetails:
		respondValidationError(w, "details too long", map[string]any{"fields": map[string]string{"details": "too long"}})
		return
	}
	me := middleware.GetUserID(r.Context())
	rv, err := h.Store.GetReview(r.Context(), in.TargetID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if rv.UserID == me {
		respondError(w, http.StatusForbidden, "own_content", "you cannot report your own content")
		return
	}
	if !reviewShownTo(rv, me) {
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

func (h *Handler) CountModerationQueue(w http.ResponseWriter, r *http.Request) {
	n, err := h.Store.CountModerationQueue(r.Context())
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]int{"count": n})
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
	h.moderateReview(w, r, model.ModerationHide)
}

func (h *Handler) RestoreReview(w http.ResponseWriter, r *http.Request) {
	h.moderateReview(w, r, model.ModerationRestore)
}

func (h *Handler) DismissReviewReports(w http.ResponseWriter, r *http.Request) {
	h.moderateReview(w, r, model.ModerationDismiss)
}

// moderateReview applies an action; a hide needs a reason, shown to the author.
func (h *Handler) moderateReview(w http.ResponseWriter, r *http.Request, action string) {
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
	rv, err := h.Store.GetReview(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if rv.UserID == me {
		respondError(w, http.StatusForbidden, "own_content", "nobody moderates their own content")
		return
	}
	res, err := h.Store.ModerateReview(r.Context(), id, me, action, in.Reason)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			respondError(w, http.StatusConflict, "moderation_state", "this action does not apply to the content any more")
			return
		}
		handleStoreError(w, r, err)
		return
	}
	switch action {
	case model.ModerationHide:
		h.notifyModeration(r, res, model.NotifyReviewHidden, in.Reason)
	case model.ModerationRestore:
		h.notifyModeration(r, res, model.NotifyReviewRestored, in.Reason)
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
