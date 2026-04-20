package handler_test

import (
	"context"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestLogin_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	mock := &testutil.MockStore{
		GetUserByUsernameFn: func(_ context.Context, username string) (*model.User, string, error) {
			if username == "admin" {
				return testutil.AdminUser(), string(hash), nil
			}
			return nil, "", store.ErrNotFound
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.LoginRequest{Username: "admin", Password: "secret"})
	resp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var result model.LoginResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Token == "" {
		t.Error("expected non-empty token")
	}
	if result.User.Username != "admin" {
		t.Errorf("expected admin, got %s", result.User.Username)
	}
}

func TestLogin_BadCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	mock := &testutil.MockStore{
		GetUserByUsernameFn: func(_ context.Context, username string) (*model.User, string, error) {
			return testutil.AdminUser(), string(hash), nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.LoginRequest{Username: "admin", Password: "wrong"})
	resp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	mock := &testutil.MockStore{} // default returns ErrNotFound
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.LoginRequest{Username: "ghost", Password: "x"})
	resp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestGetCurrentUser_Success(t *testing.T) {
	mock := &testutil.MockStore{
		GetUserByIDFn: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			if id == testutil.AdminID {
				return testutil.AdminUser(), nil
			}
			return nil, store.ErrNotFound
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/auth/me", nil)
	req.Header = testutil.AuthHeader(testutil.AdminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetCurrentUser_NoToken(t *testing.T) {
	mock := &testutil.MockStore{}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/auth/me")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
