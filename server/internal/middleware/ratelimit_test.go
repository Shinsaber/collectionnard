package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shinsaber/colec/internal/middleware"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := middleware.NewRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if ok, _ := rl.Allow("a"); !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	ok, retry := rl.Allow("a")
	if ok {
		t.Fatal("4th request should be rejected")
	}
	if retry <= 0 || retry > time.Minute {
		t.Errorf("unexpected retryAfter %s", retry)
	}
	// Another key is independent.
	if ok, _ := rl.Allow("b"); !ok {
		t.Error("other key should be allowed")
	}
}

func TestRateLimiter_Handler(t *testing.T) {
	rl := middleware.NewRateLimiter(2, time.Minute)
	h := rl.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	codes := []int{}
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		codes = append(codes, w.Code)
		if i == 2 {
			if w.Header().Get("Retry-After") == "" {
				t.Error("expected Retry-After header")
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected JSON body, got %q", ct)
			}
		}
	}
	want := []int{200, 200, 429}
	for i := range want {
		if codes[i] != want[i] {
			t.Errorf("request %d: expected %d, got %d", i+1, want[i], codes[i])
		}
	}

	// A different IP is not affected.
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("other IP: expected 200, got %d", w.Code)
	}
}
