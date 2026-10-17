package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/testutil"
)

func TestDomains(t *testing.T) {
	var gotDry []bool
	var refreshed []string
	mock := &testutil.MockStore{
		ListDomainsFn: func(context.Context) ([]model.DomainInfo, error) {
			return []model.DomainInfo{{ID: "books", Status: model.DomainNotInstalled}}, nil
		},
		InstallDomainFn: func(_ context.Context, pkg *model.DomainPackage, dryRun bool) (*model.DomainInstallReport, error) {
			gotDry = append(gotDry, dryRun)
			rep := &model.DomainInstallReport{Domain: pkg.ID, DryRun: dryRun, Applied: !dryRun, ToVersion: pkg.PackageVersion}
			if pkg.ID == "tv" {
				rep.Applied = false
				rep.Errors = []model.ImportError{{Entity: model.EntityItemTypes, Message: "name already used"}}
			}
			return rep, nil
		},
		RefreshInstalledDomainsFn: func(_ context.Context, except string) error {
			refreshed = append(refreshed, except)
			return nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	call := func(method, path, token string) (*http.Response, model.DomainInstallReport) {
		req, _ := http.NewRequest(method, srv.URL+path, nil)
		req.Header = testutil.AuthHeader(token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var rep model.DomainInstallReport
		_ = json.NewDecoder(resp.Body).Decode(&rep)
		return resp, rep
	}

	if resp, _ := call("GET", "/api/v1/domains", testutil.UserToken()); resp.StatusCode != http.StatusForbidden {
		t.Errorf("user list: %d", resp.StatusCode)
	}
	if resp, _ := call("GET", "/api/v1/domains", testutil.AdminToken()); resp.StatusCode != http.StatusOK {
		t.Errorf("admin list: %d", resp.StatusCode)
	}
	if resp, rep := call("POST", "/api/v1/domains/books/install?dry_run=true", testutil.AdminToken()); resp.StatusCode != http.StatusOK || !rep.DryRun {
		t.Errorf("dry run: %d %+v", resp.StatusCode, rep)
	}
	if len(refreshed) != 0 {
		t.Errorf("a dry run must not refresh the other domains: %v", refreshed)
	}
	if resp, rep := call("POST", "/api/v1/domains/books/install", testutil.AdminToken()); resp.StatusCode != http.StatusOK || !rep.Applied {
		t.Errorf("install: %d %+v", resp.StatusCode, rep)
	}
	if len(refreshed) != 1 || refreshed[0] != "books" {
		t.Errorf("refresh after install: %v", refreshed)
	}
	if resp, _ := call("POST", "/api/v1/domains/tv/install", testutil.AdminToken()); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("install with errors: %d", resp.StatusCode)
	}
	if resp, _ := call("POST", "/api/v1/domains/nope/install", testutil.AdminToken()); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown domain: %d", resp.StatusCode)
	}
	if len(gotDry) != 3 || !gotDry[0] || gotDry[1] {
		t.Errorf("dry_run passed to the store: %v", gotDry)
	}
}
