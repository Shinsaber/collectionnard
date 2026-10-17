// Package tvmaze reads the TVmaze API (JSON, no key). Its data is under CC BY-SA:
// TVmaze is credited in `sources` and by a link to the show; summaries and
// images are never read. The client spaces its requests (at least 20 calls per
// 10 seconds are allowed) and tries once more after a 429.
package tvmaze

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/shinsaber/colec/internal/external"
)

// DefaultBaseURL is the public API.
const DefaultBaseURL = "https://api.tvmaze.com"

// Spacing between two requests to api.tvmaze.com.
const Spacing = 550 * time.Millisecond

// ShowURL is the public page of a show (attribution link).
func ShowURL(id int) string { return "https://www.tvmaze.com/shows/" + strconv.Itoa(id) }

type Client struct {
	Fetch   *external.Client
	BaseURL string
}

// New returns a client of api.tvmaze.com; it sets the spacing of fetch for that host.
func New(fetch *external.Client) *Client {
	fetch.Throttle("api.tvmaze.com", Spacing)
	return &Client{Fetch: fetch, BaseURL: DefaultBaseURL}
}

type Show struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Premiered string `json:"premiered"`
	Ended     string `json:"ended"`
	URL       string `json:"url"`
	Network   *struct {
		Name string `json:"name"`
	} `json:"network"`
	WebChannel *struct {
		Name string `json:"name"`
	} `json:"webChannel"`
}

type Season struct {
	ID           int    `json:"id"`
	Number       int    `json:"number"`
	EpisodeOrder int    `json:"episodeOrder"`
	PremiereDate string `json:"premiereDate"`
}

type Episode struct {
	ID      int    `json:"id"`
	Season  int    `json:"season"`
	Number  int    `json:"number"` // 0 for a special
	Name    string `json:"name"`
	Airdate string `json:"airdate"`
	Runtime int    `json:"runtime"`
	Type    string `json:"type"` // regular, significant_special…
}

// LookupIMDb returns the show of an IMDb identifier (external.ErrNotFound when unknown).
func (c *Client) LookupIMDb(ctx context.Context, imdb string) (*Show, error) {
	var s Show
	if err := c.get(ctx, "/lookup/shows?imdb="+url.QueryEscape(imdb), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Seasons lists the seasons of a show.
func (c *Client) Seasons(ctx context.Context, show int) ([]Season, error) {
	var out []Season
	return out, c.get(ctx, "/shows/"+strconv.Itoa(show)+"/seasons", &out)
}

// Episodes lists the regular episodes of a show (specials left out).
func (c *Client) Episodes(ctx context.Context, show int) ([]Episode, error) {
	var out []Episode
	return out, c.get(ctx, "/shows/"+strconv.Itoa(show)+"/episodes", &out)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	body, err := c.Fetch.Get(ctx, c.BaseURL+path)
	var status *external.StatusError
	if errors.As(err, &status) && status.Code == http.StatusTooManyRequests {
		body, err = c.Fetch.Get(ctx, c.BaseURL+path) // rate limit: one more try, after the spacing
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%w: tvmaze: %v", external.ErrUnavailable, err)
	}
	return nil
}
