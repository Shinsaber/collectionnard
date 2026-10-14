package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

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

// --- Refresh tokens ---

func TestLogin_ReturnsRefreshToken(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	var storedHash string
	mock := &testutil.MockStore{
		GetUserByUsernameFn: func(_ context.Context, username string) (*model.User, string, error) {
			return testutil.AdminUser(), string(hash), nil
		},
		CreateRefreshTokenFn: func(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
			if userID != testutil.AdminID {
				t.Errorf("expected admin user id, got %s", userID)
			}
			if time.Until(expiresAt) < 23*time.Hour {
				t.Errorf("expected ~24h refresh TTL, got %s", time.Until(expiresAt))
			}
			storedHash = tokenHash
			return nil
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
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result model.LoginResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.RefreshToken == "" {
		t.Error("expected non-empty refresh_token")
	}
	if result.ExpiresIn != int(testutil.TestAccessTTL.Seconds()) {
		t.Errorf("expected expires_in %d, got %d", int(testutil.TestAccessTTL.Seconds()), result.ExpiresIn)
	}
	if storedHash == "" || storedHash == result.RefreshToken {
		t.Error("the store must receive a hash, not the raw refresh token")
	}
	if len(storedHash) != 64 {
		t.Errorf("expected hex SHA-256 (64 chars), got %d chars", len(storedHash))
	}
}

func TestRefresh_Success_Rotates(t *testing.T) {
	consumed := []string{}
	created := []string{}
	mock := &testutil.MockStore{
		ConsumeRefreshTokenFn: func(_ context.Context, tokenHash string) (uuid.UUID, error) {
			consumed = append(consumed, tokenHash)
			return testutil.UserID, nil
		},
		GetUserByIDFn: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			if id == testutil.UserID {
				return testutil.RegularUser(), nil
			}
			return nil, store.ErrNotFound
		},
		CreateRefreshTokenFn: func(_ context.Context, userID uuid.UUID, tokenHash string, _ time.Time) error {
			created = append(created, tokenHash)
			return nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.RefreshRequest{RefreshToken: "old-refresh-token"})
	resp, err := http.Post(srv.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(body))
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
	if result.Token == "" || result.RefreshToken == "" {
		t.Error("expected new access and refresh tokens")
	}
	if result.User.ID != testutil.UserID {
		t.Errorf("expected user %s, got %s", testutil.UserID, result.User.ID)
	}
	if len(consumed) != 1 || len(created) != 1 {
		t.Fatalf("expected 1 consume and 1 create, got %d/%d", len(consumed), len(created))
	}
	if consumed[0] == created[0] {
		t.Error("rotation: the new refresh token must differ from the consumed one")
	}

	// The new access token must be usable.
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/auth/me", nil)
	req.Header = testutil.AuthHeader(result.Token)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with refreshed token, got %d", resp2.StatusCode)
	}
}

func TestRefresh_UnknownOrExpiredToken(t *testing.T) {
	mock := &testutil.MockStore{} // ConsumeRefreshToken → ErrNotFound by default
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.RefreshRequest{RefreshToken: "whatever"})
	resp, err := http.Post(srv.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestRefresh_DeletedUser(t *testing.T) {
	mock := &testutil.MockStore{
		ConsumeRefreshTokenFn: func(_ context.Context, _ string) (uuid.UUID, error) {
			return testutil.UserID, nil
		},
		// GetUserByID → ErrNotFound by default: the account no longer exists.
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.RefreshRequest{RefreshToken: "valid-but-orphan"})
	resp, err := http.Post(srv.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestRefresh_MissingToken(t *testing.T) {
	srv := testutil.NewTestServer(&testutil.MockStore{})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLogout_RevokesRefreshToken(t *testing.T) {
	deleted := ""
	mock := &testutil.MockStore{
		DeleteRefreshTokenFn: func(_ context.Context, tokenHash string) error {
			deleted = tokenHash
			return nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	body, _ := json.Marshal(model.RefreshRequest{RefreshToken: "my-refresh-token"})
	resp, err := http.Post(srv.URL+"/api/v1/auth/logout", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if deleted == "" || deleted == "my-refresh-token" {
		t.Error("expected the store to receive the token hash")
	}
}

// --- Rate limiting ---

func TestLogin_RateLimited(t *testing.T) {
	cfg := testutil.TestConfig()
	cfg.LoginRateLimit = 3
	srv := testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(&testutil.MockStore{}, cfg))
	defer srv.Close()

	body, _ := json.Marshal(model.LoginRequest{Username: "ghost", Password: "x"})
	var last *http.Response
	for i := 0; i < 4; i++ {
		resp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}
		resp.Body.Close()
		if i < 3 && resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("request %d: expected 401, got %d", i+1, resp.StatusCode)
		}
		last = resp
	}
	if last.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("4th request: expected 429, got %d", last.StatusCode)
	}
	if last.Header.Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429")
	}
}
