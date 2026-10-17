// Package musicbrainz reads the MusicBrainz web service (JSON). Only the core
// data is used (CC0: artists, release groups, releases, recordings, labels,
// relations); the tags, genres, ratings and annotations of the users are under
// a non-commercial licence and are never read. No API key; the client spaces
// its requests (one per second and per IP, rule of the service).
package musicbrainz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shinsaber/colec/internal/external"
)

// DefaultBaseURL is the web service, version 2.
const DefaultBaseURL = "https://musicbrainz.org/ws/2"

// Spacing between two requests to musicbrainz.org (the service allows one per second).
const Spacing = 1100 * time.Millisecond

// VariousArtists is the special artist of compilations: never imported as an artist.
const VariousArtists = "89ad4ac3-39f7-470e-963a-56509c546377"

// NoLabel is the special label « [no label] » (self-released).
const NoLabel = "157afde4-4bf5-4039-8ad2-5a15acc85176"

// maxWords bounds the words of a search.
const maxWords = 10

var mbidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsMBID reports whether s is a MusicBrainz identifier (lower-case UUID).
func IsMBID(s string) bool { return mbidRE.MatchString(s) }

// EntityURL is the public page of an entity (kind: artist, release-group, release…).
func EntityURL(kind, id string) string { return "https://musicbrainz.org/" + kind + "/" + id }

type Client struct {
	Fetch   *external.Client
	BaseURL string
}

// New returns a client of musicbrainz.org; it sets the spacing of fetch for that host.
func New(fetch *external.Client) *Client {
	fetch.Throttle("musicbrainz.org", Spacing)
	return &Client{Fetch: fetch, BaseURL: DefaultBaseURL}
}

type ArtistCredit struct {
	Name       string `json:"name"` // name credited (may differ from the artist name)
	JoinPhrase string `json:"joinphrase"`
	Artist     Artist `json:"artist"`
}

type Artist struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SortName       string `json:"sort-name"`
	Type           string `json:"type"`
	Country        string `json:"country"`
	Disambiguation string `json:"disambiguation"`
	LifeSpan       struct {
		Begin string `json:"begin"`
		End   string `json:"end"`
	} `json:"life-span"`
	Relations []Relation `json:"relations"`
}

// Relation is a URL relation (inc=url-rels).
type Relation struct {
	Type string `json:"type"`
	URL  struct {
		Resource string `json:"resource"`
	} `json:"url"`
}

type ReleaseGroup struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	PrimaryType      string         `json:"primary-type"`
	SecondaryTypes   []string       `json:"secondary-types"`
	FirstReleaseDate string         `json:"first-release-date"`
	ArtistCredit     []ArtistCredit `json:"artist-credit"`
	Relations        []Relation     `json:"relations"`
}

type Release struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Date         string         `json:"date"`
	Country      string         `json:"country"`
	Status       string         `json:"status"`
	Barcode      string         `json:"barcode"`
	Packaging    string         `json:"packaging"`
	ArtistCredit []ArtistCredit `json:"artist-credit"`
	LabelInfo    []LabelInfo    `json:"label-info"`
	Media        []Medium       `json:"media"`
	ReleaseGroup *ReleaseGroup  `json:"release-group"`
}

type LabelInfo struct {
	CatalogNumber string `json:"catalog-number"`
	Label         *Label `json:"label"`
}

type Label struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Disambiguation string `json:"disambiguation"`
}

type Medium struct {
	Position   int     `json:"position"`
	Format     string  `json:"format"`
	TrackCount int     `json:"track-count"`
	Tracks     []Track `json:"tracks"`
}

type Track struct {
	Position     int            `json:"position"`
	Title        string         `json:"title"`
	Length       *int           `json:"length"` // milliseconds
	ArtistCredit []ArtistCredit `json:"artist-credit"`
	Recording    Recording      `json:"recording"`
}

type Recording struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Length *int     `json:"length"` // milliseconds
	ISRCs  []string `json:"isrcs"`
}

// Wikidata returns the QID of the wikidata URL relation, or "".
func Wikidata(rels []Relation) string {
	for _, r := range rels {
		if r.Type == "wikidata" {
			if i := strings.LastIndex(r.URL.Resource, "/"); i >= 0 {
				return r.URL.Resource[i+1:]
			}
		}
	}
	return ""
}

// Credit is the credit as displayed: names joined by their join phrases (« A feat. B »).
func Credit(ac []ArtistCredit) string {
	var b strings.Builder
	for _, c := range ac {
		b.WriteString(c.Name)
		b.WriteString(c.JoinPhrase)
	}
	return strings.TrimSpace(b.String())
}

// Escape quotes the special characters of the Lucene query syntax, so that
// text is searched as words.
func Escape(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) { // AND / OR / NOT are operators only in capitals
		if strings.ContainsRune(`+-&|!(){}[]^"~*?:\/`, r) {
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// SearchReleaseGroups searches release groups by title and artist, best match
// first: every word is in the title or in the artist credit (« discovery daft punk »).
func (c *Client) SearchReleaseGroups(ctx context.Context, text string, limit int) ([]ReleaseGroup, error) {
	words := strings.Fields(Escape(text))
	if len(words) > maxWords {
		words = words[:maxWords]
	}
	terms := make([]string, len(words))
	for i, w := range words {
		terms[i] = "(releasegroup:" + w + " OR artist:" + w + ")"
	}
	var resp struct {
		ReleaseGroups []ReleaseGroup `json:"release-groups"`
	}
	err := c.get(ctx, "release-group", url.Values{
		"query": {strings.Join(terms, " AND ")},
		"limit": {strconv.Itoa(limit)},
	}, &resp)
	return resp.ReleaseGroups, err
}

// SearchReleases runs a release search (Lucene query: barcode:…, release:(…)).
func (c *Client) SearchReleases(ctx context.Context, query string, limit int) ([]Release, error) {
	var resp struct {
		Releases []Release `json:"releases"`
	}
	err := c.get(ctx, "release", url.Values{"query": {query}, "limit": {strconv.Itoa(limit)}}, &resp)
	return resp.Releases, err
}

// ReleasesOf lists the releases of a release group, with their labels and media.
func (c *Client) ReleasesOf(ctx context.Context, releaseGroup string, limit int) ([]Release, error) {
	var resp struct {
		Releases []Release `json:"releases"`
	}
	err := c.get(ctx, "release", url.Values{
		"release-group": {releaseGroup}, "inc": {"labels+media"}, "limit": {strconv.Itoa(limit)},
	}, &resp)
	return resp.Releases, err
}

// ReleaseGroup returns a release group with its artists and URL relations.
func (c *Client) ReleaseGroup(ctx context.Context, id string) (*ReleaseGroup, error) {
	var rg ReleaseGroup
	if err := c.get(ctx, "release-group/"+id, url.Values{"inc": {"artist-credits+url-rels"}}, &rg); err != nil {
		return nil, err
	}
	return &rg, nil
}

// Release returns a release with its artists, labels, tracks (recordings and
// ISRCs) and release group.
func (c *Client) Release(ctx context.Context, id string) (*Release, error) {
	var r Release
	inc := "artist-credits+labels+recordings+release-groups+isrcs"
	if err := c.get(ctx, "release/"+id, url.Values{"inc": {inc}}, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Artist returns an artist with its URL relations.
func (c *Client) Artist(ctx context.Context, id string) (*Artist, error) {
	var a Artist
	if err := c.get(ctx, "artist/"+id, url.Values{"inc": {"url-rels"}}, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	q.Set("fmt", "json")
	raw := q.Encode()
	if inc := q.Get("inc"); inc != "" {
		// The inc values are joined by « + », which the service expects unescaped.
		q.Del("inc")
		raw = q.Encode() + "&inc=" + inc
	}
	body, err := c.Fetch.Get(ctx, c.BaseURL+"/"+path+"?"+raw)
	var status *external.StatusError
	if errors.As(err, &status) && status.Code == http.StatusServiceUnavailable {
		// Over the rate limit, or the service is busy: one more try, after the spacing.
		body, err = c.Fetch.Get(ctx, c.BaseURL+"/"+path+"?"+raw)
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%w: musicbrainz: %v", external.ErrUnavailable, err)
	}
	return nil
}
