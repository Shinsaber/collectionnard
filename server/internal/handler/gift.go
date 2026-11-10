package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

// Gift reservations (P7-4, docs/phases/exemplaires-prets.md). A friend of the
// owner of a wish list (type marked `x-reservable`) reserves a share of a lot
// to give it; the other friends see who reserved, to agree. The owner never
// learns anything: their routes, lists, exports and statistics never read the
// reservations, a reservation of their own list answers 404, and a share link
// or an anonymous read never shows them.

const rateReservations = 60 // par heure

// shapeReservation keeps what a friend of the owner may see: the giver's
// name, and the note of their own reservations only.
func shapeReservation(g *model.GiftReservation, reader uuid.UUID) {
	g.Mine = g.GiverID == reader
	if !g.Mine {
		g.Note = ""
	}
	g.Owner, g.Item = nil, nil
}

// withReservations attaches their reservations to the lots of a wish list,
// for a friend of the owner (readable.canReserve).
func (h *Handler) withReservations(r *http.Request, lots []model.CollectionItem, reader uuid.UUID) error {
	ids := make([]uuid.UUID, len(lots))
	for i := range lots {
		ids[i] = lots[i].ID
	}
	byLot, err := h.Store.LotReservations(r.Context(), ids)
	if err != nil {
		return err
	}
	for i := range lots {
		lots[i].Reservations = byLot[lots[i].ID]
		for j := range lots[i].Reservations {
			shapeReservation(&lots[i].Reservations[j], reader)
		}
	}
	return nil
}

func checkReservation(w http.ResponseWriter, quantity int, note *string) bool {
	*note = strings.TrimSpace(*note)
	switch {
	case quantity < 1 || quantity > model.MaxQuantity:
		respondError(w, http.StatusUnprocessableEntity, "invalid_quantity", "quantity: 1 to 100000")
	case utf8.RuneCountInString(*note) > model.MaxLoanNote:
		respondValidationError(w, "note: at most 500 characters", map[string]any{"fields": map[string]string{"note": "too long"}})
	default:
		return true
	}
	return false
}

// checkExpiry validates a due date of reservation: from today to GiftMaxDays ahead.
func checkExpiry(w http.ResponseWriter, day string) bool {
	now := time.Now()
	from, to := now.Format("2006-01-02"), now.AddDate(0, 0, model.GiftMaxDays).Format("2006-01-02")
	if !validDate(day) || day < from || day > to {
		respondValidationError(w, "expires_at: from today to one year ahead",
			map[string]any{"fields": map[string]string{"expires_at": "today to one year"}})
		return false
	}
	return true
}

func respondReservationError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrInvalidQuantity) {
		respondError(w, http.StatusUnprocessableEntity, "reservation_exceeds_lot", "not enough copies left to reserve in this lot")
		return
	}
	handleStoreError(w, r, err)
}

// CreateReservation answers POST /collections/{collectionId}/lots/{lotId}/reservations:
// a friend of the owner reserves copies of a lot of a wish list they can read.
// The owner, or anyone who may not reserve, gets 404: nothing is disclosed.
func (h *Handler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	colID, ok := parseUUID(w, r, "collectionId")
	if !ok {
		return
	}
	lotID, ok := parseUUID(w, r, "lotId")
	if !ok {
		return
	}
	rd, ok := h.loadReadable(w, r, colID, false)
	if !ok {
		return
	}
	if !rd.canReserve {
		notFound(w)
		return
	}
	if _, err := h.Store.GetCollectionLot(r.Context(), colID, lotID); err != nil {
		handleStoreError(w, r, err)
		return
	}
	var in model.ReservationCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	q := 1
	if in.Quantity != nil {
		q = *in.Quantity
	}
	if in.ExpiresAt == "" {
		in.ExpiresAt = time.Now().AddDate(0, 0, model.GiftDefaultDays).Format("2006-01-02")
	}
	if !checkReservation(w, q, &in.Note) || !checkExpiry(w, in.ExpiresAt) ||
		h.limited(w, r, "reservations", rateReservations) {
		return
	}
	g, err := h.Store.CreateReservation(r.Context(), lotID, rd.reader, q, in.ExpiresAt, in.Note)
	if err != nil {
		respondReservationError(w, r, err)
		return
	}
	shapeReservation(g, rd.reader)
	respondJSON(w, http.StatusCreated, g)
}

// ownReservation loads a reservation of the reader (404 otherwise, the owner of the list included).
func (h *Handler) ownReservation(w http.ResponseWriter, r *http.Request) (*model.GiftReservation, bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return nil, false
	}
	g, err := h.Store.GetReservation(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	if g.GiverID != middleware.GetUserID(r.Context()) {
		notFound(w)
		return nil, false
	}
	return g, true
}

// UpdateReservation answers PUT /reservations/{id}: quantity, status (given), due date, note.
func (h *Handler) UpdateReservation(w http.ResponseWriter, r *http.Request) {
	g, ok := h.ownReservation(w, r)
	if !ok {
		return
	}
	var in model.ReservationUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.Status != model.GiftReserved && in.Status != model.GiftGiven {
		respondValidationError(w, "status: reserved or given", map[string]any{"fields": map[string]string{"status": "reserved or given"}})
		return
	}
	if !checkReservation(w, in.Quantity, &in.Note) {
		return
	}
	// Échéance gardée telle quelle (même passée), ou changée dans les bornes : prolonger, raccourcir.
	if in.ExpiresAt == "" {
		in.ExpiresAt = g.ExpiresAt
	} else if in.ExpiresAt != g.ExpiresAt && !checkExpiry(w, in.ExpiresAt) {
		return
	}
	updated, err := h.Store.UpdateReservation(r.Context(), g.ID, in)
	if err != nil {
		respondReservationError(w, r, err)
		return
	}
	updated.Mine = true
	respondJSON(w, http.StatusOK, updated)
}

// DeleteReservation answers DELETE /reservations/{id}: the giver cancels.
func (h *Handler) DeleteReservation(w http.ResponseWriter, r *http.Request) {
	g, ok := h.ownReservation(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteReservation(r.Context(), g.ID); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListMyGifts answers GET /me/gifts: what I plan to give, to whom.
func (h *Handler) ListMyGifts(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	list, total, err := h.Store.ListGifts(r.Context(), middleware.GetUserID(r.Context()), pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	for i := range list {
		list[i].Mine, list[i].Giver = true, nil
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

// remindGifts tells each giver that a lot they reserved left the list, or
// that a reservation expires soon.
func (h *Handler) remindGifts(ctx context.Context) (int, error) {
	n, err := h.remindLeftGifts(ctx)
	if err != nil {
		return n, err
	}
	m, err := h.remindExpiringGifts(ctx)
	return n + m, err
}

func (h *Handler) remindExpiringGifts(ctx context.Context) (int, error) {
	soon, err := h.Store.ExpiringReservations(ctx)
	if err != nil || len(soon) == 0 {
		return 0, err
	}
	ids := make([]uuid.UUID, 0, len(soon))
	for _, g := range soon {
		payload := map[string]any{"reservation_id": g.ID, "item_id": g.ItemID, "by": g.Owner.Username, "expires_at": g.ExpiresAt}
		if err = h.Store.CreateNotifications(ctx, []uuid.UUID{g.GiverID}, model.NotifyGiftExpiring, payload); err != nil {
			break
		}
		ids = append(ids, g.ID)
	}
	if len(ids) > 0 {
		if mErr := h.Store.MarkReservationsExpiryReminded(ctx, ids); mErr != nil {
			return 0, mErr
		}
	}
	return len(ids), err
}

func (h *Handler) remindLeftGifts(ctx context.Context) (int, error) {
	left, err := h.Store.LeftReservations(ctx)
	if err != nil || len(left) == 0 {
		return 0, err
	}
	ids := make([]uuid.UUID, 0, len(left))
	for _, g := range left {
		payload := map[string]any{"reservation_id": g.ID, "item_id": g.ItemID, "by": g.Owner.Username}
		if err = h.Store.CreateNotifications(ctx, []uuid.UUID{g.GiverID}, model.NotifyGiftLeftList, payload); err != nil {
			break
		}
		ids = append(ids, g.ID)
	}
	if len(ids) > 0 {
		if mErr := h.Store.MarkReservationsNotified(ctx, ids); mErr != nil {
			return 0, mErr
		}
	}
	return len(ids), err
}
