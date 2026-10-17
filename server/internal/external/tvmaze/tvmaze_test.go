package tvmaze

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shinsaber/colec/internal/external"
)

// A 429 (rate limit) is tried once more; an unknown IMDb id is ErrNotFound.
func TestClient(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch {
		case r.URL.Query().Get("imdb") == "tt0000000":
			http.NotFound(w, r)
		case calls == 1:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			_, _ = w.Write([]byte(`{"id":169,"name":"Breaking Bad","network":{"name":"AMC"}}`))
		}
	}))
	defer srv.Close()
	c := New(external.NewClient("test"))
	c.BaseURL = srv.URL
	s, err := c.LookupIMDb(context.Background(), "tt0903747")
	if err != nil || s.ID != 169 || s.Network.Name != "AMC" || calls != 2 {
		t.Errorf("LookupIMDb = %+v, %v after %d calls", s, err, calls)
	}
	if _, err := c.LookupIMDb(context.Background(), "tt0000000"); !errors.Is(err, external.ErrNotFound) {
		t.Errorf("unknown show: %v", err)
	}
	if ShowURL(169) != "https://www.tvmaze.com/shows/169" {
		t.Error("ShowURL")
	}
}
