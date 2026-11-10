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
	status := model.GiftReserved
	var got model.ReservationUpdate
	mock := &testutil.MockStore{
		GetReservationFn: func(_ context.Context, rid uuid.UUID) (*model.GiftReservation, error) {
			if rid != id {
				return nil, store.ErrNotFound
			}
			return &model.GiftReservation{ID: id, GiverID: testutil.AdminID, OwnerID: testutil.UserID, Status: status}, nil
		},
		UpdateReservationFn: func(_ context.Context, _ uuid.UUID, in model.ReservationUpdate) (*model.GiftReservation, error) {
			got = in
			return &model.GiftReservation{ID: id}, nil
		},
		DeleteReservationFn: func(context.Context, uuid.UUID) error { return nil },
	}
	path := "/api/v1/reservations/" + id.String()
	if resp, body := call(t, mock, "PUT", path, testutil.AdminToken(), strings.NewReader(`{"quantity":2,"note":" la bleue "}`)); resp.StatusCode != http.StatusOK || got.Quantity != 2 || got.Note != "la bleue" {
		t.Errorf("edit: %d %s", resp.StatusCode, body)
	}
	// Une fois remise, la réservation ne se modifie plus : le propriétaire le sait.
	status = model.GiftGiven
	if resp, body := call(t, mock, "PUT", path, testutil.AdminToken(), strings.NewReader(`{"quantity":1}`)); resp.StatusCode != http.StatusConflict || !strings.Contains(body, "gift_state") {
		t.Errorf("edit once handed over: %d %s", resp.StatusCode, body)
	}
	for _, method := range []string{"PUT", "DELETE"} {
		if resp, _ := call(t, mock, method, path, testutil.UserToken(), strings.NewReader(`{"quantity":1}`)); resp.StatusCode != http.StatusNotFound {
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
		return &model.GiftReservation{ID: id, GiverID: testutil.AdminID, ExpiresAt: past, Status: model.GiftReserved}, nil
	}
	mock.UpdateReservationFn = func(_ context.Context, _ uuid.UUID, in model.ReservationUpdate) (*model.GiftReservation, error) {
		got = in
		return &model.GiftReservation{ID: id}, nil
	}
	upd := "/api/v1/reservations/" + id.String()
	if resp, _ := call(t, mock, "PUT", upd, testutil.AdminToken(), strings.NewReader(`{"quantity":1,"note":"x"}`)); resp.StatusCode != http.StatusOK || got.ExpiresAt != past {
		t.Errorf("expired, due date kept: %d %+v", resp.StatusCode, got)
	}
	if resp, _ := call(t, mock, "PUT", upd, testutil.AdminToken(), strings.NewReader(`{"quantity":1,"expires_at":"`+day(10)+`"}`)); resp.StatusCode != http.StatusOK || got.ExpiresAt != day(10) {
		t.Errorf("extended: %d %+v", resp.StatusCode, got)
	}
	if resp, _ := call(t, mock, "PUT", upd, testutil.AdminToken(), strings.NewReader(`{"quantity":1,"expires_at":"`+day(-1)+`"}`)); resp.StatusCode != http.StatusUnprocessableEntity {
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

// P7-4b : remise d'un cadeau, puis confirmation ou refus du propriétaire.

func TestHandOverReservation(t *testing.T) {
	id, colID := uuid.New(), uuid.New()
	status := model.GiftReserved
	var notified []uuid.UUID
	var kinds []string
	var payload map[string]any
	mock := &testutil.MockStore{
		GetReservationFn: func(context.Context, uuid.UUID) (*model.GiftReservation, error) {
			return &model.GiftReservation{ID: id, GiverID: testutil.AdminID, OwnerID: testutil.UserID, Status: status}, nil
		},
		HandOverReservationFn: func(context.Context, uuid.UUID) (*model.GiftReservation, error) {
			return &model.GiftReservation{ID: id, GiverID: testutil.AdminID, OwnerID: testutil.UserID, Status: model.GiftGiven,
				CollectionID: &colID, Giver: &model.ReviewAuthor{Username: "admin"}}, nil
		},
		CreateNotificationsFn: func(_ context.Context, to []uuid.UUID, kind string, p any) error {
			notified, kinds = append(notified, to...), append(kinds, kind)
			payload, _ = p.(map[string]any)
			return nil
		},
	}
	path := "/api/v1/reservations/" + id.String() + "/hand-over"
	if resp, body := call(t, mock, "POST", path, testutil.AdminToken(), nil); resp.StatusCode != http.StatusOK ||
		len(notified) != 1 || notified[0] != testutil.UserID || kinds[0] != model.NotifyGiftGiven ||
		payload["by"] != "admin" || payload["collection_id"] != colID {
		t.Fatalf("hand over: %d %s %v %v", resp.StatusCode, body, kinds, payload)
	}
	if resp, _ := call(t, mock, "POST", path, testutil.UserToken(), nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("by the owner of the list: %d", resp.StatusCode)
	}
	status = model.GiftGiven
	if resp, body := call(t, mock, "POST", path, testutil.AdminToken(), nil); resp.StatusCode != http.StatusConflict || !strings.Contains(body, "gift_state") {
		t.Errorf("twice: %d %s", resp.StatusCode, body)
	}
}

// ownerGiftMock: a gift handed over by the admin on a lot of the user's wish
// list, and the user's Possession, which accepts the item.
func ownerGiftMock() (*model.Collection, *model.Collection, uuid.UUID, *testutil.MockStore) {
	wish := sampleCollection(testutil.UserID)
	wish.CollectionType.AttributesSchema = json.RawMessage(wishSchema)
	owned := sampleCollection(testutil.UserID)
	owned.ID = uuid.New()
	owned.CollectionType.AttributesSchema = json.RawMessage(`{"type":"object","properties":{"condition":{"type":"string"}}}`)
	giftID, typeID := uuid.New(), uuid.New()
	status := model.GiftGiven
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			switch id {
			case wish.ID:
				return wish, nil
			case owned.ID:
				return owned, nil
			}
			return nil, store.ErrNotFound
		},
		GetReservationFn: func(_ context.Context, id uuid.UUID) (*model.GiftReservation, error) {
			if id != giftID {
				return nil, store.ErrNotFound
			}
			return &model.GiftReservation{ID: id, GiverID: testutil.AdminID, OwnerID: testutil.UserID, Status: status,
				CollectionID: &wish.ID, Owner: &model.ReviewAuthor{Username: "user"},
				Item: &model.Item{ItemTypeID: typeID}}, nil
		},
	}
	return wish, owned, giftID, mock
}

func TestReceiveGift(t *testing.T) {
	wish, owned, giftID, mock := ownerGiftMock()
	var into *uuid.UUID
	var attrs json.RawMessage
	mock.ReceiveGiftFn = func(_ context.Context, _ uuid.UUID, c *uuid.UUID, a json.RawMessage) (*model.GiftReservation, error) {
		into, attrs = c, a
		return &model.GiftReservation{}, nil
	}
	var kinds []string
	mock.CreateNotificationsFn = func(_ context.Context, to []uuid.UUID, kind string, _ any) error {
		if to[0] != testutil.AdminID {
			t.Errorf("the giver is told, got %v", to)
		}
		kinds = append(kinds, kind)
		return nil
	}
	path := "/api/v1/collections/" + wish.ID.String() + "/gifts/" + giftID.String() + "/receive"
	user := testutil.UserToken()

	if resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"collection_id":"`+owned.ID.String()+`","attributes":{"condition":"new"}}`)); resp.StatusCode != http.StatusNoContent ||
		into == nil || *into != owned.ID || !strings.Contains(string(attrs), "new") || kinds[0] != model.NotifyGiftReceived {
		t.Fatalf("into Possession: %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, mock, "POST", path, user, strings.NewReader(`{}`)); resp.StatusCode != http.StatusNoContent || into != nil {
		t.Errorf("simply removed: %d %v", resp.StatusCode, into)
	}
	if resp, _ := call(t, mock, "POST", path, user, strings.NewReader(`{"collection_id":"`+owned.ID.String()+`","attributes":{"condition":3}}`)); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("invalid attributes: %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "POST", path, user, strings.NewReader(`{"collection_id":"`+wish.ID.String()+`"}`)); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("into the same list: %d", resp.StatusCode)
	}
	owned.ItemTypeIDs = []uuid.UUID{uuid.New()}
	if resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"collection_id":"`+owned.ID.String()+`"}`)); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "item_type_not_allowed") {
		t.Errorf("collection not accepting the item: %d %s", resp.StatusCode, body)
	}
	owned.ItemTypeIDs = nil
	owned.UserID = testutil.AdminID
	if resp, _ := call(t, mock, "POST", path, user, strings.NewReader(`{"collection_id":"`+owned.ID.String()+`"}`)); resp.StatusCode != http.StatusNotFound {
		t.Errorf("another's collection: %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{}`)); resp.StatusCode != http.StatusNotFound {
		t.Errorf("the giver confirms: %d", resp.StatusCode)
	}
}

// A reservation still secret answers 404 to the owner, as an unknown one.
func TestReceiveGift_SecretReservation(t *testing.T) {
	wish, _, giftID, mock := ownerGiftMock()
	mock.GetReservationFn = func(_ context.Context, id uuid.UUID) (*model.GiftReservation, error) {
		return &model.GiftReservation{ID: id, Status: model.GiftReserved, CollectionID: &wish.ID}, nil
	}
	for _, action := range []string{"receive", "decline"} {
		path := "/api/v1/collections/" + wish.ID.String() + "/gifts/" + giftID.String() + "/" + action
		if resp, _ := call(t, mock, "POST", path, testutil.UserToken(), strings.NewReader(`{}`)); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s a secret reservation: %d", action, resp.StatusCode)
		}
	}
}

func TestDeclineGift(t *testing.T) {
	wish, _, giftID, mock := ownerGiftMock()
	declined := false
	mock.DeclineGiftFn = func(context.Context, uuid.UUID) error {
		declined = true
		return nil
	}
	var kinds []string
	mock.CreateNotificationsFn = func(_ context.Context, _ []uuid.UUID, kind string, _ any) error {
		kinds = append(kinds, kind)
		return nil
	}
	path := "/api/v1/collections/" + wish.ID.String() + "/gifts/" + giftID.String() + "/decline"
	if resp, _ := call(t, mock, "POST", path, testutil.UserToken(), nil); resp.StatusCode != http.StatusNoContent || !declined ||
		len(kinds) != 1 || kinds[0] != model.NotifyGiftDeclined {
		t.Errorf("decline: %d %v %v", resp.StatusCode, declined, kinds)
	}
}

// The owner's list carries the gifts handed over, read by GivenGifts only:
// never the reservations still secret.
func TestCollectionItems_GivenGiftsForTheOwner(t *testing.T) {
	col, mock := lotMock()
	lotID := uuid.New()
	mock.ListCollectionItemsFn = func(context.Context, uuid.UUID, model.Pagination, store.CollectionItemFilters) ([]model.CollectionItem, int, error) {
		return []model.CollectionItem{{ID: lotID, Quantity: 2}}, 1, nil
	}
	mock.LotReservationsFn = func(context.Context, []uuid.UUID) (map[uuid.UUID][]model.GiftReservation, error) {
		t.Error("the owner's list reads the secret reservations")
		return nil, nil
	}
	mock.GivenGiftsFn = func(context.Context, []uuid.UUID) (map[uuid.UUID][]model.GiftReservation, error) {
		return map[uuid.UUID][]model.GiftReservation{lotID: {{ID: uuid.New(), Status: model.GiftGiven, Quantity: 1,
			Giver: &model.ReviewAuthor{Username: "camille"}, Note: "secret de camille"}}}, nil
	}
	resp, body := call(t, mock, "GET", "/api/v1/collections/"+col.ID.String()+"/items", testutil.UserToken(), nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"gifts"`) || !strings.Contains(body, "camille") || strings.Contains(body, "secret de camille") {
		t.Errorf("%d %s", resp.StatusCode, body)
	}
}
