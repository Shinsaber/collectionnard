package external

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("User-Agent") != "test-agent" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		switch r.URL.Path {
		case "/big":
			_, _ = w.Write([]byte(strings.Repeat("x", 20)))
		case "/fail":
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			_, _ = w.Write([]byte("ok"))
		}
	}))
	defer srv.Close()

	c := NewClient("test-agent")
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	c.MaxBody = 10
	ctx := context.Background()

	for range 2 {
		if b, err := c.Get(ctx, srv.URL+"/ok"); err != nil || string(b) != "ok" {
			t.Fatalf("Get = %q, %v", b, err)
		}
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (cached)", calls)
	}
	now = now.Add(c.TTL + time.Second)
	if _, err := c.Get(ctx, srv.URL+"/ok"); err != nil || calls != 2 {
		t.Errorf("expired entry: calls = %d, err = %v", calls, err)
	}
	for _, path := range []string{"/big", "/fail"} {
		if _, err := c.Get(ctx, srv.URL+path); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: err = %v, want ErrUnavailable", path, err)
		}
	}

	c.MaxEntries = 2
	for _, p := range []string{"/a", "/b", "/c"} {
		_, _ = c.Get(ctx, srv.URL+p)
	}
	if len(c.cache) > 2 {
		t.Errorf("cache size = %d, want ≤ 2", len(c.cache))
	}
}

func TestClient_Throttle(t *testing.T) {
	var times []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		times = append(times, time.Now())
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient("test-agent")
	host := strings.TrimPrefix(srv.URL, "http://")
	c.Throttle(host, 50*time.Millisecond)
	ctx := context.Background()
	for _, p := range []string{"/a", "/b", "/a"} { // the second /a comes from the cache
		if _, err := c.Get(ctx, srv.URL+p); err != nil {
			t.Fatal(err)
		}
	}
	if len(times) != 2 {
		t.Fatalf("calls = %d, want 2", len(times))
	}
	if gap := times[1].Sub(times[0]); gap < 45*time.Millisecond {
		t.Errorf("gap = %v, want ≥ 50ms", gap)
	}

	c.Throttle(host, time.Hour)
	_, _ = c.Get(ctx, srv.URL+"/c") // takes the free slot
	short, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	if _, err := c.Get(short, srv.URL+"/d"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Get past the deadline = %v, want ErrUnavailable", err)
	}
}
