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

func TestGetItemGraph_Success(t *testing.T) {
	rootID := uuid.MustParse("e0000000-0000-0000-0000-000000000001")
	mock := &testutil.MockStore{
		GetItemGraphFn: func(_ context.Context, id uuid.UUID, depth int, rtIDs []uuid.UUID) (*model.GraphResponse, error) {
			return &model.GraphResponse{
				Root: model.GraphNode{
					ID: rootID, Type: "Serie", Label: "Breaking Bad",
					Attributes: json.RawMessage(`{"titre":"Breaking Bad"}`), Depth: 0,
				},
				Nodes: []model.GraphNode{
					{
						ID: uuid.New(), Type: "Diffusion", Label: "AMC",
						Attributes: json.RawMessage(`{"nom":"AMC"}`), Depth: 1,
					},
				},
				Edges: []model.GraphEdge{
					{
						SourceID: rootID, TargetID: uuid.New(),
						RelationType: "diffuse_en", Attributes: json.RawMessage(`{}`),
					},
				},
			}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/items/"+rootID.String()+"/graph?max_depth=2", nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result model.GraphResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Root.Label != "Breaking Bad" {
		t.Errorf("expected Breaking Bad, got %s", result.Root.Label)
	}
	if len(result.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(result.Nodes))
	}
	if len(result.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(result.Edges))
	}
}

func TestGetStats_Success(t *testing.T) {
	mock := &testutil.MockStore{
		GetStatsFn: func(_ context.Context) (*model.Stats, error) {
			return &model.Stats{
				ItemsByType:      []model.ItemsByType{{TypeName: "Serie", Count: 5}},
				TotalItems:       42,
				TotalCollections: 3,
				TotalRelations:   100,
			}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/stats", nil)
	req.Header = testutil.AuthHeader(testutil.UserToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var stats model.Stats
	json.NewDecoder(resp.Body).Decode(&stats)
	if stats.TotalItems != 42 {
		t.Errorf("expected 42 items, got %d", stats.TotalItems)
	}
}
