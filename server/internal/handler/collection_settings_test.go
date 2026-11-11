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

// P7-1b: a collection narrows the item types of its type and chooses its default sort.
func TestCollectionSettings(t *testing.T) {
	volume, card := uuid.New(), uuid.New()
	col := sampleCollection(testutil.UserID)
	col.CollectionType.ItemTypeIDs = []uuid.UUID{volume, card}
	var updated model.CollectionUpdate
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			if id != col.ID {
				return nil, store.ErrNotFound
			}
			return col, nil
		},
		GetCollectionTypeFn: func(context.Context, uuid.UUID) (*model.CollectionType, error) { return col.CollectionType, nil },
		GetItemTypeFn: func(_ context.Context, id uuid.UUID) (*model.ItemType, error) {
			return &model.ItemType{ID: id, AttributesSchema: json.RawMessage(`{"type":"object","properties":{"number":{"type":"integer"},"photos":{"type":"object"}}}`)}, nil
		},
		CreateCollectionFn: func(_ context.Context, userID uuid.UUID, in model.CollectionCreate) (*model.Collection, error) {
			return &model.Collection{ID: uuid.New(), UserID: userID, ItemTypeIDs: in.ItemTypeIDs, DefaultSort: in.DefaultSort}, nil
		},
		UpdateCollectionFn: func(_ context.Context, _ uuid.UUID, in model.CollectionUpdate) (*model.Collection, error) {
			updated = in
			return col, nil
		},
	}
	user := testutil.UserToken()
	create := func(body string) (*http.Response, string) {
		return call(t, mock, "POST", "/api/v1/collections", user, strings.NewReader(body))
	}
	ctID := col.CollectionTypeID.String()

	resp, body := create(`{"collection_type_id":"` + ctID + `","name":"Mes mangas","item_type_ids":["` + volume.String() + `","` + volume.String() + `"],"default_sort":"attr.number"}`)
	var created model.Collection
	if resp.StatusCode != http.StatusCreated || json.Unmarshal([]byte(body), &created) != nil ||
		len(created.ItemTypeIDs) != 1 || created.DefaultSort != "attr.number" {
		t.Fatalf("specialized collection sorted by number: %d %s", resp.StatusCode, body)
	}
	if resp, body := create(`{"collection_type_id":"` + ctID + `","name":"x","item_type_ids":["` + uuid.New().String() + `"]}`); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "item_type_not_allowed") {
		t.Errorf("type not allowed by the collection type: %d %s", resp.StatusCode, body)
	}
	if resp, body := create(`{"collection_type_id":"` + ctID + `","name":"x","default_sort":"attr.number"}`); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "invalid_sort") {
		t.Errorf("attribute sort of a generic collection: %d %s", resp.StatusCode, body)
	}
	for _, sort := range []string{"title", "relevance", "attr.missing", "attr.photos"} {
		resp, _ := create(`{"collection_type_id":"` + ctID + `","name":"x","item_type_ids":["` + volume.String() + `"],"default_sort":"` + sort + `"}`)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("default_sort %q: expected 422, got %d", sort, resp.StatusCode)
		}
	}

	path := "/api/v1/collections/" + col.ID.String()
	if resp, body := call(t, mock, "PUT", path, user, strings.NewReader(`{"item_type_ids":["`+card.String()+`"],"default_sort":"-attr.number"}`)); resp.StatusCode != http.StatusOK ||
		updated.ItemTypeIDs == nil || len(*updated.ItemTypeIDs) != 1 || *updated.DefaultSort != "-attr.number" {
		t.Errorf("update: %d %s %+v", resp.StatusCode, body, updated)
	}
	if resp, _ := call(t, mock, "PUT", path, user, strings.NewReader(`{"default_sort":"-quantity"}`)); resp.StatusCode != http.StatusOK {
		t.Errorf("common sort on a generic collection: %d", resp.StatusCode)
	}
}

func TestCollectionSettings_DefaultSortApplied(t *testing.T) {
	volume := uuid.New()
	col := sampleCollection(testutil.UserID)
	col.ItemTypeIDs = []uuid.UUID{volume}
	col.DefaultSort = "attr.number"
	var got store.CollectionItemFilters
	mock := &testutil.MockStore{
		GetCollectionFn: func(context.Context, uuid.UUID) (*model.Collection, error) { return col, nil },
		GetItemTypeFn: func(_ context.Context, id uuid.UUID) (*model.ItemType, error) {
			return &model.ItemType{ID: id, AttributesSchema: json.RawMessage(`{"type":"object","properties":{"number":{"type":"integer"},"title":{"type":"string"}}}`)}, nil
		},
		ListCollectionItemsFn: func(_ context.Context, _ uuid.UUID, _ model.Pagination, f store.CollectionItemFilters) ([]model.CollectionItem, int, error) {
			got = f
			return nil, 0, nil
		},
	}
	user := testutil.UserToken()
	items := "/api/v1/collections/" + col.ID.String() + "/items"

	if resp, _ := call(t, mock, "GET", items, user, nil); resp.StatusCode != http.StatusOK || got.SortAttr == nil || got.SortAttr.Key != "number" || got.ItemTypeID != nil {
		t.Errorf("default sort of a specialized collection: %d %+v", resp.StatusCode, got)
	}
	if resp, _ := call(t, mock, "GET", items+"?sort=-created_at", user, nil); resp.StatusCode != http.StatusOK || got.SortAttr != nil || got.Sort != "-created_at" {
		t.Errorf("an explicit sort wins: %d %+v", resp.StatusCode, got)
	}
	if resp, _ := call(t, mock, "GET", items+"?sort=attr.title&attr.number.gte=2", user, nil); resp.StatusCode != http.StatusOK || got.SortAttr == nil || got.SortAttr.Key != "title" || len(got.Attrs) != 1 || got.ItemTypeID != nil {
		t.Errorf("attributes of a specialized collection without item_type_id: %d %+v", resp.StatusCode, got)
	}

	// The attribute is gone from the schema: back to the latest added, without error.
	col.DefaultSort = "attr.gone"
	if resp, _ := call(t, mock, "GET", items, user, nil); resp.StatusCode != http.StatusOK || got.SortAttr != nil || got.Sort != "" {
		t.Errorf("stale default sort: %d %+v", resp.StatusCode, got)
	}
	// A generic collection: common default sort.
	col.ItemTypeIDs, col.DefaultSort = nil, "-quantity"
	if resp, _ := call(t, mock, "GET", items, user, nil); resp.StatusCode != http.StatusOK || got.Sort != "-quantity" {
		t.Errorf("common default sort: %d %+v", resp.StatusCode, got)
	}
}
