package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/itemquery"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

// searchType is an ItemType whose schema covers every filterable kind.
func searchType() *model.ItemType {
	it := testutil.SampleItemType()
	it.AttributesSchema = json.RawMessage(`{"type":"object","properties":{
		"title":{"type":"string"},
		"kind":{"type":"string","enum":["tv","web"]},
		"start_year":{"type":"integer"},
		"synopsis":{"type":"object","x-widget":"i18n-text"}
	}}`)
	return it
}

// listItems calls GET /items with query and returns the status and the filters
// received by the store (zero value when the store was not reached).
func listItems(t *testing.T, query url.Values) (int, store.ItemFilters) {
	t.Helper()
	var got store.ItemFilters
	mock := &testutil.MockStore{
		GetItemTypeFn: func(_ context.Context, id uuid.UUID) (*model.ItemType, error) {
			if id != searchType().ID {
				return nil, store.ErrNotFound
			}
			return searchType(), nil
		},
		ListItemsFn: func(_ context.Context, _ model.Pagination, f store.ItemFilters) ([]model.Item, int, error) {
			got = f
			return []model.Item{}, 0, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items?"+query.Encode(), nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode, got
}

func TestListItems_AttributeFiltersAndSort(t *testing.T) {
	typeID := searchType().ID.String()
	related := uuid.New()
	relType := uuid.New()
	status, f := listItems(t, url.Values{
		"item_type_id":           {typeID},
		"attr.kind":              {"tv", "web"},
		"attr.start_year.gte":    {"2008"},
		"attr.synopsis.contains": {"chimie"},
		"sort":                   {"-attr.start_year"},
		"related_to":             {related.String()},
		"relation_type_id":       {relType.String()},
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if len(f.Attrs) != 3 || f.Attrs[0].Key != "kind" || len(f.Attrs[0].Values) != 2 ||
		f.Attrs[1].Op != itemquery.OpGte || f.Attrs[1].Values[0] != int64(2008) {
		t.Errorf("unexpected attribute filters: %+v", f.Attrs)
	}
	if f.SortAttr == nil || *f.SortAttr != (itemquery.Sort{Key: "start_year", Numeric: true, Desc: true}) {
		t.Errorf("unexpected sort: %+v", f.SortAttr)
	}
	if f.RelatedTo == nil || *f.RelatedTo != related || f.RelationTypeID == nil || *f.RelationTypeID != relType {
		t.Errorf("unexpected relation filter: %v %v", f.RelatedTo, f.RelationTypeID)
	}
}

func TestListItems_RelevanceSortWithoutType(t *testing.T) {
	status, f := listItems(t, url.Values{"search": {"breaking"}, "sort": {"-relevance"}})
	if status != http.StatusOK || f.Sort != "-relevance" || f.Search != "breaking" {
		t.Fatalf("got %d %+v", status, f)
	}
}

func TestListItems_SearchErrors(t *testing.T) {
	typeID := searchType().ID.String()
	cases := map[string]url.Values{
		"attr without type":          {"attr.kind": {"tv"}},
		"attr sort without type":     {"sort": {"attr.title"}},
		"unknown type":               {"item_type_id": {uuid.NewString()}, "attr.kind": {"tv"}},
		"unknown attribute":          {"item_type_id": {typeID}, "attr.nope": {"x"}},
		"bad operator":               {"item_type_id": {typeID}, "attr.title.gte": {"x"}},
		"bad integer":                {"item_type_id": {typeID}, "attr.start_year": {"abc"}},
		"unsortable attribute":       {"item_type_id": {typeID}, "sort": {"attr.synopsis"}},
		"bad sort with attr":         {"item_type_id": {typeID}, "attr.kind": {"tv"}, "sort": {"title"}},
		"bad sort":                   {"sort": {"title"}},
		"relation type without item": {"relation_type_id": {uuid.NewString()}},
		"bad related_to":             {"related_to": {"nope"}},
	}
	for name, q := range cases {
		if status, _ := listItems(t, q); status != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, status)
		}
	}
}
