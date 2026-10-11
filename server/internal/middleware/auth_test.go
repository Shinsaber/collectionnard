package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
)

const (
	testSecret = "test-secret-key-test-secret-key-0000"
	testTTL    = 15 * time.Minute
)

func newAuth() *middleware.Auth {
	return middleware.NewAuth(testSecret, testTTL)
}

// serve runs the Authenticate middleware around a handler that records whether it was called.
func serve(t *testing.T, auth *middleware.Auth, authorization string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	called := false
	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w, called
}

func TestAuthenticate_ValidToken(t *testing.T) {
	auth := newAuth()
	userID := uuid.New()

	token, err := auth.GenerateToken(userID, "admin")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	called := false
	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if gotID := middleware.GetUserID(r.Context()); gotID != userID {
			t.Errorf("expected user ID %s, got %s", userID, gotID)
		}
		if gotRole := middleware.GetRole(r.Context()); gotRole != "admin" {
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

func TestGenerateToken_HasRegisteredClaims(t *testing.T) {
	auth := newAuth()
	token, _ := auth.GenerateToken(uuid.New(), "user")

	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Issuer != middleware.Issuer {
		t.Errorf("expected issuer %q, got %q", middleware.Issuer, claims.Issuer)
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Fatal("expected exp and iat claims")
	}
	if got := claims.ExpiresAt.Sub(claims.IssuedAt.Time); got != testTTL {
		t.Errorf("expected exp-iat = %s, got %s", testTTL, got)
	}
}

func TestAuthenticate_MissingToken(t *testing.T) {
	w, called := serve(t, newAuth(), "")
	if called {
		t.Error("handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected JSON error, got Content-Type %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["error"] != "unauthorized" {
		t.Errorf("expected APIError body, got %s", w.Body.String())
	}
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	w, called := serve(t, newAuth(), "Bearer invalid.token.here")
	if called {
		t.Error("handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_WrongSecret(t *testing.T) {
	auth1 := middleware.NewAuth("secret-1-secret-1-secret-1-secret-1", testTTL)
	auth2 := middleware.NewAuth("secret-2-secret-2-secret-2-secret-2", testTTL)
	token, _ := auth1.GenerateToken(uuid.New(), "user")

	w, called := serve(t, auth2, "Bearer "+token)
	if called {
		t.Error("handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_ExpiredToken(t *testing.T) {
	// Negative TTL: the token is already expired (beyond the 30 s leeway).
	expiredAuth := middleware.NewAuth(testSecret, -2*time.Minute)
	token, _ := expiredAuth.GenerateToken(uuid.New(), "admin")

	w, called := serve(t, newAuth(), "Bearer "+token)
	if called {
		t.Error("handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_MissingExpClaim(t *testing.T) {
	// A token signed with the right secret but without `exp` must be rejected.
	claims := jwt.MapClaims{"sub": uuid.New().String(), "role": "admin", "iss": middleware.Issuer}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))

	w, called := serve(t, newAuth(), "Bearer "+token)
	if called {
		t.Error("handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_WrongIssuer(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Issuer: "someone-else", Subject: uuid.New().String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))

	w, _ := serve(t, newAuth(), "Bearer "+token)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_RejectsOtherHMACAlgorithms(t *testing.T) {
	// HS512 with the right secret: only HS256 is accepted.
	claims := jwt.RegisteredClaims{
		Issuer: middleware.Issuer, Subject: uuid.New().String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(testSecret))

	w, called := serve(t, newAuth(), "Bearer "+token)
	if called {
		t.Error("handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_RejectsNoneAlgorithm(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Issuer: middleware.Issuer, Subject: uuid.New().String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}

	w, called := serve(t, newAuth(), "Bearer "+token)
	if called {
		t.Error("handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAdmin_AdminAllowed(t *testing.T) {
	auth := newAuth()
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
	auth := newAuth()
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
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected JSON error, got Content-Type %q", ct)
	}
}
