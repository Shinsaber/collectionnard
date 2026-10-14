package handler_test

// Table-driven coverage of the handlers that had no dedicated test: for each
// route, the success path, the authorization path (401 / 403 / owner check)
// and the store error mapping (404 / 409).

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

var (
	someID    = uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	otherID   = uuid.MustParse("c0000000-0000-0000-0000-000000000002")
	emptyJSON = json.RawMessage(`{}`)
)

type routeCase struct {
	name   string
	method string
	path   string
	token  string // "" = anonymous
	body   string
	mock   *testutil.MockStore
	status int
	check  func(t *testing.T, body []byte)
}

func runRouteCases(t *testing.T, cases []routeCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := tc.mock
			if mock == nil {
				mock = &testutil.MockStore{}
			}
			srv := testutil.NewTestServer(mock)
			defer srv.Close()

			var body io.Reader
			if tc.body != "" {
				body = bytes.NewReader([]byte(tc.body))
			}
			req, _ := http.NewRequest(tc.method, srv.URL+"/api/v1"+tc.path, body)
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, resp.StatusCode, string(b))
			}
			if tc.check != nil {
				tc.check(t, b)
			}
		})
	}
}

func sampleRelationType() *model.RelationType {
	return &model.RelationType{
		ID: someID, Name: "contient_saison",
		SourceItemTypeIDs: []uuid.UUID{uuid.New()}, TargetItemTypeIDs: []uuid.UUID{uuid.New()},
		AttributesSchema: emptyJSON, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func sampleCollectionType() *model.CollectionType {
	return &model.CollectionType{
		ID: someID, Name: "Possession", AttributesSchema: emptyJSON,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func sampleRelation() *model.ItemRelation {
	return &model.ItemRelation{
		ID: someID, RelationTypeID: otherID, SourceItemID: uuid.New(), TargetItemID: uuid.New(),
		Attributes: emptyJSON, CreatedAt: time.Now(),
	}
}

// --- Relation types ---

func TestRelationTypes_Coverage(t *testing.T) {
	rt := sampleRelationType()
	listMock := &testutil.MockStore{
		ListRelationTypesFn: func(_ context.Context, pg model.Pagination, s, tg *uuid.UUID) ([]model.RelationType, int, error) {
			if s == nil || *s != otherID {
				t.Errorf("source filter not forwarded: %v", s)
			}
			return []model.RelationType{*rt}, 1, nil
		},
	}
	getMock := &testutil.MockStore{
		GetRelationTypeFn: func(_ context.Context, id uuid.UUID) (*model.RelationType, error) {
			if id == someID {
				return rt, nil
			}
			return nil, store.ErrNotFound
		},
	}
	createMock := &testutil.MockStore{
		CreateRelationTypeFn: func(_ context.Context, in model.RelationTypeCreate) (*model.RelationType, error) {
			if in.Name == "dup" {
				return nil, store.ErrConflict
			}
			out := sampleRelationType()
			out.Name = in.Name
			return out, nil
		},
	}
	updateMock := &testutil.MockStore{
		UpdateRelationTypeFn: func(_ context.Context, id uuid.UUID, in model.RelationTypeUpdate) (*model.RelationType, error) {
			if id != someID {
				return nil, store.ErrNotFound
			}
			out := sampleRelationType()
			if in.Name != nil {
				out.Name = *in.Name
			}
			return out, nil
		},
	}
	deleteMock := &testutil.MockStore{
		DeleteRelationTypeFn: func(_ context.Context, id uuid.UUID) error {
			switch id {
			case someID:
				return nil
			case otherID:
				return store.ErrConflict // still referenced by relations
			}
			return store.ErrNotFound
		},
	}

	runRouteCases(t, []routeCase{
		{"list ok", "GET", "/relation-types?source_item_type_id=" + otherID.String(), testutil.UserToken(), "", listMock, 200,
			func(t *testing.T, b []byte) {
				var res model.PaginatedResponse
				_ = json.Unmarshal(b, &res)
				if res.Total != 1 {
					t.Errorf("total = %d", res.Total)
				}
			}},
		{"list anonymous", "GET", "/relation-types", "", "", nil, 401, nil},
		{"get ok", "GET", "/relation-types/" + someID.String(), testutil.UserToken(), "", getMock, 200, nil},
		{"get 404", "GET", "/relation-types/" + uuid.New().String(), testutil.UserToken(), "", getMock, 404, nil},
		{"get bad uuid", "GET", "/relation-types/abc", testutil.UserToken(), "", getMock, 400, nil},
		{"create ok", "POST", "/relation-types", testutil.AdminToken(), `{"name":"adapte","source_item_type_ids":[],"target_item_type_ids":[]}`, createMock, 201, nil},
		{"create conflict", "POST", "/relation-types", testutil.AdminToken(), `{"name":"dup"}`, createMock, 409, nil},
		{"create missing name", "POST", "/relation-types", testutil.AdminToken(), `{"name":""}`, createMock, 422, nil},
		{"create bad schema", "POST", "/relation-types", testutil.AdminToken(), `{"name":"x","attributes_schema":{"type":"string"}}`, createMock, 422, nil},
		{"create user forbidden", "POST", "/relation-types", testutil.UserToken(), `{"name":"x"}`, createMock, 403, nil},
		{"update ok", "PUT", "/relation-types/" + someID.String(), testutil.AdminToken(), `{"name":"renamed"}`, updateMock, 200,
			func(t *testing.T, b []byte) {
				var rt model.RelationType
				_ = json.Unmarshal(b, &rt)
				if rt.Name != "renamed" {
					t.Errorf("name = %q", rt.Name)
				}
			}},
		{"update 404", "PUT", "/relation-types/" + uuid.New().String(), testutil.AdminToken(), `{"name":"x"}`, updateMock, 404, nil},
		{"update user forbidden", "PUT", "/relation-types/" + someID.String(), testutil.UserToken(), `{"name":"x"}`, updateMock, 403, nil},
		{"delete ok", "DELETE", "/relation-types/" + someID.String(), testutil.AdminToken(), "", deleteMock, 204, nil},
		{"delete conflict", "DELETE", "/relation-types/" + otherID.String(), testutil.AdminToken(), "", deleteMock, 409, nil},
		{"delete 404", "DELETE", "/relation-types/" + uuid.New().String(), testutil.AdminToken(), "", deleteMock, 404, nil},
		{"delete user forbidden", "DELETE", "/relation-types/" + someID.String(), testutil.UserToken(), "", deleteMock, 403, nil},
	})
}

// --- Collection types ---

func TestCollectionTypes_Coverage(t *testing.T) {
	ct := sampleCollectionType()
	mock := &testutil.MockStore{
		ListCollectionTypesFn: func(_ context.Context, pg model.Pagination) ([]model.CollectionType, int, error) {
			return []model.CollectionType{*ct}, 1, nil
		},
		GetCollectionTypeFn: func(_ context.Context, id uuid.UUID) (*model.CollectionType, error) {
			if id == someID {
				return ct, nil
			}
			return nil, store.ErrNotFound
		},
		CreateCollectionTypeFn: func(_ context.Context, in model.CollectionTypeCreate) (*model.CollectionType, error) {
			if in.Name == "dup" {
				return nil, store.ErrConflict
			}
			return ct, nil
		},
		DeleteCollectionTypeFn: func(_ context.Context, id uuid.UUID) error {
			switch id {
			case someID:
				return nil
			case otherID:
				return store.ErrConflict
			}
			return store.ErrNotFound
		},
	}

	runRouteCases(t, []routeCase{
		{"list ok", "GET", "/collection-types", testutil.UserToken(), "", mock, 200, nil},
		{"list anonymous", "GET", "/collection-types", "", "", mock, 401, nil},
		{"get ok", "GET", "/collection-types/" + someID.String(), testutil.UserToken(), "", mock, 200, nil},
		{"get 404", "GET", "/collection-types/" + uuid.New().String(), testutil.UserToken(), "", mock, 404, nil},
		{"create ok", "POST", "/collection-types", testutil.AdminToken(), `{"name":"Wishlist"}`, mock, 201, nil},
		{"create conflict", "POST", "/collection-types", testutil.AdminToken(), `{"name":"dup"}`, mock, 409, nil},
		{"create missing name", "POST", "/collection-types", testutil.AdminToken(), `{}`, mock, 422, nil},
		{"create user forbidden", "POST", "/collection-types", testutil.UserToken(), `{"name":"x"}`, mock, 403, nil},
		{"delete ok", "DELETE", "/collection-types/" + someID.String(), testutil.AdminToken(), "", mock, 204, nil},
		{"delete conflict", "DELETE", "/collection-types/" + otherID.String(), testutil.AdminToken(), "", mock, 409, nil},
		{"delete 404", "DELETE", "/collection-types/" + uuid.New().String(), testutil.AdminToken(), "", mock, 404, nil},
		{"delete user forbidden", "DELETE", "/collection-types/" + someID.String(), testutil.UserToken(), "", mock, 403, nil},
	})
}

// --- Item types (update) and items (update / delete / collections) ---

func TestItemTypesAndItems_Coverage(t *testing.T) {
	it := testutil.SampleItemType()
	item := testutil.SampleItem()
	var deletedItem uuid.UUID
	var updatedTags *[]string
	mock := &testutil.MockStore{
		UpdateItemTypeFn: func(_ context.Context, id uuid.UUID, in model.ItemTypeUpdate) (*model.ItemType, error) {
			if id != it.ID {
				return nil, store.ErrNotFound
			}
			if in.Name != nil && *in.Name == "dup" {
				return nil, store.ErrConflict
			}
			return it, nil
		},
		GetItemFn: func(_ context.Context, id uuid.UUID) (*model.Item, error) {
			if id == item.ID {
				return item, nil
			}
			return nil, store.ErrNotFound
		},
		UpdateItemFn: func(_ context.Context, id uuid.UUID, in model.ItemUpdate) (*model.Item, error) {
			if id != item.ID {
				return nil, store.ErrNotFound
			}
			updatedTags = in.Tags
			return item, nil
		},
		DeleteItemFn: func(_ context.Context, id uuid.UUID) error {
			if id != item.ID {
				return store.ErrNotFound
			}
			deletedItem = id
			return nil
		},
		ListItemCollectionsFn: func(_ context.Context, userID, itemID uuid.UUID) ([]model.ItemCollectionMembership, error) {
			if userID != testutil.UserID {
				t.Errorf("expected the caller's user id, got %s", userID)
			}
			return nil, nil // nil must be rendered as []
		},
	}

	runRouteCases(t, []routeCase{
		{"update item type ok", "PUT", "/item-types/" + it.ID.String(), testutil.AdminToken(), `{"name":"Series"}`, mock, 200, nil},
		{"update item type conflict", "PUT", "/item-types/" + it.ID.String(), testutil.AdminToken(), `{"name":"dup"}`, mock, 409, nil},
		{"update item type 404", "PUT", "/item-types/" + uuid.New().String(), testutil.AdminToken(), `{"name":"x"}`, mock, 404, nil},
		{"update item type bad schema", "PUT", "/item-types/" + it.ID.String(), testutil.AdminToken(), `{"attributes_schema":{"type":"array"}}`, mock, 422, nil},
		{"update item type user forbidden", "PUT", "/item-types/" + it.ID.String(), testutil.UserToken(), `{"name":"x"}`, mock, 403, nil},

		{"update item ok (attributes validated)", "PUT", "/items/" + item.ID.String(), testutil.AdminToken(), `{"attributes":{"titre":"Better Call Saul"},"tags":["drame"]}`, mock, 200,
			func(t *testing.T, _ []byte) {
				if updatedTags == nil || len(*updatedTags) != 1 {
					t.Errorf("tags not forwarded: %v", updatedTags)
				}
			}},
		{"update item invalid attributes", "PUT", "/items/" + item.ID.String(), testutil.AdminToken(), `{"attributes":{"titre":123}}`, mock, 422, nil},
		{"update item tags only", "PUT", "/items/" + item.ID.String(), testutil.AdminToken(), `{"tags":[]}`, mock, 200, nil},
		{"update item 404", "PUT", "/items/" + uuid.New().String(), testutil.AdminToken(), `{"attributes":{"titre":"x"}}`, mock, 404, nil},
		{"update item user forbidden", "PUT", "/items/" + item.ID.String(), testutil.UserToken(), `{"attributes":{}}`, mock, 403, nil},
		{"update item invalid json", "PUT", "/items/" + item.ID.String(), testutil.AdminToken(), `{`, mock, 400, nil},

		{"delete item ok", "DELETE", "/items/" + item.ID.String(), testutil.AdminToken(), "", mock, 204,
			func(t *testing.T, _ []byte) {
				if deletedItem != item.ID {
					t.Error("store.DeleteItem not called with the item id")
				}
			}},
		{"delete item 404", "DELETE", "/items/" + uuid.New().String(), testutil.AdminToken(), "", mock, 404, nil},
		{"delete item user forbidden", "DELETE", "/items/" + item.ID.String(), testutil.UserToken(), "", mock, 403, nil},

		{"item collections ok (empty array)", "GET", "/items/" + item.ID.String() + "/collections", testutil.UserToken(), "", mock, 200,
			func(t *testing.T, b []byte) {
				if string(bytes.TrimSpace(b)) != "[]" {
					t.Errorf("expected [] for no membership, got %s", b)
				}
			}},
		{"item collections anonymous", "GET", "/items/" + item.ID.String() + "/collections", "", "", mock, 401, nil},
	})
}

// --- Relations (list / get / delete) ---

func TestRelations_Coverage(t *testing.T) {
	rel := sampleRelation()
	mock := &testutil.MockStore{
		ListItemRelationsFn: func(_ context.Context, itemID uuid.UUID, rtID *uuid.UUID) ([]model.ItemRelation, error) {
			if rtID != nil && *rtID != otherID {
				t.Errorf("relation_type_id filter not forwarded: %v", rtID)
			}
			return []model.ItemRelation{*rel}, nil
		},
		GetRelationFn: func(_ context.Context, id uuid.UUID) (*model.ItemRelation, error) {
			if id == someID {
				return rel, nil
			}
			return nil, store.ErrNotFound
		},
		DeleteRelationFn: func(_ context.Context, id uuid.UUID) error {
			if id == someID {
				return nil
			}
			return store.ErrNotFound
		},
	}

	runRouteCases(t, []routeCase{
		{"list item relations ok", "GET", "/items/" + uuid.New().String() + "/relations?relation_type_id=" + otherID.String(), testutil.UserToken(), "", mock, 200,
			func(t *testing.T, b []byte) {
				var rels []model.ItemRelation
				_ = json.Unmarshal(b, &rels)
				if len(rels) != 1 {
					t.Errorf("expected 1 relation, got %d", len(rels))
				}
			}},
		{"list item relations bad filter", "GET", "/items/" + uuid.New().String() + "/relations?relation_type_id=nope", testutil.UserToken(), "", mock, 400, nil},
		{"list item relations anonymous", "GET", "/items/" + uuid.New().String() + "/relations", "", "", mock, 401, nil},
		{"get relation ok", "GET", "/relations/" + someID.String(), testutil.UserToken(), "", mock, 200, nil},
		{"get relation 404", "GET", "/relations/" + uuid.New().String(), testutil.UserToken(), "", mock, 404, nil},
		{"delete relation ok", "DELETE", "/relations/" + someID.String(), testutil.AdminToken(), "", mock, 204, nil},
		{"delete relation 404", "DELETE", "/relations/" + uuid.New().String(), testutil.AdminToken(), "", mock, 404, nil},
		{"delete relation user forbidden", "DELETE", "/relations/" + someID.String(), testutil.UserToken(), "", mock, 403, nil},
	})
}

// --- Collections: update, list items, remove item (owner checks) ---

func TestCollections_OwnerChecks_Coverage(t *testing.T) {
	mine := &model.Collection{ID: someID, UserID: testutil.UserID, CollectionTypeID: uuid.New(), Name: "Ma collection"}
	theirs := &model.Collection{ID: otherID, UserID: testutil.AdminID, CollectionTypeID: uuid.New(), Name: "Pas à moi"}
	itemID := uuid.New()
	removed := false
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			switch id {
			case mine.ID:
				return mine, nil
			case theirs.ID:
				return theirs, nil
			}
			return nil, store.ErrNotFound
		},
		UpdateCollectionFn: func(_ context.Context, id uuid.UUID, in model.CollectionUpdate) (*model.Collection, error) {
			if in.Name != nil && *in.Name == "dup" {
				return nil, store.ErrConflict
			}
			out := *mine
			if in.Name != nil {
				out.Name = *in.Name
			}
			return &out, nil
		},
		ListCollectionItemsFn: func(_ context.Context, colID uuid.UUID, pg model.Pagination, itID *uuid.UUID, sort string) ([]model.CollectionItem, int, error) {
			if sort != "-created_at" {
				t.Errorf("sort not forwarded: %q", sort)
			}
			return []model.CollectionItem{{ID: uuid.New(), CollectionID: colID, ItemID: itemID, Attributes: emptyJSON}}, 1, nil
		},
		RemoveCollectionItemFn: func(_ context.Context, colID, itID uuid.UUID) error {
			if itID != itemID {
				return store.ErrNotFound
			}
			removed = true
			return nil
		},
	}

	runRouteCases(t, []routeCase{
		{"update own collection", "PUT", "/collections/" + mine.ID.String(), testutil.UserToken(), `{"name":"Renommée"}`, mock, 200,
			func(t *testing.T, b []byte) {
				var c model.Collection
				_ = json.Unmarshal(b, &c)
				if c.Name != "Renommée" {
					t.Errorf("name = %q", c.Name)
				}
			}},
		{"update other's collection → 404", "PUT", "/collections/" + theirs.ID.String(), testutil.UserToken(), `{"name":"x"}`, mock, 404, nil},
		{"update collection conflict", "PUT", "/collections/" + mine.ID.String(), testutil.UserToken(), `{"name":"dup"}`, mock, 409, nil},
		{"update unknown collection", "PUT", "/collections/" + uuid.New().String(), testutil.UserToken(), `{"name":"x"}`, mock, 404, nil},
		{"update collection anonymous", "PUT", "/collections/" + mine.ID.String(), "", `{"name":"x"}`, mock, 401, nil},

		{"list own collection items", "GET", "/collections/" + mine.ID.String() + "/items?sort=-created_at", testutil.UserToken(), "", mock, 200,
			func(t *testing.T, b []byte) {
				var res model.PaginatedResponse
				_ = json.Unmarshal(b, &res)
				if res.Total != 1 {
					t.Errorf("total = %d", res.Total)
				}
			}},
		{"list other's collection items → 404", "GET", "/collections/" + theirs.ID.String() + "/items", testutil.UserToken(), "", mock, 404, nil},
		{"list collection items bad filter", "GET", "/collections/" + mine.ID.String() + "/items?item_type_id=x", testutil.UserToken(), "", mock, 400, nil},

		{"remove item from own collection", "DELETE", "/collections/" + mine.ID.String() + "/items/" + itemID.String(), testutil.UserToken(), "", mock, 204,
			func(t *testing.T, _ []byte) {
				if !removed {
					t.Error("store.RemoveCollectionItem not called")
				}
			}},
		{"remove item from other's collection → 404", "DELETE", "/collections/" + theirs.ID.String() + "/items/" + itemID.String(), testutil.UserToken(), "", mock, 404, nil},
		{"remove unknown item", "DELETE", "/collections/" + mine.ID.String() + "/items/" + uuid.New().String(), testutil.UserToken(), "", mock, 404, nil},
	})
}

// --- Tags ---

func TestListTags_Coverage(t *testing.T) {
	mock := &testutil.MockStore{
		ListTagsFn: func(_ context.Context, search string) ([]model.Tag, error) {
			if search != "dr" {
				t.Errorf("search not forwarded: %q", search)
			}
			return []model.Tag{{ID: uuid.New(), Name: "drame"}}, nil
		},
	}
	runRouteCases(t, []routeCase{
		{"list tags ok", "GET", "/tags?search=dr", testutil.UserToken(), "", mock, 200,
			func(t *testing.T, b []byte) {
				var tags []model.Tag
				_ = json.Unmarshal(b, &tags)
				if len(tags) != 1 || tags[0].Name != "drame" {
					t.Errorf("unexpected tags: %s", b)
				}
			}},
		{"list tags anonymous", "GET", "/tags", "", "", mock, 401, nil},
	})
}
