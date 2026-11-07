package handler

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/privacy"
	"github.com/shinsaber/colec/internal/review"
	"github.com/shinsaber/colec/internal/validator"
)

// Reviews (I-0, docs/phases/interactions.md). Every note counts in the average
// of the item, whatever its visibility; what the reader sees of a review is
// decided here, never by the client: the author leaves the server only when the
// review is signed for the reader.

// signedFor keeps the author only when the review is signed for the reader, and
// the edition used with it: a rare edition could point at its owner.
func signedFor(reviews []model.Review, viewer uuid.UUID) {
	for i := range reviews {
		if !privacy.CanView(reviews[i].Visibility, reviews[i].UserID, viewer) {
			reviews[i].Author = nil
			reviews[i].ContextItemID, reviews[i].ContextItem = nil, nil
		}
	}
}

// ListItemReviews returns the reviews of an item shown to the reader (optional
// authentication), their own excluded: it comes from GET …/reviews/mine.
func (h *Handler) ListItemReviews(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	viewer := middleware.GetUserID(r.Context())
	p := model.ReviewListParams{Viewer: viewer, Levels: privacy.Levels(viewer),
		Sort: r.URL.Query().Get("sort"), Lang: r.URL.Query().Get("lang")}
	if p.Sort != "" && p.Sort != "recent" && p.Sort != "rating" && p.Sort != "interesting" {
		respondError(w, http.StatusBadRequest, "bad_request", "sort must be recent, rating or interesting")
		return
	}
	if p.Lang != "" && !review.ValidLang(p.Lang) {
		respondError(w, http.StatusBadRequest, "bad_request", "unknown lang")
		return
	}
	if p.ContextItemID, ok = parseOptionalUUID(w, r, "context_item_id"); !ok {
		return
	}
	if _, err := h.Store.GetItemReviewType(r.Context(), id); err != nil {
		handleStoreError(w, r, err)
		return
	}
	pg := parsePagination(r)
	list, total, err := h.Store.ListItemReviews(r.Context(), id, p, pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	signedFor(list, viewer)
	if err := h.withReactions(r, list); err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

// GetItemReviewSummary returns the averages of an item (optional
// authentication), hidden below review.MinRatings notes.
func (h *Handler) GetItemReviewSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	rt, err := h.Store.GetItemReviewType(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	viewer := middleware.GetUserID(r.Context())
	st, err := h.Store.GetReviewStats(r.Context(), id, viewer, privacy.Levels(viewer))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	out := model.ReviewSummary{Enabled: rt != nil, ReviewType: rt, Count: st.Count, MinRatings: review.MinRatings,
		NoteOnlyCount: st.NoteOnly, Rating: averageAbove(st.Avg, st.Count), Criteria: []model.ReviewCriterionSummary{},
		ContextItems: []model.Item{}}
	if rt != nil && len(rt.ContextRelationTypeIDs) > 0 {
		if out.ContextItems, err = h.Store.ListReviewContextItems(r.Context(), rt.ID, id); err != nil {
			handleStoreError(w, r, err)
			return
		}
	}
	if rt != nil {
		for _, key := range privacy.Properties(rt.AttributesSchema) {
			c := st.Criteria[key]
			out.Criteria = append(out.Criteria, model.ReviewCriterionSummary{Key: key, Count: c.Count, Rating: averageAbove(c.Avg, c.Count)})
		}
	}
	respondJSON(w, http.StatusOK, out)
}

// averageAbove rounds an average, or hides it below the threshold.
func averageAbove(avg float64, count int) *float64 {
	if count < review.MinRatings {
		return nil
	}
	v := review.Round(avg)
	return &v
}

func (h *Handler) GetMyReview(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	rv, err := h.Store.GetMyReview(r.Context(), id, middleware.GetUserID(r.Context()))
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	one := []model.Review{*rv}
	if err := h.withReactions(r, one); err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, one[0])
}

// PutMyReview creates or replaces the review of the reader on an item.
func (h *Handler) PutMyReview(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var in model.ReviewPut
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	rt, err := h.Store.GetItemReviewType(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if rt == nil {
		respondError(w, http.StatusConflict, "reviews_closed", "reviews are closed on this item type")
		return
	}
	if h.limited(w, r, "reviews", rateReviews) {
		return
	}
	write, field, err := checkReview(rt, in)
	if err != nil {
		respondValidationError(w, err.Error(), map[string]any{"fields": map[string]string{field: err.Error()}})
		return
	}
	if in.ContextItemID != nil {
		linked, err := h.Store.IsReviewContext(r.Context(), rt.ID, id, *in.ContextItemID)
		if err != nil {
			handleStoreError(w, r, err)
			return
		}
		if !linked {
			msg := "context_item_id must be an edition linked to this item"
			respondValidationError(w, msg, map[string]any{"fields": map[string]string{"context_item_id": msg}})
			return
		}
		write.ContextItemID = in.ContextItemID
	}
	rv, err := h.Store.PutReview(r.Context(), id, middleware.GetUserID(r.Context()), *write)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, rv)
}

type reviewError string

func (e reviewError) Error() string { return string(e) }

// checkReview turns a request into what is stored; on error, field names the culprit.
func checkReview(rt *model.ReviewType, in model.ReviewPut) (write *model.ReviewWrite, field string, err error) {
	hasCriteria := len(in.Criteria) > 0 && strings.TrimSpace(string(in.Criteria)) != "null"
	write = &model.ReviewWrite{Body: strings.TrimSpace(in.Body), Spoiler: in.Spoiler, Lang: in.Lang, Visibility: in.Visibility}
	switch {
	case in.Rating != nil && hasCriteria:
		return nil, "rating", reviewError("give an overall rating or the criteria, not both")
	case in.Rating != nil:
		if *in.Rating < 0 || *in.Rating > review.MaxRating {
			return nil, "rating", reviewError("rating must be an integer from 0 to 10")
		}
		write.Rating = float64(*in.Rating)
	case hasCriteria:
		if err := validator.ValidateAttributes(rt.AttributesSchema, in.Criteria); err != nil {
			return nil, "criteria", err
		}
		if write.Rating, err = review.Rate(rt.AttributesSchema, in.Criteria); err != nil {
			return nil, "criteria", err
		}
		write.ReviewTypeID, write.Criteria = &rt.ID, in.Criteria
	default:
		return nil, "rating", reviewError("a review always carries a rating")
	}
	if err := review.CheckBody(write.Body); err != nil {
		return nil, "body", err
	}
	if !review.ValidLang(in.Lang) {
		return nil, "lang", reviewError("lang must be a language of the interface")
	}
	if !review.ValidVisibility(in.Visibility) {
		return nil, "visibility", reviewError("visibility must be private, members, anonymous or public")
	}
	return write, "", nil
}

func (h *Handler) DeleteMyReview(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Store.DeleteReview(r.Context(), id, middleware.GetUserID(r.Context())); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListMyReviews returns every review of the reader, with its item.
func (h *Handler) ListMyReviews(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	list, total, err := h.Store.ListUserReviews(r.Context(), middleware.GetUserID(r.Context()), nil, pg)
	if err == nil {
		err = h.withReactions(r, list)
	}
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

// ListPublicReviews returns the reviews a member signed for the reader: an
// anonymous or note-only review listed under a name would lose its anonymity.
func (h *Handler) ListPublicReviews(w http.ResponseWriter, r *http.Request) {
	u, ok := h.visibleProfile(w, r)
	if !ok {
		return
	}
	viewer := middleware.GetUserID(r.Context())
	pg := parsePagination(r)
	blocked, err := h.Store.IsBlocked(r.Context(), viewer, u.ID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if blocked { // ses contenus me sont masqués
		respondJSON(w, http.StatusOK, model.NewPaginatedResponse([]model.Review{}, 0, pg))
		return
	}
	list, total, err := h.Store.ListUserReviews(r.Context(), u.ID, privacy.Levels(viewer), pg)
	if err == nil {
		err = h.withReactions(r, list)
	}
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}
