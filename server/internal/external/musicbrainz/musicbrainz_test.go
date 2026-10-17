package musicbrainz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shinsaber/colec/internal/external"
)

func TestEscapeAndIDs(t *testing.T) {
	if got := Escape(`AC/DC: Back (Live) AND "more"`); got != `ac\/dc\: back \(live\) and \"more\"` {
		t.Errorf("Escape = %q", got)
	}
	if !IsMBID("b1392450-e666-3926-a536-22c65f834433") || IsMBID("B1392450-E666-3926-A536-22C65F834433") || IsMBID("Q202996") {
		t.Error("IsMBID")
	}
	ac := []ArtistCredit{{Name: "Daft Punk", JoinPhrase: " feat. "}, {Name: "Pharrell Williams"}}
	if Credit(ac) != "Daft Punk feat. Pharrell Williams" {
		t.Errorf("Credit = %q", Credit(ac))
	}
	if Wikidata([]Relation{{Type: "discogs"}, {Type: "wikidata", URL: struct {
		Resource string `json:"resource"`
	}{"https://www.wikidata.org/wiki/Q44190"}}}) != "Q44190" {
		t.Error("Wikidata")
	}
}

// The inc values keep their « + », the query text is encoded.
func TestRequestURL(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.URL.RequestURI())
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := New(external.NewClient("test"))
	c.BaseURL = srv.URL + "/ws/2"
	ctx := context.Background()
	if _, err := c.Release(ctx, "0c324f3a-dcbb-4f09-b33a-c03c676b4044"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SearchReleaseGroups(ctx, "a+b", 5); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/ws/2/release/0c324f3a-dcbb-4f09-b33a-c03c676b4044?fmt=json&inc=artist-credits+labels+recordings+release-groups+isrcs",
		"/ws/2/release-group?fmt=json&limit=5&query=%28releasegroup%3Aa%5C%2Bb+OR+artist%3Aa%5C%2Bb%29",
	}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Errorf("request %d = %v, want %s", i, got, want[i])
		}
	}
}

// A 503 (rate limit) is tried once more.
func TestRetryOn503(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"id":"a74b1b7f-71a5-4011-9441-d0b5e4122711","name":"Radiohead"}`))
	}))
	defer srv.Close()
	c := New(external.NewClient("test"))
	c.BaseURL = srv.URL
	a, err := c.Artist(context.Background(), "a74b1b7f-71a5-4011-9441-d0b5e4122711")
	if err != nil || a.Name != "Radiohead" || calls != 2 {
		t.Errorf("Artist = %+v, %v after %d calls", a, err, calls)
	}
}
