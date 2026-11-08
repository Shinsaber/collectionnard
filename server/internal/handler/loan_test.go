package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

const lendableSchema = `{"type":"object","x-lendable":true,"properties":{"condition":{"type":"string"}}}`

// loanMock: a lot of the user's lendable collection; AdminID is the user's friend.
func loanMock() (*model.Collection, uuid.UUID, *testutil.MockStore, *[]model.LoanInput, *[]string) {
	col, mock := lotMock()
	col.CollectionType.AttributesSchema = json.RawMessage(lendableSchema)
	lotID, itemID := uuid.New(), uuid.New()
	var created []model.LoanInput
	var kinds []string
	mock.FriendIDsFn = friendsOf
	mock.GetCollectionLotFn = func(_ context.Context, c, l uuid.UUID) (*model.CollectionItem, error) {
		if c != col.ID || l != lotID {
			return nil, store.ErrNotFound
		}
		return &model.CollectionItem{ID: lotID, CollectionID: col.ID, ItemID: itemID, Quantity: 3}, nil
	}
	mock.GetUserByUsernameFn = func(_ context.Context, name string) (*model.User, string, error) {
		switch name {
		case "admin":
			return testutil.AdminUser(), "", nil
		case "stranger":
			return &model.User{ID: uuid.New(), Username: "stranger"}, "", nil
		}
		return nil, "", store.ErrNotFound
	}
	mock.CreateLoanFn = func(_ context.Context, lot uuid.UUID, in model.LoanInput) (*model.Loan, error) {
		created = append(created, in)
		return &model.Loan{ID: uuid.New(), LotID: lot, Quantity: in.Quantity, BorrowerID: in.BorrowerID,
			BorrowerName: in.BorrowerName, OwnerID: testutil.UserID, Owner: &model.ReviewAuthor{Username: "user"},
			Item: &model.Item{ID: itemID}}, nil
	}
	mock.CreateNotificationsFn = func(_ context.Context, _ []uuid.UUID, kind string, _ any) error {
		kinds = append(kinds, kind)
		return nil
	}
	return col, lotID, mock, &created, &kinds
}

func TestCreateLoan(t *testing.T) {
	col, lotID, mock, created, kinds := loanMock()
	path := "/api/v1/collections/" + col.ID.String() + "/lots/" + lotID.String() + "/loans"
	user := testutil.UserToken()

	resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"borrower_username":"admin","quantity":2,"due_at":"2999-01-01"}`))
	if resp.StatusCode != http.StatusCreated || len(*created) != 1 || (*created)[0].BorrowerID == nil ||
		*(*created)[0].BorrowerID != testutil.AdminID || len(*kinds) != 1 || (*kinds)[0] != model.NotifyLoanStarted {
		t.Fatalf("friend: %d %s %+v %v", resp.StatusCode, body, *created, *kinds)
	}
	if strings.Contains(body, `"owner"`) {
		t.Errorf("the owner is not named to themselves: %s", body)
	}
	if resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"borrower_name":" Tante Odile "}`)); resp.StatusCode != http.StatusCreated ||
		(*created)[1].BorrowerName != "Tante Odile" || (*created)[1].BorrowerID != nil || (*created)[1].Quantity != 1 || len(*kinds) != 1 {
		t.Errorf("mere name, not notified: %d %s", resp.StatusCode, body)
	}
	cases := []struct {
		body, code string
	}{
		{`{"borrower_username":"stranger"}`, "borrower_not_friend"},
		{`{"borrower_username":"nobody"}`, "borrower_not_friend"},
		{`{}`, "validation_error"},
		{`{"borrower_username":"admin","borrower_name":"x"}`, "validation_error"},
		{`{"borrower_name":"x","quantity":0}`, "invalid_quantity"},
		{`{"borrower_name":"x","lent_at":"2026-11-08","due_at":"2026-11-01"}`, "validation_error"},
		{`{"borrower_name":"x","due_at":"demain"}`, "validation_error"},
		{`{"borrower_name":"` + strings.Repeat("x", 101) + `"}`, "validation_error"},
	}
	for _, c := range cases {
		if resp, body := call(t, mock, "POST", path, user, strings.NewReader(c.body)); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, c.code) {
			t.Errorf("%s: %d %s", c.body, resp.StatusCode, body)
		}
	}
	if resp, _ := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{"borrower_name":"x"}`)); resp.StatusCode != http.StatusNotFound {
		t.Errorf("another's lot: %d", resp.StatusCode)
	}

	mock.CreateLoanFn = func(context.Context, uuid.UUID, model.LoanInput) (*model.Loan, error) {
		return nil, store.ErrInvalidQuantity
	}
	if resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"borrower_name":"x","quantity":3}`)); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "loan_exceeds_lot") {
		t.Errorf("beyond the lot: %d %s", resp.StatusCode, body)
	}
	col.CollectionType.AttributesSchema = json.RawMessage(`{"type":"object"}`)
	if resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"borrower_name":"x"}`)); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "not_lendable") {
		t.Errorf("not lendable: %d %s", resp.StatusCode, body)
	}
}

func TestUpdateAndDeleteLoan(t *testing.T) {
	_, _, mock, _, _ := loanMock()
	loanID := uuid.New()
	mock.GetLoanFn = func(_ context.Context, id uuid.UUID) (*model.Loan, error) {
		if id != loanID {
			return nil, store.ErrNotFound
		}
		return &model.Loan{ID: id, OwnerID: testutil.UserID, LentAt: "2026-11-01"}, nil
	}
	var got model.LoanInput
	mock.UpdateLoanFn = func(_ context.Context, id uuid.UUID, in model.LoanInput) (*model.Loan, error) {
		got = in
		return &model.Loan{ID: id, Owner: &model.ReviewAuthor{Username: "user"}}, nil
	}
	mock.DeleteLoanFn = func(context.Context, uuid.UUID) error { return nil }
	path := "/api/v1/loans/" + loanID.String()
	user := testutil.UserToken()

	if resp, body := call(t, mock, "PUT", path, user, strings.NewReader(`{"quantity":1,"returned_at":"2026-11-08","note":" rendu "}`)); resp.StatusCode != http.StatusOK ||
		got.LentAt != "2026-11-01" || got.ReturnedAt == nil || got.Note != "rendu" || strings.Contains(body, `"owner"`) {
		t.Errorf("return: %d %s %+v", resp.StatusCode, body, got)
	}
	if resp, _ := call(t, mock, "PUT", path, user, strings.NewReader(`{"quantity":1,"returned_at":"2026-10-01"}`)); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("returned before lent: %d", resp.StatusCode)
	}
	for _, method := range []string{"PUT", "DELETE"} {
		if resp, _ := call(t, mock, method, path, testutil.AdminToken(), strings.NewReader(`{"quantity":1}`)); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s by the borrower: %d", method, resp.StatusCode)
		}
	}
	if resp, _ := call(t, mock, "DELETE", path, user, nil); resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete: %d", resp.StatusCode)
	}
}

// The borrower sees who lent, never the owner's note nor the collection's name.
func TestMyBorrowings(t *testing.T) {
	mock := &testutil.MockStore{
		ListBorrowingsFn: func(_ context.Context, me uuid.UUID, status string, _ model.Pagination) ([]model.Loan, int, error) {
			if me != testutil.AdminID || status != "active" {
				t.Errorf("borrower %s, status %q", me, status)
			}
			return []model.Loan{{ID: uuid.New(), Note: "il abîme tout", CollectionName: "Mes trésors",
				Owner: &model.ReviewAuthor{Username: "user"}}}, 1, nil
		},
	}
	resp, body := call(t, mock, "GET", "/api/v1/me/borrowings?status=active", testutil.AdminToken(), nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"username":"user"`) ||
		strings.Contains(body, "abîme") || strings.Contains(body, "trésors") {
		t.Errorf("%d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, mock, "GET", "/api/v1/me/borrowings?status=late", testutil.AdminToken(), nil); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad status: %d", resp.StatusCode)
	}
}

func TestReminders(t *testing.T) {
	friend := uuid.New()
	loans := []model.Loan{
		{ID: uuid.New(), OwnerID: testutil.UserID, Owner: &model.ReviewAuthor{Username: "user"}, BorrowerID: &friend,
			Borrower: &model.ReviewAuthor{Username: "camille"}, BorrowerName: "Camille", Item: &model.Item{ID: uuid.New()}},
		{ID: uuid.New(), OwnerID: testutil.UserID, Owner: &model.ReviewAuthor{Username: "user"}, BorrowerName: "Odile",
			Item: &model.Item{ID: uuid.New()}},
	}
	var to []uuid.UUID
	var marked []uuid.UUID
	mock := &testutil.MockStore{
		OverdueLoansFn: func(context.Context) ([]model.Loan, error) { return loans, nil },
		CreateNotificationsFn: func(_ context.Context, users []uuid.UUID, kind string, _ any) error {
			if kind != model.NotifyLoanOverdue {
				t.Errorf("kind %s", kind)
			}
			to = append(to, users...)
			return nil
		},
		MarkLoansRemindedFn: func(_ context.Context, ids []uuid.UUID) error {
			marked = ids
			return nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // un seul passage
	testutil.NewTestHandler(mock).RunReminders(ctx, time.Hour)
	if len(to) != 3 || to[1] != friend || len(marked) != 2 {
		t.Errorf("notified %v, marked %v", to, marked)
	}
}

func TestCollectionItems_ActiveLoansForTheOwner(t *testing.T) {
	col, mock := lotMock()
	lotID := uuid.New()
	mock.ListCollectionItemsFn = func(context.Context, uuid.UUID, model.Pagination, store.CollectionItemFilters) ([]model.CollectionItem, int, error) {
		return []model.CollectionItem{{ID: lotID, Quantity: 3}}, 1, nil
	}
	mock.ActiveLoansFn = func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID][]model.Loan, error) {
		return map[uuid.UUID][]model.Loan{lotID: {{ID: uuid.New(), Quantity: 2, BorrowerName: "Camille",
			Owner: &model.ReviewAuthor{Username: "user"}, Item: &model.Item{ID: uuid.New()}}}}, nil
	}
	resp, body := call(t, mock, "GET", "/api/v1/collections/"+col.ID.String()+"/items", testutil.UserToken(), nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"borrower_name":"Camille"`) || strings.Contains(body, `"owner"`) {
		t.Errorf("%d %s", resp.StatusCode, body)
	}
}
