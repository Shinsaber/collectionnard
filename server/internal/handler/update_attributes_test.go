package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

// PUT /relations/{id} and PUT /collections/{c}/items/{i} replace `attributes`:
// omitting the field (or sending null) must be a 422, not a 500 or a silent wipe.

func TestUpdateRelation_MissingAttributes(t *testing.T) {
	relID := uuid.MustParse("f0000000-0000-0000-0000-000000000001")
	mock := &testutil.MockStore{
		GetRelationFn: func(_ context.Context, id uuid.UUID) (*model.ItemRelation, error) {
			return &model.ItemRelation{
				ID:           relID,
				RelationType: &model.RelationType{AttributesSchema: json.RawMessage(`{"type":"object"}`)},
				Attributes:   json.RawMessage(`{"disque":1}`),
				CreatedAt:    time.Now(),
			}, nil
		},
		UpdateRelationFn: func(_ context.Context, _ uuid.UUID, _ model.ItemRelationUpdate) (*model.ItemRelation, error) {
			t.Error("store must not be called when attributes are missing")
			return nil, store.ErrNotFound
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	for _, body := range []string{`{}`, `{"attributes":null}`} {
		req, _ := http.NewRequest("PUT", srv.URL+"/api/v1/relations/"+relID.String(), bytes.NewReader([]byte(body)))
		req.Header = testutil.AuthHeader(testutil.AdminToken())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("body %s: expected 422, got %d", body, resp.StatusCode)
		}
	}
}

func TestUpdateCollectionItem_MissingAttributes(t *testing.T) {
	colID := uuid.MustParse("d1000000-0000-0000-0000-000000000001")
	itemID := uuid.MustParse("e0000000-0000-0000-0000-000000000001")
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			return &model.Collection{ID: colID, UserID: testutil.UserID}, nil
		},
		UpdateCollectionItemFn: func(_ context.Context, _, _ uuid.UUID, _ model.CollectionItemUpdate) (*model.CollectionItem, error) {
			t.Error("store must not be called when attributes are missing")
			return nil, store.ErrNotFound
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	url := srv.URL + "/api/v1/collections/" + colID.String() + "/items/" + itemID.String()
	req, _ := http.NewRequest("PUT", url, bytes.NewReader([]byte(`{}`)))
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}
