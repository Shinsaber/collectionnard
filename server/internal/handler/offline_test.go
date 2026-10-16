package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/testutil"
)

func TestGetOfflineSnapshot(t *testing.T) {
	var gotUser uuid.UUID
	mock := &testutil.MockStore{
		GetOfflineSnapshotFn: func(_ context.Context, userID uuid.UUID) (*model.OfflineSnapshot, error) {
			gotUser = userID
			return &model.OfflineSnapshot{User: model.User{ID: userID}, Items: []model.Item{}}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/me/offline", nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	var snap model.OfflineSnapshot
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&snap) != nil {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if gotUser != testutil.UserID || snap.User.ID != testutil.UserID {
		t.Errorf("snapshot for %s, want the authenticated user", gotUser)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "private, no-store" {
		t.Errorf("Cache-Control = %q", cc)
	}

	anon, _ := http.Get(srv.URL + "/api/v1/me/offline")
	if anon.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous: %d, want 401", anon.StatusCode)
	}
	_ = anon.Body.Close()
}
