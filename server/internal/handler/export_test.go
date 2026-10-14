package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

func call(t *testing.T, mock *testutil.MockStore, method, path, token string, body io.Reader) (*http.Response, string) {
	t.Helper()
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	req, _ := http.NewRequest(method, srv.URL+path, body)
	req.Header = testutil.AuthHeader(token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestExportCatalog(t *testing.T) {
	mock := &testutil.MockStore{}
	if resp, _ := call(t, mock, "GET", "/api/v1/export", testutil.UserToken(), nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user: expected 403, got %d", resp.StatusCode)
	}
	resp, body := call(t, mock, "GET", "/api/v1/export", testutil.AdminToken(), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, `filename="collectionnard-catalog-`) {
		t.Errorf("Content-Disposition: %q", cd)
	}
	var cat model.CatalogExport
	if err := json.Unmarshal([]byte(body), &cat); err != nil || cat.Format != model.CatalogFormat {
		t.Errorf("body: %v %+v", err, cat)
	}
}

func TestImportCatalog(t *testing.T) {
	valid := `{"format":"collectionnard.catalog","version":1,"item_types":[],"collection_types":[],"relation_types":[],"items":[],"relations":[]}`
	var gotDryRun bool
	var failing bool
	mock := &testutil.MockStore{
		ImportCatalogFn: func(_ context.Context, _ *model.CatalogExport, dryRun bool) (*model.ImportReport, error) {
			gotDryRun = dryRun
			rep := model.NewImportReport(dryRun)
			if failing {
				id := uuid.New()
				rep.Fail(model.EntityItems, 3, &id, "attributes: /: missing properties: 'title'")
			} else if !dryRun {
				rep.Applied = true
			}
			return rep, nil
		},
	}
	admin := testutil.AdminToken()

	if resp, _ := call(t, mock, "POST", "/api/v1/import", testutil.UserToken(), strings.NewReader(valid)); resp.StatusCode != http.StatusForbidden {
		t.Errorf("user: expected 403, got %d", resp.StatusCode)
	}
	resp, body := call(t, mock, "POST", "/api/v1/import?dry_run=true", admin, strings.NewReader(valid))
	if resp.StatusCode != http.StatusOK || !gotDryRun || !strings.Contains(body, `"applied":false`) {
		t.Errorf("dry run: %d %v %s", resp.StatusCode, gotDryRun, body)
	}
	resp, body = call(t, mock, "POST", "/api/v1/import", admin, strings.NewReader(valid))
	if resp.StatusCode != http.StatusOK || gotDryRun || !strings.Contains(body, `"applied":true`) {
		t.Errorf("apply: %d %v %s", resp.StatusCode, gotDryRun, body)
	}
	failing = true
	resp, body = call(t, mock, "POST", "/api/v1/import", admin, strings.NewReader(valid))
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, `"index":3`) {
		t.Errorf("errors: expected 422 with the report, got %d %s", resp.StatusCode, body)
	}

	cases := map[string]struct {
		path string
		body io.Reader
		want int
	}{
		"bad dry_run": {"/api/v1/import?dry_run=maybe", strings.NewReader(valid), http.StatusBadRequest},
		"bad JSON":    {"/api/v1/import", strings.NewReader("{"), http.StatusBadRequest},
		"too large":   {"/api/v1/import", bytes.NewReader(bytes.Repeat([]byte(" "), 1<<20+1)), http.StatusRequestEntityTooLarge},
	}
	for name, c := range cases {
		if resp, _ := call(t, mock, "POST", c.path, admin, c.body); resp.StatusCode != c.want {
			t.Errorf("%s: expected %d, got %d", name, c.want, resp.StatusCode)
		}
	}
}

func TestExportItemsCSV(t *testing.T) {
	it := testutil.SampleItemType()
	it.AttributesSchema = json.RawMessage(`{"type":"object","x-label":"{titre}","properties":{"titre":{"type":"string"},"annee":{"type":"integer"}}}`)
	item := *testutil.SampleItem()
	item.Attributes = json.RawMessage(`{"titre":"=Breaking; Bad","annee":2008}`)
	item.Tags = []model.Tag{{Name: "drama"}, {Name: "crime"}}
	var gotFilters store.ItemFilters
	mock := &testutil.MockStore{
		GetItemTypeFn: func(_ context.Context, id uuid.UUID) (*model.ItemType, error) {
			if id != it.ID {
				return nil, store.ErrNotFound
			}
			return it, nil
		},
		ListItemsFn: func(_ context.Context, _ model.Pagination, f store.ItemFilters) ([]model.Item, int, error) {
			gotFilters = f
			return []model.Item{item}, 1, nil
		},
	}
	user := testutil.UserToken()
	resp, body := call(t, mock, "GET", "/api/v1/items/export?item_type_id="+it.ID.String()+"&delimiter=semicolon&search=bad", user, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("Content-Type %q", ct)
	}
	// No x-order: attribute columns sorted by key; cells containing the delimiter are quoted.
	want := "\xef\xbb\xbfid;label;tags;annee;titre\r\n" +
		item.ID.String() + ";\"'=Breaking; Bad\";\"drama; crime\";2008;\"'=Breaking; Bad\"\r\n"
	if body != want {
		t.Errorf("body:\n%q\nwant\n%q", body, want)
	}
	if gotFilters.Search != "bad" || gotFilters.ItemTypeID == nil {
		t.Errorf("filters not forwarded: %+v", gotFilters)
	}

	for name, path := range map[string]string{
		"no type":       "/api/v1/items/export",
		"unknown type":  "/api/v1/items/export?item_type_id=" + uuid.NewString(),
		"bad delimiter": "/api/v1/items/export?item_type_id=" + it.ID.String() + "&delimiter=pipe",
		"bad lang":      "/api/v1/items/export?item_type_id=" + it.ID.String() + "&lang=../x",
		"bad filter":    "/api/v1/items/export?item_type_id=" + it.ID.String() + "&attr.nope=1",
	} {
		if resp, _ := call(t, mock, "GET", path, user, nil); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, resp.StatusCode)
		}
	}
}

func TestExportCollection(t *testing.T) {
	ct := &model.CollectionType{ID: uuid.New(), Name: "Ownership",
		AttributesSchema: json.RawMessage(`{"type":"object","properties":{"condition":{"type":"string"}}}`)}
	col := &model.Collection{ID: uuid.New(), UserID: testutil.UserID, Name: "Mes DVD", CollectionType: ct}
	item := testutil.SampleItem()
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			if id != col.ID {
				return nil, store.ErrNotFound
			}
			return col, nil
		},
		ListCollectionItemsFn: func(context.Context, uuid.UUID, model.Pagination, *uuid.UUID, string) ([]model.CollectionItem, int, error) {
			return []model.CollectionItem{{ItemID: item.ID, Item: item, Attributes: json.RawMessage(`{"condition":"good"}`)}}, 1, nil
		},
	}
	user := testutil.UserToken()
	path := "/api/v1/collections/" + col.ID.String() + "/export"

	resp, body := call(t, mock, "GET", path, user, nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("Content-Disposition"), "collection-Mes-DVD-") {
		t.Fatalf("json: %d %s", resp.StatusCode, resp.Header.Get("Content-Disposition"))
	}
	var exp model.CollectionExport
	if err := json.Unmarshal([]byte(body), &exp); err != nil || exp.Format != model.CollectionFormat ||
		len(exp.Items) != 1 || exp.Items[0].ItemType != "Serie" || exp.Collection.CollectionType.Name != "Ownership" {
		t.Errorf("json body: %v %+v", err, exp)
	}

	resp, body = call(t, mock, "GET", path+"?format=csv", user, nil)
	lines := strings.Split(strings.TrimPrefix(body, "\xef\xbb\xbf"), "\r\n")
	if resp.StatusCode != http.StatusOK || lines[0] != "item_id,item_type,label,condition,added_at" ||
		!strings.HasPrefix(lines[1], item.ID.String()+",Serie,") || !strings.Contains(lines[1], ",good,") {
		t.Errorf("csv: %d %q", resp.StatusCode, lines)
	}

	if resp, _ := call(t, mock, "GET", path+"?format=xml", user, nil); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad format: expected 400, got %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "GET", path, testutil.AdminToken(), nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("other user's collection: expected 404, got %d", resp.StatusCode)
	}
}

func TestImportCollection(t *testing.T) {
	valid := `{"format":"collectionnard.collection","version":1,"collection":{"name":"Mes DVD"},"items":[]}`
	var gotUser uuid.UUID
	var gotName string
	mock := &testutil.MockStore{
		ImportCollectionFn: func(_ context.Context, userID uuid.UUID, exp *model.CollectionExport, dryRun bool) (*model.ImportReport, error) {
			gotUser, gotName = userID, exp.Collection.Name
			rep := model.NewCollectionImportReport(dryRun)
			if exp.Collection.Name == "bad" {
				rep.Fail(model.EntityItems, 0, nil, "unknown item_id (not in the catalogue)")
			}
			return rep, nil
		},
	}
	user := testutil.UserToken()
	resp, _ := call(t, mock, "POST", "/api/v1/collections/import?dry_run=true", user, strings.NewReader(valid))
	if resp.StatusCode != http.StatusOK || gotUser != testutil.UserID || gotName != "Mes DVD" {
		t.Errorf("import: %d user %v name %q", resp.StatusCode, gotUser, gotName)
	}
	bad := strings.Replace(valid, "Mes DVD", "bad", 1)
	if resp, _ := call(t, mock, "POST", "/api/v1/collections/import", user, strings.NewReader(bad)); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("errors: expected 422, got %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "POST", "/api/v1/collections/import", user, strings.NewReader("[")); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad JSON: expected 400, got %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "POST", "/api/v1/collections/import", "", strings.NewReader(valid)); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous: expected 401, got %d", resp.StatusCode)
	}
}
