package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/shinsaber/colec/internal/testutil"
)

func TestHealthz(t *testing.T) {
	cases := []struct {
		name   string
		ping   error
		status int
		body   string
	}{
		{"database up", nil, http.StatusOK, "ok"},
		{"database down", errors.New("connection refused"), http.StatusServiceUnavailable, "degraded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &testutil.MockStore{PingFn: func(context.Context) error { return tc.ping }}
			srv := testutil.NewTestServer(mock)
			defer srv.Close()

			resp, err := http.Get(srv.URL + "/api/v1/healthz")
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.status {
				t.Fatalf("expected %d, got %d", tc.status, resp.StatusCode)
			}
			var body map[string]string
			_ = json.NewDecoder(resp.Body).Decode(&body)
			if body["status"] != tc.body {
				t.Errorf("expected status %q, got %q", tc.body, body["status"])
			}
		})
	}
}
