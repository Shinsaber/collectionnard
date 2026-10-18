package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

func TestGetCollectionStats(t *testing.T) {
	mine := sampleCollection(testutil.UserID)
	other := sampleCollection(testutil.AdminID)
	other.ID = uuid.New()
	var asked uuid.UUID
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			switch id {
			case mine.ID:
				return mine, nil
			case other.ID:
				return other, nil
			}
			return nil, store.ErrNotFound
		},
		GetCollectionStatsFn: func(_ context.Context, id uuid.UUID, _ []string) (*model.CollectionStats, error) {
			asked = id
			return &model.CollectionStats{CollectionID: id, TotalItems: 3}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	status, _, raw := callJSON(t, srv, http.MethodGet, "/api/v1/collections/"+mine.ID.String()+"/stats", testutil.UserToken(), nil)
	var got model.CollectionStats
	if status != http.StatusOK || json.Unmarshal(raw, &got) != nil || got.TotalItems != 3 || asked != mine.ID {
		t.Fatalf("own collection: %d %s", status, raw)
	}

	asked = uuid.Nil
	if status, _, _ := callJSON(t, srv, http.MethodGet, "/api/v1/collections/"+other.ID.String()+"/stats", testutil.UserToken(), nil); status != http.StatusNotFound || asked != uuid.Nil {
		t.Errorf("someone else's collection: %d (stats computed: %v)", status, asked != uuid.Nil)
	}
	if status, _, _ := callJSON(t, srv, http.MethodGet, "/api/v1/collections/"+uuid.NewString()+"/stats", testutil.UserToken(), nil); status != http.StatusNotFound {
		t.Errorf("unknown collection: %d", status)
	}
	if status, _, _ := callJSON(t, srv, http.MethodGet, "/api/v1/collections/nope/stats", testutil.UserToken(), nil); status != http.StatusBadRequest {
		t.Errorf("invalid id: %d", status)
	}
	if status, _, _ := callJSON(t, srv, http.MethodGet, "/api/v1/collections/"+mine.ID.String()+"/stats", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", status)
	}
}

func TestGetMyStats(t *testing.T) {
	var asked uuid.UUID
	mock := &testutil.MockStore{
		GetUserStatsFn: func(_ context.Context, userID uuid.UUID) (*model.MeStats, error) {
			asked = userID
			return &model.MeStats{TotalCollections: 2}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	status, _, raw := callJSON(t, srv, http.MethodGet, "/api/v1/me/stats", testutil.UserToken(), nil)
	if status != http.StatusOK || asked != testutil.UserID {
		t.Fatalf("got %d %s for %v", status, raw, asked)
	}
	if status, _, _ := callJSON(t, srv, http.MethodGet, "/api/v1/me/stats", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", status)
	}
}
