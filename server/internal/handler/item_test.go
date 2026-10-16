package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

func TestCreateItem_WithValidation(t *testing.T) {
	itemTypeID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	itemType := &model.ItemType{
		ID:               itemTypeID,
		Name:             "Serie",
		AttributesSchema: json.RawMessage(`{"type":"object","required":["titre"],"properties":{"titre":{"type":"string","minLength":1}}}`),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	mock := &testutil.MockStore{
		GetItemTypeFn: func(_ context.Context, id uuid.UUID) (*model.ItemType, error) {
			if id == itemTypeID {
				return itemType, nil
			}
			return nil, store.ErrNotFound
		},
		CreateItemFn: func(_ context.Context, in model.ItemCreate) (*model.Item, error) {
			return &model.Item{
				ID: uuid.New(), ItemTypeID: in.ItemTypeID, ItemType: itemType,
				Attributes: in.Attributes, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	// Valid attributes
	body, _ := json.Marshal(model.ItemCreate{
		ItemTypeID: itemTypeID,
		Attributes: json.RawMessage(`{"titre":"Breaking Bad"}`),
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/items", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestCreateItem_ValidationError(t *testing.T) {
	itemTypeID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	itemType := &model.ItemType{
		ID:               itemTypeID,
		Name:             "Serie",
		AttributesSchema: json.RawMessage(`{"type":"object","required":["titre"],"properties":{"titre":{"type":"string","minLength":1}}}`),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	mock := &testutil.MockStore{
		GetItemTypeFn: func(_ context.Context, id uuid.UUID) (*model.ItemType, error) {
			if id == itemTypeID {
				return itemType, nil
			}
			return nil, store.ErrNotFound
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	// Missing required field "titre"
	body, _ := json.Marshal(model.ItemCreate{
		ItemTypeID: itemTypeID,
		Attributes: json.RawMessage(`{"synopsis":"test"}`),
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/items", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestCreateItem_UserForbidden(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.ItemCreate{
		ItemTypeID: uuid.New(),
		Attributes: json.RawMessage(`{}`),
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/items", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestListItems_Success(t *testing.T) {
	items := []model.Item{*testutil.SampleItem()}
	mock := &testutil.MockStore{
		ListItemsFn: func(_ context.Context, pg model.Pagination, f store.ItemFilters) ([]model.Item, int, error) {
			return items, 1, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items?search=breaking", nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result model.PaginatedResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 1 {
		t.Errorf("expected 1 total, got %d", result.Total)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items/"+uuid.New().String(), nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestListRelatedItemCollections(t *testing.T) {
	item := testutil.SampleItem()
	linked, col := uuid.New(), uuid.New()
	var gotUser uuid.UUID
	mock := &testutil.MockStore{
		GetItemFn: func(_ context.Context, id uuid.UUID) (*model.Item, error) {
			if id == item.ID {
				return item, nil
			}
			return nil, store.ErrNotFound
		},
		ListRelatedItemCollectionsFn: func(_ context.Context, userID, itemID uuid.UUID) ([]model.RelatedItemCollections, error) {
			gotUser = userID
			return []model.RelatedItemCollections{{ItemID: linked, Collections: []model.CollectionRef{{ID: col, Name: "Mes DVDs"}}}}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	get := func(id uuid.UUID) (*http.Response, []byte) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items/"+id.String()+"/relations/collections", nil)
		req.Header = testutil.AuthHeader(testutil.UserToken())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp, b
	}
	resp, body := get(item.ID)
	var out []model.RelatedItemCollections
	if resp.StatusCode != http.StatusOK || json.Unmarshal(body, &out) != nil ||
		len(out) != 1 || out[0].ItemID != linked || out[0].Collections[0].Name != "Mes DVDs" {
		t.Fatalf("got %d %s", resp.StatusCode, body)
	}
	if gotUser != testutil.UserID {
		t.Errorf("store called for user %s, want the authenticated user", gotUser)
	}
	if resp, _ := get(uuid.New()); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown item: %d", resp.StatusCode)
	}
}
