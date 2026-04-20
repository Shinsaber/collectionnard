package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
)

const testSecret = "test-secret-key"

func TestAuthenticate_ValidToken(t *testing.T) {
	auth := middleware.NewAuth(testSecret)
	userID := uuid.New()

	token, err := auth.GenerateToken(userID, "admin")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	called := false
	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotID := middleware.GetUserID(r.Context())
		if gotID != userID {
			t.Errorf("expected user ID %s, got %s", userID, gotID)
		}
		gotRole := middleware.GetRole(r.Context())
		if gotRole != "admin" {
			t.Errorf("expected role admin, got %s", gotRole)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthenticate_MissingToken(t *testing.T) {
	auth := middleware.NewAuth(testSecret)
	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	auth := middleware.NewAuth(testSecret)
	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_WrongSecret(t *testing.T) {
	auth1 := middleware.NewAuth("secret-1")
	auth2 := middleware.NewAuth("secret-2")

	token, _ := auth1.GenerateToken(uuid.New(), "user")

	handler := auth2.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAdmin_AdminAllowed(t *testing.T) {
	auth := middleware.NewAuth(testSecret)
	token, _ := auth.GenerateToken(uuid.New(), "admin")

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := auth.Authenticate(auth.RequireAdmin(inner))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
}

func TestRequireAdmin_UserDenied(t *testing.T) {
	auth := middleware.NewAuth(testSecret)
	token, _ := auth.GenerateToken(uuid.New(), "user")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	handler := auth.Authenticate(auth.RequireAdmin(inner))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
