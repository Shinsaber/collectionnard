package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/testutil"
)

func TestDecodeJSON_BodyTooLarge(t *testing.T) {
	srv := testutil.NewTestServer(&testutil.MockStore{})
	defer srv.Close()

	// 2 MiB of valid JSON: above the 1 MiB cap of decodeJSON.
	big := `{"item_type_id":"b0000000-0000-0000-0000-000000000001","attributes":{"titre":"` +
		strings.Repeat("x", 2<<20) + `"}}`
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/items", bytes.NewReader([]byte(big)))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", resp.StatusCode)
	}
	var apiErr model.APIError
	json.NewDecoder(resp.Body).Decode(&apiErr)
	if apiErr.Error != "payload_too_large" {
		t.Errorf("expected payload_too_large, got %q", apiErr.Error)
	}
}

func TestDecodeJSON_Invalid(t *testing.T) {
	srv := testutil.NewTestServer(&testutil.MockStore{})
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/items", bytes.NewReader([]byte(`{not json`)))
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUnauthorized_IsJSON(t *testing.T) {
	srv := testutil.NewTestServer(&testutil.MockStore{})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/items")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
	var apiErr model.APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil || apiErr.Error != "unauthorized" {
		t.Errorf("expected APIError{unauthorized}, got %+v (%v)", apiErr, err)
	}
}
