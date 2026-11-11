package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

// Loans (P7-3, docs/phases/exemplaires.md). The owner of a lot of a
// collection whose type is marked `x-lendable` lends copies to a friend
// (notified, nothing to accept) or to a mere name. Only the owner writes; the
// loan is shown to the owner and to the borrowing friend, never to the other
// readers of the collection. A reminder goes to both once a loan is overdue.

// RemindersInterval: how often RunReminders looks for overdue loans; each
// due date is reminded once (loans.reminded_at).
const RemindersInterval = time.Hour

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// checkLoanDates validates the dates of a loan; lentAt "" is today.
func checkLoanDates(w http.ResponseWriter, lentAt string, dueAt, returnedAt *string) bool {
	fields := map[string]string{}
	if lentAt != "" && !validDate(lentAt) {
		fields["lent_at"] = "YYYY-MM-DD"
	}
	for name, d := range map[string]*string{"due_at": dueAt, "returned_at": returnedAt} {
		if d == nil {
			continue
		}
		switch {
		case !validDate(*d):
			fields[name] = "YYYY-MM-DD"
		case lentAt != "" && *d < lentAt:
			fields[name] = "before lent_at"
		}
	}
	if len(fields) > 0 {
		respondValidationError(w, "invalid loan dates", map[string]any{"fields": fields})
		return false
	}
	return true
}

func checkLoanQuantity(w http.ResponseWriter, q int) bool {
	if q < 1 || q > model.MaxQuantity {
		respondError(w, http.StatusUnprocessableEntity, "invalid_quantity", "quantity: 1 to 100000")
		return false
	}
	return true
}

func checkLoanNote(w http.ResponseWriter, note *string) bool {
	*note = strings.TrimSpace(*note)
	if utf8.RuneCountInString(*note) > model.MaxLoanNote {
		respondValidationError(w, "note: at most 500 characters", map[string]any{"fields": map[string]string{"note": "too long"}})
		return false
	}
	return true
}

func respondLoanError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrInvalidQuantity) {
		respondError(w, http.StatusUnprocessableEntity, "loan_exceeds_lot", "not enough copies left to lend in this lot")
		return
	}
	handleStoreError(w, r, err)
}

// forOwner / forBorrower keep what each side may see of a loan.
func forOwner(l *model.Loan) { l.Owner = nil }

func forBorrower(l *model.Loan) { l.CollectionName, l.Note = "", "" }

// withActiveLoans attaches their loans in progress to the lots of the owner's list.
func (h *Handler) withActiveLoans(r *http.Request, lots []model.CollectionItem) error {
	ids := make([]uuid.UUID, len(lots))
	for i := range lots {
		ids[i] = lots[i].ID
	}
	loans, err := h.Store.ActiveLoans(r.Context(), ids)
	if err != nil {
		return err
	}
	for i := range lots {
		lots[i].Loans = loans[lots[i].ID]
		for j := range lots[i].Loans {
			forOwner(&lots[i].Loans[j])
			lots[i].Loans[j].Item = nil
		}
	}
	return nil
}

// ownLot loads a lot of a collection of the reader (404 otherwise).
func (h *Handler) ownLot(w http.ResponseWriter, r *http.Request) (*model.Collection, *model.CollectionItem, bool) {
	col, ok := h.ownCollection(w, r, "collectionId")
	if !ok {
		return nil, nil, false
	}
	lotID, ok := parseUUID(w, r, "lotId")
	if !ok {
		return nil, nil, false
	}
	lot, err := h.Store.GetCollectionLot(r.Context(), col.ID, lotID)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, nil, false
	}
	return col, lot, true
}

// ListLotLoans answers GET …/lots/{lotId}/loans: in progress, then the history.
func (h *Handler) ListLotLoans(w http.ResponseWriter, r *http.Request) {
	_, lot, ok := h.ownLot(w, r)
	if !ok {
		return
	}
	list, err := h.Store.ListLotLoans(r.Context(), lot.ID)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	for i := range list {
		forOwner(&list[i])
		list[i].Item = nil
	}
	respondJSON(w, http.StatusOK, list)
}

// CreateLoan answers POST …/lots/{lotId}/loans.
func (h *Handler) CreateLoan(w http.ResponseWriter, r *http.Request) {
	col, lot, ok := h.ownLot(w, r)
	if !ok {
		return
	}
	var in model.LoanCreate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if col.CollectionType == nil || !model.IsLendable(col.CollectionType.AttributesSchema) {
		respondError(w, http.StatusUnprocessableEntity, "not_lendable", "the type of this collection does not open loans")
		return
	}
	q := 1
	if in.Quantity != nil {
		q = *in.Quantity
	}
	in.BorrowerUsername, in.BorrowerName = strings.TrimSpace(in.BorrowerUsername), strings.TrimSpace(in.BorrowerName)
	if !checkLoanQuantity(w, q) || !checkLoanDates(w, in.LentAt, in.DueAt, nil) || !checkLoanNote(w, &in.Note) {
		return
	}
	if in.DueAt != nil && in.LentAt == "" && *in.DueAt < time.Now().Format("2006-01-02") {
		respondValidationError(w, "due_at before lent_at", map[string]any{"fields": map[string]string{"due_at": "before lent_at"}})
		return
	}
	loan := model.LoanInput{Quantity: q, LentAt: in.LentAt, DueAt: in.DueAt, Note: in.Note}
	switch {
	case (in.BorrowerUsername == "") == (in.BorrowerName == ""):
		respondValidationError(w, "borrower_username or borrower_name, not both",
			map[string]any{"fields": map[string]string{"borrower": "one of borrower_username, borrower_name"}})
		return
	case in.BorrowerName != "":
		if utf8.RuneCountInString(in.BorrowerName) > model.MaxLoanBorrowerName {
			respondValidationError(w, "borrower_name: at most 100 characters",
				map[string]any{"fields": map[string]string{"borrower_name": "too long"}})
			return
		}
		loan.BorrowerName = in.BorrowerName
	default:
		friend, ok := h.loanFriend(w, r, in.BorrowerUsername)
		if !ok {
			return
		}
		loan.BorrowerID = &friend.ID
		loan.BorrowerName = friend.Username
		if friend.DisplayName != nil && *friend.DisplayName != "" {
			loan.BorrowerName = *friend.DisplayName
		}
	}
	created, err := h.Store.CreateLoan(r.Context(), lot.ID, loan)
	if err != nil {
		respondLoanError(w, r, err)
		return
	}
	if created.BorrowerID != nil {
		payload := map[string]any{"loan_id": created.ID, "item_id": created.Item.ID}
		if created.Owner != nil {
			payload["by"] = created.Owner.Username
		}
		if err := h.Store.CreateNotifications(r.Context(), []uuid.UUID{*created.BorrowerID}, model.NotifyLoanStarted, payload); err != nil {
			slog.WarnContext(r.Context(), "notification not recorded", "kind", model.NotifyLoanStarted, "err", err)
		}
	}
	forOwner(created)
	created.Item = nil
	respondJSON(w, http.StatusCreated, created)
}

// loanFriend resolves a borrowing member: a friend of the reader, else 422
// borrower_not_friend (whether the account exists or not: nothing to probe).
func (h *Handler) loanFriend(w http.ResponseWriter, r *http.Request, username string) (*model.User, bool) {
	viewer, err := h.viewer(r)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	u, _, err := h.Store.GetUserByUsername(r.Context(), username)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		handleStoreError(w, r, err)
		return nil, false
	}
	if u == nil || u.DisabledAt != nil || !viewer.FriendOf(u.ID) {
		respondError(w, http.StatusUnprocessableEntity, "borrower_not_friend", "a borrowing member must be a friend; otherwise give a name")
		return nil, false
	}
	return u, true
}

// ownLoan loads a loan of a lot of the reader (404 otherwise, the borrower included).
func (h *Handler) ownLoan(w http.ResponseWriter, r *http.Request) (*model.Loan, bool) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return nil, false
	}
	l, err := h.Store.GetLoan(r.Context(), id)
	if err != nil {
		handleStoreError(w, r, err)
		return nil, false
	}
	if l.OwnerID != middleware.GetUserID(r.Context()) {
		notFound(w)
		return nil, false
	}
	return l, true
}

// UpdateLoan answers PUT /loans/{id}: due date, return, quantity, note.
func (h *Handler) UpdateLoan(w http.ResponseWriter, r *http.Request) {
	l, ok := h.ownLoan(w, r)
	if !ok {
		return
	}
	var in model.LoanUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		respondDecodeError(w, err)
		return
	}
	if in.LentAt == "" {
		in.LentAt = l.LentAt
	}
	if !checkLoanQuantity(w, in.Quantity) || !checkLoanDates(w, in.LentAt, in.DueAt, in.ReturnedAt) || !checkLoanNote(w, &in.Note) {
		return
	}
	updated, err := h.Store.UpdateLoan(r.Context(), l.ID, model.LoanInput{Quantity: in.Quantity, LentAt: in.LentAt,
		DueAt: in.DueAt, ReturnedAt: in.ReturnedAt, Note: in.Note})
	if err != nil {
		respondLoanError(w, r, err)
		return
	}
	forOwner(updated)
	respondJSON(w, http.StatusOK, updated)
}

// DeleteLoan answers DELETE /loans/{id}: a loan entered by mistake.
func (h *Handler) DeleteLoan(w http.ResponseWriter, r *http.Request) {
	l, ok := h.ownLoan(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteLoan(r.Context(), l.ID); err != nil {
		handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseLoanStatus(w http.ResponseWriter, r *http.Request) (string, bool) {
	s := r.URL.Query().Get("status")
	if s != "" && s != "active" && s != "returned" {
		respondError(w, http.StatusBadRequest, "bad_request", "status must be active or returned")
		return "", false
	}
	return s, true
}

// ListMyLoans answers GET /me/loans: what I lent.
func (h *Handler) ListMyLoans(w http.ResponseWriter, r *http.Request) {
	status, ok := parseLoanStatus(w, r)
	if !ok {
		return
	}
	pg := parsePagination(r)
	list, total, err := h.Store.ListLoans(r.Context(), middleware.GetUserID(r.Context()), status, pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	for i := range list {
		forOwner(&list[i])
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

// ListMyBorrowings answers GET /me/borrowings: what friends lent me.
func (h *Handler) ListMyBorrowings(w http.ResponseWriter, r *http.Request) {
	status, ok := parseLoanStatus(w, r)
	if !ok {
		return
	}
	pg := parsePagination(r)
	list, total, err := h.Store.ListBorrowings(r.Context(), middleware.GetUserID(r.Context()), status, pg)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	for i := range list {
		forBorrower(&list[i])
	}
	respondJSON(w, http.StatusOK, model.NewPaginatedResponse(list, total, pg))
}

// RunReminders notifies, every interval, the lender and the borrowing friend
// of each loan past its due date, once per due date, and the giver of each
// reservation whose lot left the wish list (P7-4).
func (h *Handler) RunReminders(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if n, err := h.sendReminders(ctx); err != nil {
			if ctx.Err() == nil {
				slog.Error("loan reminders", "err", err)
			}
		} else if n > 0 {
			slog.Info("loan reminders", "loans", n)
		}
		if n, err := h.remindGifts(ctx); err != nil {
			if ctx.Err() == nil {
				slog.Error("gift reminders", "err", err)
			}
		} else if n > 0 {
			slog.Info("gift reminders", "reservations", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (h *Handler) sendReminders(ctx context.Context) (int, error) {
	loans, err := h.Store.OverdueLoans(ctx)
	if err != nil || len(loans) == 0 {
		return 0, err
	}
	// Les rappels envoyés sont marqués même si un suivant échoue : jamais deux fois le même.
	ids := make([]uuid.UUID, 0, len(loans))
	for i := range loans {
		if err = h.remindLoan(ctx, &loans[i]); err != nil {
			break
		}
		ids = append(ids, loans[i].ID)
	}
	if len(ids) > 0 {
		if mErr := h.Store.MarkLoansReminded(ctx, ids); mErr != nil {
			return 0, mErr
		}
	}
	return len(ids), err
}

func (h *Handler) remindLoan(ctx context.Context, l *model.Loan) error {
	payload := map[string]any{"loan_id": l.ID, "item_id": l.Item.ID, "borrower": l.BorrowerName}
	if err := h.Store.CreateNotifications(ctx, []uuid.UUID{l.OwnerID}, model.NotifyLoanOverdue, payload); err != nil {
		return err
	}
	if l.BorrowerID == nil || l.Borrower == nil {
		return nil
	}
	payload = map[string]any{"loan_id": l.ID, "item_id": l.Item.ID, "by": l.Owner.Username, "borrowed": true}
	return h.Store.CreateNotifications(ctx, []uuid.UUID{*l.BorrowerID}, model.NotifyLoanOverdue, payload)
}
