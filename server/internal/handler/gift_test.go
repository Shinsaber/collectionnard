package handler_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

const wishSchema = `{"type":"object","x-reservable":true,"properties":{"priority":{"type":"integer"}}}`

// giftMock: UserID's wish list, visible to members; AdminID is UserID's
// friend, the moderator is not. One lot, reserved by the admin and by Camille.
func giftMock(t *testing.T) (*sharingFixture, uuid.UUID, *testutil.MockStore, *bool) {
	f, mock := newSharing("members")
	f.col.CollectionType.AttributesSchema = json.RawMessage(wishSchema)
	lotID := uuid.New()
	read := false
	mock.FriendIDsFn = friendsOf
	mock.ListCollectionItemsFn = func(context.Context, uuid.UUID, model.Pagination, store.CollectionItemFilters) ([]model.CollectionItem, int, error) {
		return []model.CollectionItem{{ID: lotID, CollectionID: f.col.ID, Quantity: 3, Attributes: json.RawMessage(`{}`)}}, 1, nil
	}
	mock.LotReservationsFn = func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID][]model.GiftReservation, error) {
		read = true
		return map[uuid.UUID][]model.GiftReservation{lotID: {
			{ID: uuid.New(), GiverID: testutil.AdminID, Giver: &model.ReviewAuthor{Username: "admin"}, Quantity: 1, Note: "ma note"},
			{ID: uuid.New(), GiverID: uuid.New(), Giver: &model.ReviewAuthor{Username: "camille"}, Quantity: 1, Note: "secret de camille",
				Owner: &model.ReviewAuthor{Username: "user"}, Item: &model.Item{}},
		}}, nil
	}
	mock.GetCollectionLotFn = func(_ context.Context, c, l uuid.UUID) (*model.CollectionItem, error) {
		if c != f.col.ID || l != lotID {
			return nil, store.ErrNotFound
		}
		return &model.CollectionItem{ID: lotID, Quantity: 3}, nil
	}
	return f, lotID, mock, &read
}

// The heart of P7-4: the owner never gets a reservation, by no route; nor
// does a share link or a reader who is not a friend.
func TestReservations_NeverShownToTheOwner(t *testing.T) {
	f, _, mock, read := giftMock(t)
	f.shareHash = hashOf("tok")
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	items := "/api/v1/public/collections/" + f.col.ID.String() + "/items"
	for name, c := range map[string]struct{ path, token string }{
		"owner":      {items, testutil.UserToken()},
		"not friend": {items, testutil.ModeratorToken()},
		"anonymous":  {items, ""},
		"share link": {"/api/v1/share/items?token=tok", testutil.AdminToken()},
	} {
		*read = false
		status, _, raw := callJSON(t, srv, http.MethodGet, c.path, c.token, nil)
		if status != http.StatusOK && name != "anonymous" {
			t.Fatalf("%s: %d %s", name, status, raw)
		}
		if *read || strings.Contains(string(raw), "reservations") || strings.Contains(string(raw), "camille") {
			t.Errorf("%s sees reservations: %s", name, raw)
		}
	}
	for name, token := range map[string]string{"owner": testutil.UserToken(), "not friend": testutil.ModeratorToken()} {
		_, _, raw := callJSON(t, srv, http.MethodGet, "/api/v1/public/collections/"+f.col.ID.String(), token, nil)
		if strings.Contains(string(raw), "can_reserve") {
			t.Errorf("%s may reserve: %s", name, raw)
		}
	}
}

func TestReservations_SeenByFriends(t *testing.T) {
	f, _, mock, _ := giftMock(t)
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	_, _, raw := callJSON(t, srv, http.MethodGet, "/api/v1/public/collections/"+f.col.ID.String(), testutil.AdminToken(), nil)
	if !strings.Contains(string(raw), `"can_reserve":true`) {
		t.Errorf("a friend may reserve: %s", raw)
	}
	status, _, raw := callJSON(t, srv, http.MethodGet, "/api/v1/public/collections/"+f.col.ID.String()+"/items", testutil.AdminToken(), nil)
	body := string(raw)
	if status != http.StatusOK || !strings.Contains(body, `"username":"camille"`) || !strings.Contains(body, "ma note") ||
		strings.Contains(body, "secret de camille") || !strings.Contains(body, `"mine":true`) || strings.Contains(body, `"owner"`) {
		t.Errorf("friend: %d %s", status, body)
	}
}

func TestCreateReservation(t *testing.T) {
	f, lotID, mock, _ := giftMock(t)
	var giver uuid.UUID
	mock.CreateReservationFn = func(_ context.Context, lot, g uuid.UUID, q int, _, note string) (*model.GiftReservation, error) {
		giver = g
		return &model.GiftReservation{ID: uuid.New(), GiverID: g, Quantity: q, Note: note, Owner: &model.ReviewAuthor{Username: "user"}}, nil
	}
	path := "/api/v1/collections/" + f.col.ID.String() + "/lots/" + lotID.String() + "/reservations"
	if resp, body := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{"quantity":2,"note":" la bleue "}`)); resp.StatusCode != http.StatusCreated ||
		giver != testutil.AdminID || !strings.Contains(body, `"note":"la bleue"`) || strings.Contains(body, `"owner"`) {
		t.Fatalf("friend: %d %s", resp.StatusCode, body)
	}
	for name, token := range map[string]string{"owner": testutil.UserToken(), "not friend": testutil.ModeratorToken()} {
		if resp, _ := call(t, mock, "POST", path, token, strings.NewReader(`{}`)); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: %d", name, resp.StatusCode)
		}
	}
	if resp, body := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{"quantity":0}`)); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "invalid_quantity") {
		t.Errorf("quantity 0: %d %s", resp.StatusCode, body)
	}
	mock.CreateReservationFn = func(context.Context, uuid.UUID, uuid.UUID, int, string, string) (*model.GiftReservation, error) {
		return nil, store.ErrInvalidQuantity
	}
	if resp, body := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{"quantity":3}`)); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "reservation_exceeds_lot") {
		t.Errorf("beyond the lot: %d %s", resp.StatusCode, body)
	}
	f.col.CollectionType.AttributesSchema = json.RawMessage(`{"type":"object"}`)
	if resp, _ := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{}`)); resp.StatusCode != http.StatusNotFound {
		t.Errorf("not a wish list: %d", resp.StatusCode)
	}
}

func TestUpdateAndCancelReservation(t *testing.T) {
	id := uuid.New()
	var got model.ReservationUpdate
	mock := &testutil.MockStore{
		GetReservationFn: func(_ context.Context, rid uuid.UUID) (*model.GiftReservation, error) {
			if rid != id {
				return nil, store.ErrNotFound
			}
			return &model.GiftReservation{ID: id, GiverID: testutil.AdminID, OwnerID: testutil.UserID}, nil
		},
		UpdateReservationFn: func(_ context.Context, _ uuid.UUID, in model.ReservationUpdate) (*model.GiftReservation, error) {
			got = in
			return &model.GiftReservation{ID: id, Status: in.Status}, nil
		},
		DeleteReservationFn: func(context.Context, uuid.UUID) error { return nil },
	}
	path := "/api/v1/reservations/" + id.String()
	if resp, body := call(t, mock, "PUT", path, testutil.AdminToken(), strings.NewReader(`{"quantity":1,"status":"given"}`)); resp.StatusCode != http.StatusOK || got.Status != "given" {
		t.Errorf("given: %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, mock, "PUT", path, testutil.AdminToken(), strings.NewReader(`{"quantity":1,"status":"cancelled"}`)); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("unknown status: %d", resp.StatusCode)
	}
	for _, method := range []string{"PUT", "DELETE"} {
		if resp, _ := call(t, mock, method, path, testutil.UserToken(), strings.NewReader(`{"quantity":1,"status":"given"}`)); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s by the owner of the list: %d", method, resp.StatusCode)
		}
	}
	if resp, _ := call(t, mock, "DELETE", path, testutil.AdminToken(), nil); resp.StatusCode != http.StatusNoContent {
		t.Errorf("cancel: %d", resp.StatusCode)
	}
}

func TestGiftReminders(t *testing.T) {
	giver := uuid.New()
	var to, marked []uuid.UUID
	mock := &testutil.MockStore{
		LeftReservationsFn: func(context.Context) ([]model.GiftReservation, error) {
			return []model.GiftReservation{{ID: uuid.New(), GiverID: giver, Owner: &model.ReviewAuthor{Username: "user"}}}, nil
		},
		CreateNotificationsFn: func(_ context.Context, users []uuid.UUID, kind string, _ any) error {
			if kind == model.NotifyGiftLeftList {
				to = append(to, users...)
			}
			return nil
		},
		MarkReservationsNotifiedFn: func(_ context.Context, ids []uuid.UUID) error {
			marked = ids
			return nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	testutil.NewTestHandler(mock).RunReminders(ctx, time.Hour)
	if len(to) != 1 || to[0] != giver || len(marked) != 1 {
		t.Errorf("notified %v, marked %v", to, marked)
	}
}

// hashOf: the hash the server stores for a share token.
func hashOf(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func TestReservationExpiry(t *testing.T) {
	f, lotID, mock, _ := giftMock(t)
	var expires string
	mock.CreateReservationFn = func(_ context.Context, _, g uuid.UUID, q int, exp, _ string) (*model.GiftReservation, error) {
		expires = exp
		return &model.GiftReservation{ID: uuid.New(), GiverID: g, Quantity: q, ExpiresAt: exp}, nil
	}
	path := "/api/v1/collections/" + f.col.ID.String() + "/lots/" + lotID.String() + "/reservations"
	day := func(days int) string { return time.Now().AddDate(0, 0, days).Format("2006-01-02") }

	if resp, body := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{}`)); resp.StatusCode != http.StatusCreated || expires != day(model.GiftDefaultDays) {
		t.Errorf("default due date: %d %s %s", resp.StatusCode, body, expires)
	}
	if resp, _ := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{"expires_at":"`+day(30)+`"}`)); resp.StatusCode != http.StatusCreated || expires != day(30) {
		t.Errorf("chosen due date: %d %s", resp.StatusCode, expires)
	}
	for _, d := range []string{day(-1), day(model.GiftMaxDays + 1), "bientôt"} {
		if resp, body := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{"expires_at":"`+d+`"}`)); resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("%s: %d %s", d, resp.StatusCode, body)
		}
	}

	// Modification : échéance absente = inchangée, même passée ; changée = dans les bornes.
	id := uuid.New()
	past := day(-3)
	var got model.ReservationUpdate
	mock.GetReservationFn = func(context.Context, uuid.UUID) (*model.GiftReservation, error) {
		return &model.GiftReservation{ID: id, GiverID: testutil.AdminID, ExpiresAt: past}, nil
	}
	mock.UpdateReservationFn = func(_ context.Context, _ uuid.UUID, in model.ReservationUpdate) (*model.GiftReservation, error) {
		got = in
		return &model.GiftReservation{ID: id}, nil
	}
	upd := "/api/v1/reservations/" + id.String()
	if resp, _ := call(t, mock, "PUT", upd, testutil.AdminToken(), strings.NewReader(`{"quantity":1,"status":"given"}`)); resp.StatusCode != http.StatusOK || got.ExpiresAt != past {
		t.Errorf("expired, marked given: %d %+v", resp.StatusCode, got)
	}
	if resp, _ := call(t, mock, "PUT", upd, testutil.AdminToken(), strings.NewReader(`{"quantity":1,"status":"reserved","expires_at":"`+day(10)+`"}`)); resp.StatusCode != http.StatusOK || got.ExpiresAt != day(10) {
		t.Errorf("extended: %d %+v", resp.StatusCode, got)
	}
	if resp, _ := call(t, mock, "PUT", upd, testutil.AdminToken(), strings.NewReader(`{"quantity":1,"status":"reserved","expires_at":"`+day(-1)+`"}`)); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("moved into the past: %d", resp.StatusCode)
	}
}

func TestGiftExpiringReminders(t *testing.T) {
	giver := uuid.New()
	var kinds []string
	var marked []uuid.UUID
	mock := &testutil.MockStore{
		ExpiringReservationsFn: func(context.Context) ([]model.GiftReservation, error) {
			return []model.GiftReservation{{ID: uuid.New(), GiverID: giver, ExpiresAt: "2026-12-24", Owner: &model.ReviewAuthor{Username: "user"}}}, nil
		},
		CreateNotificationsFn: func(_ context.Context, _ []uuid.UUID, kind string, _ any) error {
			kinds = append(kinds, kind)
			return nil
		},
		MarkReservationsExpiryRemindedFn: func(_ context.Context, ids []uuid.UUID) error {
			marked = ids
			return nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	testutil.NewTestHandler(mock).RunReminders(ctx, time.Hour)
	if len(kinds) != 1 || kinds[0] != model.NotifyGiftExpiring || len(marked) != 1 {
		t.Errorf("kinds %v, marked %v", kinds, marked)
	}
}
