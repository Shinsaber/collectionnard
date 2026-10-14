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

func sampleCollection(userID uuid.UUID) *model.Collection {
	return &model.Collection{
		ID:               uuid.MustParse("f0000000-0000-0000-0000-000000000001"),
		UserID:           userID,
		CollectionTypeID: uuid.MustParse("d0000000-0000-0000-0000-000000000001"),
		CollectionType: &model.CollectionType{
			ID:   uuid.MustParse("d0000000-0000-0000-0000-000000000001"),
			Name: "Possession", AttributesSchema: json.RawMessage(`{}`),
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		Name: "Mes DVDs", ItemCount: 5,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func TestListCollections_OwnerOnly(t *testing.T) {
	col := sampleCollection(testutil.UserID)
	mock := &testutil.MockStore{
		ListCollectionsFn: func(_ context.Context, userID uuid.UUID, pg model.Pagination, ctID *uuid.UUID) ([]model.Collection, int, error) {
			if userID == testutil.UserID {
				return []model.Collection{*col}, 1, nil
			}
			return nil, 0, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/collections", nil)
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
	if result.Total != 1 {
		t.Errorf("expected 1, got %d", result.Total)
	}
}

func TestCreateCollection_Success(t *testing.T) {
	ctID := uuid.MustParse("d0000000-0000-0000-0000-000000000001")
	mock := &testutil.MockStore{
		CreateCollectionFn: func(_ context.Context, userID uuid.UUID, in model.CollectionCreate) (*model.Collection, error) {
			return sampleCollection(userID), nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.CollectionCreate{CollectionTypeID: ctID, Name: "Mes DVDs"})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/collections", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.UserToken())
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

func TestGetCollection_OtherUserDenied(t *testing.T) {
	// Collection belongs to AdminID, but UserID tries to access it
	col := sampleCollection(testutil.AdminID)
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			if id == col.ID {
				return col, nil
			}
			return nil, store.ErrNotFound
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/collections/"+col.ID.String(), nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 (ownership check), got %d", resp.StatusCode)
	}
}

func TestAddItemToCollection_Conflict(t *testing.T) {
	col := sampleCollection(testutil.UserID)
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			if id == col.ID {
				return col, nil
			}
			return nil, store.ErrNotFound
		},
		GetCollectionTypeFn: func(_ context.Context, id uuid.UUID) (*model.CollectionType, error) {
			return col.CollectionType, nil
		},
		AddCollectionItemFn: func(_ context.Context, colID uuid.UUID, in model.CollectionItemAdd) (*model.CollectionItem, error) {
			return nil, store.ErrConflict
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.CollectionItemAdd{ItemID: uuid.New()})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/collections/"+col.ID.String()+"/items", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestDeleteCollection_OwnerOnly(t *testing.T) {
	col := sampleCollection(testutil.UserID)
	deleted := false
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			if id == col.ID {
				return col, nil
			}
			return nil, store.ErrNotFound
		},
		DeleteCollectionFn: func(_ context.Context, id uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/v1/collections/"+col.ID.String(), nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if !deleted {
		t.Error("expected delete to be called")
	}
}
