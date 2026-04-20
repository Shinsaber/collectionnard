package handler_test

import (
	"context"
	"bytes"
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

func sampleItemTypes() []model.ItemType {
	return []model.ItemType{
		{
			ID: uuid.MustParse("b0000000-0000-0000-0000-000000000001"),
			Name: "Serie", AttributesSchema: json.RawMessage(`{}`),
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: uuid.MustParse("b0000000-0000-0000-0000-000000000002"),
			Name: "Episode", AttributesSchema: json.RawMessage(`{}`),
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
}

func TestListItemTypes_Success(t *testing.T) {
	items := sampleItemTypes()
	mock := &testutil.MockStore{
		ListItemTypesFn: func(_ context.Context, pg model.Pagination) ([]model.ItemType, int, error) {
			return items, len(items), nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/item-types", nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var result model.PaginatedResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("expected 2 total, got %d", result.Total)
	}
}

func TestListItemTypes_Unauthenticated(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/item-types")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestGetItemType_Success(t *testing.T) {
	it := testutil.SampleItemType()
	mock := &testutil.MockStore{
		GetItemTypeFn: func(_ context.Context, id uuid.UUID) (*model.ItemType, error) {
			if id == it.ID {
				return it, nil
			}
			return nil, store.ErrNotFound
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/item-types/"+it.ID.String(), nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetItemType_NotFound(t *testing.T) {
	mock := &testutil.MockStore{} // default returns ErrNotFound
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/item-types/"+uuid.New().String(), nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestCreateItemType_AdminOnly(t *testing.T) {
	mock := &testutil.MockStore{
		CreateItemTypeFn: func(_ context.Context, in model.ItemTypeCreate) (*model.ItemType, error) {
			return testutil.SampleItemType(), nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.ItemTypeCreate{
		Name:             "Serie",
		AttributesSchema: json.RawMessage(`{}`),
	})

	// User should be forbidden
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/item-types", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for user, got %d", resp.StatusCode)
	}

	// Admin should succeed
	req, _ = http.NewRequest("POST", srv.URL+"/api/v1/item-types", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201 for admin, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestCreateItemType_Conflict(t *testing.T) {
	mock := &testutil.MockStore{
		CreateItemTypeFn: func(_ context.Context, in model.ItemTypeCreate) (*model.ItemType, error) {
			return nil, store.ErrConflict
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.ItemTypeCreate{Name: "Serie", AttributesSchema: json.RawMessage(`{}`)})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/item-types", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestDeleteItemType_AdminOnly(t *testing.T) {
	id := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	mock := &testutil.MockStore{
		DeleteItemTypeFn: func(_ context.Context, delID uuid.UUID) error {
			if delID == id {
				return nil
			}
			return store.ErrNotFound
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	// User → 403
	req, _ := http.NewRequest("DELETE", srv.URL+"/api/v1/item-types/"+id.String(), nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}

	// Admin → 204
	req, _ = http.NewRequest("DELETE", srv.URL+"/api/v1/item-types/"+id.String(), nil)
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}
