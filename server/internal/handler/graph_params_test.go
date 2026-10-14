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

func graphMock(t *testing.T, wantDepth, wantMaxNodes int, wantRT int) (*testutil.MockStore, *int) {
	t.Helper()
	calls := 0
	return &testutil.MockStore{
		GetItemGraphFn: func(_ context.Context, id uuid.UUID, depth, maxNodes int, rtIDs []uuid.UUID) (*model.GraphResponse, error) {
			calls++
			if depth != wantDepth {
				t.Errorf("depth = %d, want %d", depth, wantDepth)
			}
			if maxNodes != wantMaxNodes {
				t.Errorf("maxNodes = %d, want %d", maxNodes, wantMaxNodes)
			}
			if len(rtIDs) != wantRT {
				t.Errorf("len(rtIDs) = %d, want %d", len(rtIDs), wantRT)
			}
			return &model.GraphResponse{Root: model.GraphNode{ID: id}, Truncated: true}, nil
		},
	}, &calls
}

func getGraph(t *testing.T, srv string, token, query string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", srv+"/api/v1/items/"+uuid.New().String()+"/graph"+query, nil)
	req.Header = testutil.AuthHeader(token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return resp
}

func TestGetItemGraph_DefaultsAndTruncated(t *testing.T) {
	mock, calls := graphMock(t, 3, 500, 0)
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	resp := getGraph(t, srv.URL, testutil.UserToken(), "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var g model.GraphResponse
	_ = json.NewDecoder(resp.Body).Decode(&g)
	if !g.Truncated {
		t.Error("truncated flag must be forwarded to the client")
	}
	if *calls != 1 {
		t.Errorf("store called %d times", *calls)
	}
}

func TestGetItemGraph_DepthLimits(t *testing.T) {
	cases := []struct {
		name   string
		token  string
		query  string
		status int
	}{
		{"user depth 5 ok", testutil.UserToken(), "?max_depth=5", http.StatusOK},
		{"user depth 6 refused", testutil.UserToken(), "?max_depth=6", http.StatusBadRequest},
		{"admin depth 10 ok", testutil.AdminToken(), "?max_depth=10", http.StatusOK},
		{"admin depth 11 refused", testutil.AdminToken(), "?max_depth=11", http.StatusBadRequest},
		{"zero refused", testutil.UserToken(), "?max_depth=0", http.StatusBadRequest},
		{"not a number", testutil.UserToken(), "?max_depth=deep", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &testutil.MockStore{
				GetItemGraphFn: func(_ context.Context, id uuid.UUID, _, _ int, _ []uuid.UUID) (*model.GraphResponse, error) {
					return &model.GraphResponse{Root: model.GraphNode{ID: id}}, nil
				},
			}
			srv := testutil.NewTestServer(mock)
			defer srv.Close()
			resp := getGraph(t, srv.URL, tc.token, tc.query)
			resp.Body.Close()
			if resp.StatusCode != tc.status {
				t.Errorf("expected %d, got %d", tc.status, resp.StatusCode)
			}
		})
	}
}

func TestGetItemGraph_RelationTypeIDs(t *testing.T) {
	mock, _ := graphMock(t, 3, 500, 2)
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	ok := getGraph(t, srv.URL, testutil.UserToken(), "?relation_type_ids="+uuid.New().String()+",%20"+uuid.New().String()+",")
	ok.Body.Close()
	if ok.StatusCode != http.StatusOK {
		t.Errorf("valid ids: expected 200, got %d", ok.StatusCode)
	}

	bad := getGraph(t, srv.URL, testutil.UserToken(), "?relation_type_ids=not-a-uuid")
	bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid id: expected 400, got %d", bad.StatusCode)
	}

	many := "?relation_type_ids=" + uuid.New().String()
	for i := 0; i < 20; i++ {
		many += "," + uuid.New().String()
	}
	tooMany := getGraph(t, srv.URL, testutil.UserToken(), many)
	tooMany.Body.Close()
	if tooMany.StatusCode != http.StatusBadRequest {
		t.Errorf("21 ids: expected 400, got %d", tooMany.StatusCode)
	}
}
