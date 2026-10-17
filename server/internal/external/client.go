// Package external fetches the open data sources used to fill the catalogue
// (Wikidata, BnF, MusicBrainz). Subpackages hold one client per source; the
// connectors (package connectors) map their data onto the item types of a domain.
package external

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ErrUnavailable wraps every failure to reach a source or to read its answer
// (network, timeout, HTTP status, oversized or malformed body).
var ErrUnavailable = errors.New("external source unavailable")

// ErrNotFound means the source answered but does not know the requested entity.
var ErrNotFound = errors.New("external entity not found")

// StatusError is an answer other than 200 or 404 (an ErrUnavailable).
type StatusError struct {
	Host string
	Code int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%v: %s answered HTTP %d", ErrUnavailable, e.Host, e.Code)
}

func (e *StatusError) Unwrap() error { return ErrUnavailable }

const (
	defaultTimeout    = 20 * time.Second
	defaultMaxBody    = 20 << 20 // 20 MiB: a BnF page of 1000 UNIMARC records is about 7 MiB
	defaultTTL        = 15 * time.Minute
	defaultMaxEntries = 256
)

// Client performs GET requests with an identified User-Agent, a timeout, a
// body size cap and a small in-memory cache: the preview and the import of the
// same selection reuse the answers instead of querying the sources twice.
type Client struct {
	HTTP       *http.Client
	UserAgent  string
	MaxBody    int64
	TTL        time.Duration
	MaxEntries int

	mu    sync.Mutex
	cache map[string]cacheEntry
	now   func() time.Time

	// spacing: minimum delay between two requests to a host (rate limit of
	// the source, such as MusicBrainz: one request per second).
	throttleMu sync.Mutex
	spacing    map[string]time.Duration
	next       map[string]time.Time
}

type cacheEntry struct {
	body    []byte
	expires time.Time
}

// NewClient returns a client with the default timeout, size cap and cache.
func NewClient(userAgent string) *Client {
	return &Client{
		HTTP:       &http.Client{Timeout: defaultTimeout},
		UserAgent:  userAgent,
		MaxBody:    defaultMaxBody,
		TTL:        defaultTTL,
		MaxEntries: defaultMaxEntries,
		cache:      map[string]cacheEntry{},
		now:        time.Now,
	}
}

// Throttle spaces the requests to host by at least d (answers from the cache
// are not delayed).
func (c *Client) Throttle(host string, d time.Duration) {
	c.throttleMu.Lock()
	defer c.throttleMu.Unlock()
	if c.spacing == nil {
		c.spacing, c.next = map[string]time.Duration{}, map[string]time.Time{}
	}
	c.spacing[host] = d
}

// wait blocks until a request to host is allowed (or ctx ends).
func (c *Client) wait(ctx context.Context, host string) error {
	c.throttleMu.Lock()
	d := c.spacing[host]
	if d <= 0 {
		c.throttleMu.Unlock()
		return nil
	}
	now := time.Now()
	at := c.next[host]
	if at.Before(now) {
		at = now
	}
	c.next[host] = at.Add(d) // the slot is reserved before releasing the lock
	c.throttleMu.Unlock()
	if delay := time.Until(at); delay > 0 {
		t := time.NewTimer(delay)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
		case <-t.C:
		}
	}
	return nil
}

// Get returns the body of a 200 answer (from the cache when still fresh).
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	if body, ok := c.cached(url); ok {
		return body, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if err := c.wait(ctx, req.URL.Host); err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s answered HTTP 404", ErrNotFound, req.URL.Host)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Host: req.URL.Host, Code: resp.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.MaxBody+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if int64(len(body)) > c.MaxBody {
		return nil, fmt.Errorf("%w: %s answer larger than %d bytes", ErrUnavailable, req.URL.Host, c.MaxBody)
	}
	c.store(url, body)
	return body, nil
}

func (c *Client) cached(url string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[url]
	if !ok || c.clock().After(e.expires) {
		return nil, false
	}
	return e.body, true
}

func (c *Client) store(url string, body []byte) {
	if c.TTL <= 0 || c.MaxEntries <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = map[string]cacheEntry{}
	}
	now := c.clock()
	if len(c.cache) >= c.MaxEntries {
		for k, e := range c.cache {
			if now.After(e.expires) {
				delete(c.cache, k)
			}
		}
		// Still full: drop arbitrary entries (map order), the cache is only a shortcut.
		for k := range c.cache {
			if len(c.cache) < c.MaxEntries {
				break
			}
			delete(c.cache, k)
		}
	}
	c.cache[url] = cacheEntry{body: body, expires: now.Add(c.TTL)}
}

func (c *Client) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}
