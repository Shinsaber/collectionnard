package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

func TestParsePagination_Clamped(t *testing.T) {
	cases := map[string]int{
		"":              20,
		"?per_page=50":  50,
		"?per_page=100": 100,
		"?per_page=101": 100,
		"?per_page=0":   20,
		"?per_page=-5":  20,
		"?per_page=abc": 20,
	}
	for query, want := range cases {
		t.Run(query, func(t *testing.T) {
			var got int
			mock := &testutil.MockStore{
				ListItemsFn: func(_ context.Context, pg model.Pagination, _ store.ItemFilters) ([]model.Item, int, error) {
					got = pg.PerPage
					return nil, 0, nil
				},
			}
			srv := testutil.NewTestServer(mock)
			defer srv.Close()
			req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items"+query, nil)
			req.Header = testutil.AuthHeader(testutil.UserToken())
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if got != want {
				t.Errorf("per_page %q: got %d, want %d", query, got, want)
			}
		})
	}
}

func TestListItems_InvalidFilterUUID(t *testing.T) {
	srv := testutil.NewTestServer(&testutil.MockStore{})
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items?item_type_id=nope", nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListItems_SearchTooLong(t *testing.T) {
	srv := testutil.NewTestServer(&testutil.MockStore{})
	defer srv.Close()
	long := bytes.Repeat([]byte("a"), 201)
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items?search="+string(long), nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateItemType_InvalidSchema(t *testing.T) {
	created := false
	mock := &testutil.MockStore{
		CreateItemTypeFn: func(_ context.Context, in model.ItemTypeCreate) (*model.ItemType, error) {
			created = true
			return &model.ItemType{ID: uuid.New(), Name: in.Name, AttributesSchema: in.AttributesSchema}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	post := func(schema string) *http.Response {
		body := []byte(`{"name":"Test","attributes_schema":` + schema + `}`)
		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/item-types", bytes.NewReader(body))
		req.Header = testutil.AuthHeader(testutil.AdminToken())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	for _, schema := range []string{`{"type":"objet"}`, `{"properties":{}}`, `{"type":"object","properties":{"a":{"$ref":"http://x/y"}}}`} {
		resp := post(schema)
		var apiErr model.APIError
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("schema %s: expected 422, got %d", schema, resp.StatusCode)
		}
		if apiErr.Error != "validation_error" {
			t.Errorf("schema %s: expected validation_error, got %q", schema, apiErr.Error)
		}
	}
	if created {
		t.Error("store must not be called with an invalid schema")
	}

	resp := post(`{"type":"object","properties":{"title":{"type":"string"}}}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("valid schema: expected 201, got %d", resp.StatusCode)
	}
	empty := post(`{}`)
	empty.Body.Close()
	if empty.StatusCode != http.StatusCreated {
		t.Errorf("empty schema: expected 201, got %d", empty.StatusCode)
	}
}

func TestUpdateCollectionType_InvalidSchema(t *testing.T) {
	srv := testutil.NewTestServer(&testutil.MockStore{})
	defer srv.Close()
	body := []byte(`{"attributes_schema":{"type":"string"}}`)
	req, _ := http.NewRequest("PUT", srv.URL+"/api/v1/collection-types/"+uuid.New().String(), bytes.NewReader(body))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestListItems_InvalidSort(t *testing.T) {
	srv := testutil.NewTestServer(&testutil.MockStore{})
	defer srv.Close()
	for _, q := range []string{"?sort=attributes.titre", "?sort=-name", "?sort=created_at,id"} {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items"+q, nil)
		req.Header = testutil.AuthHeader(testutil.UserToken())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", q, resp.StatusCode)
		}
	}
	for _, q := range []string{"", "?sort=created_at", "?sort=-updated_at"} {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items"+q, nil)
		req.Header = testutil.AuthHeader(testutil.UserToken())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", q, resp.StatusCode)
		}
	}
}
