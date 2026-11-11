package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

// lotMock: the user's collection (sampleCollection), with a schema that only
// accepts a string `condition`.
func lotMock() (*model.Collection, *testutil.MockStore) {
	col := sampleCollection(testutil.UserID)
	ct := &model.CollectionType{ID: col.CollectionTypeID,
		AttributesSchema: json.RawMessage(`{"type":"object","properties":{"condition":{"type":"string"}}}`)}
	return col, &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			if id != col.ID {
				return nil, store.ErrNotFound
			}
			return col, nil
		},
		GetCollectionTypeFn: func(context.Context, uuid.UUID) (*model.CollectionType, error) { return ct, nil },
	}
}

func TestLots_Quantity(t *testing.T) {
	col, mock := lotMock()
	var added model.CollectionItemAdd
	mock.AddCollectionItemFn = func(_ context.Context, _ uuid.UUID, in model.CollectionItemAdd) (*model.CollectionItem, error) {
		added = in
		return &model.CollectionItem{ID: uuid.New(), ItemID: in.ItemID, Quantity: *in.Quantity}, nil
	}
	var updated model.CollectionItemUpdate
	mock.UpdateCollectionItemFn = func(_ context.Context, _, _ uuid.UUID, in model.CollectionItemUpdate) (*model.CollectionItem, error) {
		updated = in
		return &model.CollectionItem{}, nil
	}
	user := testutil.UserToken()
	items := "/api/v1/collections/" + col.ID.String() + "/items"
	lot := "/api/v1/collections/" + col.ID.String() + "/lots/" + uuid.New().String()

	itemID := uuid.New()
	if resp, body := call(t, mock, "POST", items, user, strings.NewReader(`{"item_id":"`+itemID.String()+`","quantity":5}`)); resp.StatusCode != http.StatusCreated || added.Quantity == nil || *added.Quantity != 5 {
		t.Fatalf("add 5: %d %s", resp.StatusCode, body)
	}
	for _, q := range []string{"0", "-1", "100001"} {
		if resp, body := call(t, mock, "POST", items, user, strings.NewReader(`{"item_id":"`+itemID.String()+`","quantity":`+q+`}`)); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "invalid_quantity") {
			t.Errorf("add %s: %d %s", q, resp.StatusCode, body)
		}
	}
	if resp, body := call(t, mock, "PUT", lot, user, strings.NewReader(`{"attributes":{},"quantity":3}`)); resp.StatusCode != http.StatusOK || updated.Quantity == nil || *updated.Quantity != 3 {
		t.Errorf("update quantity: %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, mock, "PUT", lot, user, strings.NewReader(`{"attributes":{}}`)); resp.StatusCode != http.StatusOK || updated.Quantity != nil {
		t.Errorf("update without quantity must leave it unchanged: %d %v", resp.StatusCode, updated.Quantity)
	}
	if resp, _ := call(t, mock, "PUT", lot, user, strings.NewReader(`{"attributes":{},"quantity":0}`)); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("update quantity 0: %d", resp.StatusCode)
	}
}

func TestLots_Split(t *testing.T) {
	col, mock := lotMock()
	lotID := uuid.New()
	var gotQty int
	var gotAttrs json.RawMessage
	mock.SplitCollectionLotFn = func(_ context.Context, colID, id uuid.UUID, quantity int, attrs json.RawMessage) (*model.CollectionItem, error) {
		if colID != col.ID || id != lotID {
			return nil, store.ErrNotFound
		}
		if quantity >= 5 {
			return nil, store.ErrInvalidQuantity
		}
		gotQty, gotAttrs = quantity, attrs
		return &model.CollectionItem{ID: uuid.New(), Quantity: quantity}, nil
	}
	user := testutil.UserToken()
	path := "/api/v1/collections/" + col.ID.String() + "/lots/" + lotID.String() + "/split"

	if resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"quantity":1,"attributes":{"condition":"damaged"}}`)); resp.StatusCode != http.StatusCreated || gotQty != 1 || !strings.Contains(string(gotAttrs), "damaged") {
		t.Fatalf("split: %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, mock, "POST", path, user, strings.NewReader(`{"quantity":2}`)); resp.StatusCode != http.StatusCreated || len(gotAttrs) != 0 {
		t.Errorf("split without attributes (copy of the lot's): %d %s", resp.StatusCode, gotAttrs)
	}
	if resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"quantity":5}`)); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "invalid_quantity") {
		t.Errorf("split all copies: %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, mock, "POST", path, user, strings.NewReader(`{"quantity":1,"attributes":{"condition":3}}`)); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("split invalid attributes: %d", resp.StatusCode)
	}
	other := "/api/v1/collections/" + uuid.New().String() + "/lots/" + lotID.String() + "/split"
	if resp, _ := call(t, mock, "POST", other, user, strings.NewReader(`{"quantity":1}`)); resp.StatusCode != http.StatusNotFound {
		t.Errorf("split in an unknown collection: %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{"quantity":1}`)); resp.StatusCode != http.StatusNotFound {
		t.Errorf("split in another user's collection: %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "POST", path, "", strings.NewReader(`{"quantity":1}`)); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous split: %d", resp.StatusCode)
	}
}

func TestLots_Merge(t *testing.T) {
	col, mock := lotMock()
	lotID, into := uuid.New(), uuid.New()
	mock.MergeCollectionLotsFn = func(_ context.Context, colID, id, target uuid.UUID, _ []string) (*model.CollectionItem, error) {
		switch {
		case colID != col.ID || id != lotID:
			return nil, store.ErrNotFound
		case target == lotID:
			return nil, store.ErrNotMergeable
		}
		return &model.CollectionItem{ID: target, Quantity: 7}, nil
	}
	user := testutil.UserToken()
	path := "/api/v1/collections/" + col.ID.String() + "/lots/" + lotID.String() + "/merge"

	resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"into":"`+into.String()+`"}`))
	var merged model.CollectionItem
	if resp.StatusCode != http.StatusOK || json.Unmarshal([]byte(body), &merged) != nil || merged.ID != into || merged.Quantity != 7 {
		t.Fatalf("merge: %d %s", resp.StatusCode, body)
	}
	if resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"into":"`+lotID.String()+`"}`)); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "lots_not_mergeable") {
		t.Errorf("merge into itself: %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, mock, "POST", path, user, strings.NewReader(`{}`)); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("merge without into: %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "POST", path, testutil.AdminToken(), strings.NewReader(`{"into":"`+into.String()+`"}`)); resp.StatusCode != http.StatusNotFound {
		t.Errorf("merge in another user's collection: %d", resp.StatusCode)
	}
}

func TestLots_MergeGalleriesOfTheType(t *testing.T) {
	col, mock := lotMock()
	col.CollectionType.AttributesSchema = json.RawMessage(`{"type":"object","properties":{"condition":{"type":"string"},
		"photos":{"type":"object","x-widget":"image-gallery"},"scans":{"type":"object","x-widget":"image-gallery"}}}`)
	var got []string
	mock.MergeCollectionLotsFn = func(_ context.Context, _, _, into uuid.UUID, galleries []string) (*model.CollectionItem, error) {
		got = galleries
		return &model.CollectionItem{ID: into}, nil
	}
	path := "/api/v1/collections/" + col.ID.String() + "/lots/" + uuid.New().String() + "/merge"
	if resp, body := call(t, mock, "POST", path, testutil.UserToken(), strings.NewReader(`{"into":"`+uuid.New().String()+`"}`)); resp.StatusCode != http.StatusOK {
		t.Fatalf("merge: %d %s", resp.StatusCode, body)
	}
	if strings.Join(got, ",") != "photos,scans" {
		t.Errorf("galleries passed to the store: %v", got)
	}
}

func TestLots_ListFilters(t *testing.T) {
	col, mock := lotMock()
	typeID := uuid.New()
	mock.GetItemTypeFn = func(context.Context, uuid.UUID) (*model.ItemType, error) {
		return &model.ItemType{ID: typeID, AttributesSchema: json.RawMessage(`{"type":"object","properties":{"number":{"type":"integer"}}}`)}, nil
	}
	var got store.CollectionItemFilters
	mock.ListCollectionItemsFn = func(_ context.Context, _ uuid.UUID, _ model.Pagination, f store.CollectionItemFilters) ([]model.CollectionItem, int, error) {
		got = f
		return nil, 0, nil
	}
	user := testutil.UserToken()
	base := "/api/v1/collections/" + col.ID.String() + "/items"

	if resp, body := call(t, mock, "GET", base+"?multiple=true&search=bad&sort=-quantity", user, nil); resp.StatusCode != http.StatusOK ||
		!got.Multiple || got.Search != "bad" || got.Sort != "-quantity" {
		t.Fatalf("filters: %d %s %+v", resp.StatusCode, body, got)
	}
	if resp, _ := call(t, mock, "GET", base+"?item_type_id="+typeID.String()+"&sort=attr.number&attr.number.gte=3", user, nil); resp.StatusCode != http.StatusOK ||
		got.SortAttr == nil || got.SortAttr.Key != "number" || len(got.Attrs) != 1 {
		t.Errorf("attribute sort and filter: %d %+v", resp.StatusCode, got)
	}
	for _, q := range []string{"?sort=title", "?multiple=yes", "?attr.number=3", "?sort=attr.number"} {
		if resp, _ := call(t, mock, "GET", base+q, user, nil); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", q, resp.StatusCode)
		}
	}
}
