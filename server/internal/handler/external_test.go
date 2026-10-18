package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/connectors/connectorstest"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/testutil"
)

// externalMock is a store holding the seed types, wired to fake sources.
func externalMock(t *testing.T) (*testutil.MockStore, *connectorstest.Sources) {
	its, rts := connectorstest.Domain(t)
	mock := &testutil.MockStore{
		ListItemTypesFn: func(context.Context, model.Pagination) ([]model.ItemType, int, error) {
			return its, len(its), nil
		},
		ListRelationTypesFn: func(context.Context, model.Pagination, *uuid.UUID, *uuid.UUID) ([]model.RelationType, int, error) {
			return rts, len(rts), nil
		},
	}
	return mock, connectorstest.NewSources(t)
}

func callExternal(t *testing.T, mock *testutil.MockStore, src *connectorstest.Sources, method, path, token, body string) (*http.Response, string) {
	t.Helper()
	h := testutil.NewTestHandler(mock)
	h.Connectors = []external.Connector{src.Connector(mock), src.NovelsConnector(mock), src.MusicConnector(mock), src.FilmsConnector(mock), src.SeriesConnector(mock), src.GamesConnector(mock)}
	srv := testutil.NewTestServerFromHandler(h)
	defer srv.Close()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, srv.URL+path, r)
	req.Header = testutil.AuthHeader(token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

// Search and preview: any member (P4-3b, members propose); direct import:
// admins and contributors.
func TestExternal_Access(t *testing.T) {
	mock, src := externalMock(t)
	for _, c := range []struct {
		method, path, body string
		member             int
	}{
		{"GET", "/api/v1/external/bd-manga/works?q=dragon+ball", "", http.StatusOK},
		{"GET", "/api/v1/external/bd-manga/editions?q=dragon+ball", "", http.StatusOK},
		{"POST", "/api/v1/external/bd-manga/preview", `{"work":"Q15051307"}`, http.StatusOK},
		{"POST", "/api/v1/external/bd-manga/import", `{"work":"Q15051307","selected":["wikidata:Q15051307"]}`, http.StatusForbidden},
	} {
		if resp, _ := callExternal(t, mock, src, c.method, c.path, "", c.body); resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s anonymous: %d", c.method, c.path, resp.StatusCode)
		}
		if resp, _ := callExternal(t, mock, src, c.method, c.path, testutil.UserToken(), c.body); resp.StatusCode != c.member {
			t.Errorf("%s %s member: %d, want %d", c.method, c.path, resp.StatusCode, c.member)
		}
	}
}

func TestExternal_Search(t *testing.T) {
	mock, src := externalMock(t)
	admin := testutil.AdminToken()

	resp, body := callExternal(t, mock, src, "GET", "/api/v1/external/bd-manga/works?q=dragon+ball", admin, "")
	var works []model.ExternalWorkHit
	if resp.StatusCode != http.StatusOK || json.Unmarshal([]byte(body), &works) != nil || len(works) == 0 || works[0].WikidataID != "Q15051307" {
		t.Errorf("works: %d %s", resp.StatusCode, body)
	}
	resp, body = callExternal(t, mock, src, "GET", "/api/v1/external/bd-manga/editions?isbn=2-87695-205-X", admin, "")
	var editions []model.ExternalEditionHit
	if resp.StatusCode != http.StatusOK || json.Unmarshal([]byte(body), &editions) != nil || len(editions) != 2 {
		t.Errorf("editions by ISBN-10: %d %s", resp.StatusCode, body)
	}

	for _, path := range []string{
		"/api/v1/external/bd-manga/works?q=x",
		"/api/v1/external/bd-manga/works",
		"/api/v1/external/bd-manga/editions",
		"/api/v1/external/bd-manga/editions?q=naruto&isbn=9782876952058",
		"/api/v1/external/bd-manga/editions?isbn=9782876952059",
	} {
		if resp, _ := callExternal(t, mock, src, "GET", path, admin, ""); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", path, resp.StatusCode)
		}
	}
}

func TestExternal_Errors(t *testing.T) {
	mock, src := externalMock(t)
	admin := testutil.AdminToken()

	resp, body := callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/preview", admin, `{"work":"Q999"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown work: %d %s", resp.StatusCode, body)
	}
	resp, _ = callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/preview", admin, `{"editions":["ark:/12148/cb34338439x"]}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad ARK: %d", resp.StatusCode)
	}
	resp, _ = callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/preview", admin, `{`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad JSON: %d", resp.StatusCode)
	}

	src.Down.Store(true)
	resp, body = callExternal(t, mock, src, "GET", "/api/v1/external/bd-manga/works?q=naruto", admin, "")
	if resp.StatusCode != http.StatusBadGateway || !strings.Contains(body, `"external_unavailable"`) {
		t.Errorf("source down: %d %s", resp.StatusCode, body)
	}
	src.Down.Store(false)

	// A catalogue without the BD / Manga types.
	mock.ListItemTypesFn = func(context.Context, model.Pagination) ([]model.ItemType, int, error) {
		return []model.ItemType{*testutil.SampleItemType()}, 1, nil
	}
	resp, body = callExternal(t, mock, src, "GET", "/api/v1/external/bd-manga/works?q=naruto", admin, "")
	if resp.StatusCode != http.StatusConflict || !strings.Contains(body, `"connector_unavailable"`) || !strings.Contains(body, `"missing":["Manga",`) {
		t.Errorf("types missing: %d %s", resp.StatusCode, body)
	}
}

func TestExternal_PreviewAndImport(t *testing.T) {
	mock, src := externalMock(t)
	admin := testutil.AdminToken()
	req := `{"work":"Q15051307","editions":["ark:/12148/cb343384394"]}`

	resp, body := callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/preview", admin, req)
	var plan model.ExternalPlan
	if resp.StatusCode != http.StatusOK || json.Unmarshal([]byte(body), &plan) != nil {
		t.Fatalf("preview: %d %s", resp.StatusCode, body)
	}
	var selected []string
	for _, n := range plan.Nodes {
		if n.Selected {
			selected = append(selected, n.Key)
		}
	}

	var got *model.CatalogExport
	var gotDryRun bool
	var failing bool
	mock.ImportCatalogFn = func(_ context.Context, cat *model.CatalogExport, dryRun bool) (*model.ImportReport, error) {
		got, gotDryRun = cat, dryRun
		rep := model.NewImportReport(dryRun)
		rep.Counts[model.EntityItems].Created = len(cat.Items)
		if failing {
			rep.Fail(model.EntityItems, 0, nil, "attributes: invalid")
		} else if !dryRun {
			rep.Applied = true
		}
		return rep, nil
	}
	sel, _ := json.Marshal(selected)
	importBody := `{"work":"Q15051307","editions":["ark:/12148/cb343384394"],"selected":` + string(sel) + `}`

	resp, body = callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/import?dry_run=true", admin, importBody)
	var rep model.ExternalImportReport
	if resp.StatusCode != http.StatusOK || !gotDryRun || json.Unmarshal([]byte(body), &rep) != nil {
		t.Fatalf("dry run: %d %s", resp.StatusCode, body)
	}
	if len(got.Items) != len(selected) || rep.Counts[model.EntityItems].Created != len(selected) || len(rep.ItemIDs) != len(selected) {
		t.Errorf("dry run: %d items for %d selected, ids %d", len(got.Items), len(selected), len(rep.ItemIDs))
	}
	if !slices.ContainsFunc(got.Relations, func(r model.ExportRelation) bool {
		return r.SourceItemID == rep.ItemIDs["wikidata:Q15051307"] && r.TargetItemID == rep.ItemIDs["wikidata:Q208582"]
	}) {
		t.Error("created_by relation missing from the import file")
	}

	resp, body = callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/import", admin, importBody)
	if resp.StatusCode != http.StatusOK || gotDryRun || !strings.Contains(body, `"applied":true`) {
		t.Errorf("import: %d %s", resp.StatusCode, body)
	}
	failing = true
	if resp, body = callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/import", admin, importBody); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("failing import: %d %s", resp.StatusCode, body)
	}

	for _, bad := range []string{
		`{"work":"Q15051307"}`,
		`{"work":"Q15051307","selected":["wikidata:Q1"]}`,
	} {
		if resp, _ := callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/import", admin, bad); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", bad, resp.StatusCode)
		}
	}
	if resp, _ := callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/import?dry_run=maybe", admin, importBody); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad dry_run: %d", resp.StatusCode)
	}
}

func TestExternal_Connectors(t *testing.T) {
	mock, src := externalMock(t)
	resp, body := callExternal(t, mock, src, "GET", "/api/v1/external/connectors", testutil.AdminToken(), "")
	var list []model.ConnectorInfo
	if resp.StatusCode != http.StatusOK || json.Unmarshal([]byte(body), &list) != nil ||
		len(list) != 6 || list[5].ID != "games" || list[5].Domain != "games" || list[4].ID != "series" || list[4].Domain != "tv" || list[3].ID != "films" || list[3].Domain != "video" || list[3].EditionCode != "barcode" || list[0].ID != "bd-manga" || list[0].Domain != "books" || list[0].EditionCode != "isbn" ||
		list[2].ID != "music" || list[2].Domain != "music" || list[2].EditionCode != "barcode" || !list[2].EditionsByWork {
		t.Errorf("connectors: %d %s", resp.StatusCode, body)
	}
	if resp, _ := callExternal(t, mock, src, "GET", "/api/v1/external/nope/works?q=naruto", testutil.AdminToken(), ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown connector: %d", resp.StatusCode)
	}
	if resp, _ := callExternal(t, mock, src, "GET", "/api/v1/external/connectors", testutil.UserToken(), ""); resp.StatusCode != http.StatusOK {
		t.Errorf("member: %d", resp.StatusCode)
	}
}

func TestExternal_MusicEditions(t *testing.T) {
	mock, src := externalMock(t)
	mock.FindItemsByAttributeFn = func(context.Context, uuid.UUID, []string, []string, bool) ([]model.Item, error) {
		return nil, nil
	}
	admin := testutil.AdminToken()
	resp, body := callExternal(t, mock, src, "GET", "/api/v1/external/music/editions?barcode=6%2034904%20078126", admin, "")
	var hits []model.ExternalEditionHit
	if resp.StatusCode != http.StatusOK || json.Unmarshal([]byte(body), &hits) != nil || len(hits) != 3 || hits[0].Barcode != connectorstest.OKComputerUPC {
		t.Errorf("by barcode: %d %s", resp.StatusCode, body)
	}
	for _, bad := range []string{
		"/api/v1/external/music/editions?barcode=634904078127",               // check digit
		"/api/v1/external/music/editions?work=Q202996",                       // not a release group
		"/api/v1/external/music/editions?isbn=9782876952058",                 // books only
		"/api/v1/external/bd-manga/editions?barcode=634904078126",            // music only
		"/api/v1/external/novels/editions?work=" + connectorstest.OKComputer, // not a Wikidata work
	} {
		if resp, _ := callExternal(t, mock, src, "GET", bad, admin, ""); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", bad, resp.StatusCode)
		}
	}
}

// A member turns a selection into a submitted proposal: nothing is imported,
// the payload is checked by a dry run.
func TestExternal_Propose(t *testing.T) {
	mock, src := externalMock(t)
	var created *model.ProposalInput
	var dryRuns, writes int
	mock.ImportCatalogFn = func(_ context.Context, _ *model.CatalogExport, dryRun bool) (*model.ImportReport, error) {
		if dryRun {
			dryRuns++
		} else {
			writes++
		}
		return model.NewImportReport(dryRun), nil
	}
	mock.CreateProposalFn = func(_ context.Context, author uuid.UUID, in model.ProposalInput) (*model.Proposal, error) {
		created = &in
		return &model.Proposal{ID: uuid.New(), Kind: in.Kind, Status: model.ProposalDraft, Title: in.Title,
			Payload: in.Payload, AuthorID: author}, nil
	}
	mock.SubmitProposalFn = func(_ context.Context, id uuid.UUID, _ *model.ProposalBase) (*model.Proposal, error) {
		return &model.Proposal{ID: id, Status: model.ProposalPending}, nil
	}
	resp, body := callExternal(t, mock, src, "POST", "/api/v1/external/bd-manga/propose", testutil.UserToken(),
		`{"work":"Q15051307","selected":["wikidata:Q15051307"],"title":"Dragon Ball"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("propose: %d %s", resp.StatusCode, body)
	}
	if created == nil || created.Title != "Dragon Ball" || created.Payload == nil || len(created.Payload.Items) == 0 {
		t.Fatalf("proposal: %+v", created)
	}
	if dryRuns != 1 || writes != 0 {
		t.Errorf("dry runs %d, writes %d", dryRuns, writes)
	}
}
