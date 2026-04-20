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

func TestCreateRelation_TypeConstraintValidation(t *testing.T) {
	serieTypeID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	episodeTypeID := uuid.MustParse("b0000000-0000-0000-0000-000000000002")
	rtID := uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	serieItemID := uuid.MustParse("e0000000-0000-0000-0000-000000000001")
	episodeItemID := uuid.MustParse("e0000000-0000-0000-0000-000000000002")
	wrongItemID := uuid.MustParse("e0000000-0000-0000-0000-000000000099")

	rt := &model.RelationType{
		ID: rtID, Name: "contient_episode",
		SourceItemTypeIDs: []uuid.UUID{serieTypeID}, TargetItemTypeIDs: []uuid.UUID{episodeTypeID},
		AttributesSchema: json.RawMessage(`{}`),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	mock := &testutil.MockStore{
		GetRelationTypeFn: func(_ context.Context, id uuid.UUID) (*model.RelationType, error) {
			if id == rtID { return rt, nil }
			return nil, store.ErrNotFound
		},
		GetItemFn: func(_ context.Context, id uuid.UUID) (*model.Item, error) {
			switch id {
			case serieItemID:
				return &model.Item{ID: serieItemID, ItemTypeID: serieTypeID, Attributes: json.RawMessage(`{}`)}, nil
			case episodeItemID:
				return &model.Item{ID: episodeItemID, ItemTypeID: episodeTypeID, Attributes: json.RawMessage(`{}`)}, nil
			case wrongItemID:
				// Wrong type: this is a Serie, not an Episode
				return &model.Item{ID: wrongItemID, ItemTypeID: serieTypeID, Attributes: json.RawMessage(`{}`)}, nil
			}
			return nil, store.ErrNotFound
		},
		CreateRelationFn: func(_ context.Context, in model.ItemRelationCreate) (*model.ItemRelation, error) {
			return &model.ItemRelation{
				ID: uuid.New(), RelationTypeID: in.RelationTypeID,
				SourceItemID: in.SourceItemID, TargetItemID: in.TargetItemID,
				Attributes: in.Attributes, CreatedAt: time.Now(),
			}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	// Valid relation: Serie → Episode
	body, _ := json.Marshal(model.ItemRelationCreate{
		RelationTypeID: rtID, SourceItemID: serieItemID, TargetItemID: episodeItemID,
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/relations", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil { t.Fatalf("request: %v", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(b))
	}

	// Invalid: target is wrong type (Serie instead of Episode)
	body, _ = json.Marshal(model.ItemRelationCreate{
		RelationTypeID: rtID, SourceItemID: serieItemID, TargetItemID: wrongItemID,
	})
	req, _ = http.NewRequest("POST", srv.URL+"/api/v1/relations", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp2, _ := http.DefaultClient.Do(req)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp2.StatusCode)
	}
}

func TestCreateRelation_SelfReference(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	itemID := uuid.New()
	body, _ := json.Marshal(model.ItemRelationCreate{
		RelationTypeID: uuid.New(), SourceItemID: itemID, TargetItemID: itemID,
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/relations", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for self-reference, got %d", resp.StatusCode)
	}
}

func TestCreateRelation_UserForbidden(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.ItemRelationCreate{
		RelationTypeID: uuid.New(), SourceItemID: uuid.New(), TargetItemID: uuid.New(),
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/relations", bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}
