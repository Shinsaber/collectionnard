// Package wikidata reads entities from the Wikidata API (structured data under
// CC0: labels, descriptions and statements; never Wikipedia text).
package wikidata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/shinsaber/colec/internal/external"
)

// DefaultBaseURL is the action API of Wikidata.
const DefaultBaseURL = "https://www.wikidata.org/w/api.php"

// maxIDs is the number of entities wbgetentities accepts per call.
const maxIDs = 50

var qidRE = regexp.MustCompile(`^Q[1-9][0-9]*$`)

// IsQID reports whether s is a Wikidata item identifier (Q42).
func IsQID(s string) bool { return qidRE.MatchString(s) }

// EntityURL is the public page of an entity.
func EntityURL(id string) string { return "https://www.wikidata.org/wiki/" + id }

type Client struct {
	Fetch   *external.Client
	BaseURL string
}

func New(fetch *external.Client) *Client {
	return &Client{Fetch: fetch, BaseURL: DefaultBaseURL}
}

// Search returns the identifiers of the items matching text that are an
// instance (P31) of one of classes, best match first.
func (c *Client) Search(ctx context.Context, text string, classes []string, limit int) ([]string, error) {
	filter := make([]string, len(classes))
	for i, cl := range classes {
		filter[i] = "P31=" + cl
	}
	q := url.Values{
		"action":      {"query"},
		"list":        {"search"},
		"srsearch":    {text + " haswbstatement:" + strings.Join(filter, "|")},
		"srnamespace": {"0"},
		"srlimit":     {strconv.Itoa(limit)},
		"srprop":      {""},
		"format":      {"json"},
	}
	var resp struct {
		Query struct {
			Search []struct {
				Title string `json:"title"`
			} `json:"search"`
		} `json:"query"`
		Error *apiError `json:"error"`
	}
	if err := c.get(ctx, q, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%w: wikidata search: %s", external.ErrUnavailable, resp.Error.Info)
	}
	ids := make([]string, 0, len(resp.Query.Search))
	for _, s := range resp.Query.Search {
		if IsQID(s.Title) {
			ids = append(ids, s.Title)
		}
	}
	return ids, nil
}

// Entities fetches entities by id with the labels and descriptions in langs.
// withClaims adds the statements (much larger answers). Unknown ids are
// missing from the map.
func (c *Client) Entities(ctx context.Context, ids, langs []string, withClaims bool) (map[string]*Entity, error) {
	props := "labels|descriptions"
	if withClaims {
		props += "|claims"
	}
	out := make(map[string]*Entity, len(ids))
	for start := 0; start < len(ids); start += maxIDs {
		chunk := ids[start:min(start+maxIDs, len(ids))]
		q := url.Values{
			"action":    {"wbgetentities"},
			"ids":       {strings.Join(chunk, "|")},
			"props":     {props},
			"languages": {strings.Join(langs, "|")},
			"format":    {"json"},
		}
		var resp struct {
			Entities map[string]rawEntity `json:"entities"`
			Error    *apiError            `json:"error"`
		}
		if err := c.get(ctx, q, &resp); err != nil {
			return nil, err
		}
		// One unknown id fails the whole call: retry the ids one by one.
		if resp.Error != nil && resp.Error.Code == "no-such-entity" && len(chunk) > 1 {
			for _, id := range chunk {
				one, err := c.Entities(ctx, []string{id}, langs, withClaims)
				if err != nil {
					return nil, err
				}
				for k, v := range one {
					out[k] = v
				}
			}
			continue
		}
		if resp.Error != nil {
			if resp.Error.Code == "no-such-entity" {
				continue
			}
			return nil, fmt.Errorf("%w: wikidata entities: %s", external.ErrUnavailable, resp.Error.Info)
		}
		for id, raw := range resp.Entities {
			if raw.Missing != nil || raw.ID == "" {
				continue
			}
			out[id] = raw.entity()
		}
	}
	return out, nil
}

// StringClaims returns the string values of one property of an entity, without
// downloading the whole entity (used for the ISO code P297 of large countries).
func (c *Client) StringClaims(ctx context.Context, id, property string) ([]string, error) {
	q := url.Values{
		"action":   {"wbgetclaims"},
		"entity":   {id},
		"property": {property},
		"format":   {"json"},
	}
	var resp struct {
		Claims map[string][]rawStatement `json:"claims"`
		Error  *apiError                 `json:"error"`
	}
	if err := c.get(ctx, q, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%w: wikidata claims: %s", external.ErrUnavailable, resp.Error.Info)
	}
	var out []string
	for _, st := range orderStatements(resp.Claims[property]) {
		if v := st.value(); v.Text != "" {
			out = append(out, v.Text)
		}
	}
	return out, nil
}

func (c *Client) get(ctx context.Context, q url.Values, v any) error {
	body, err := c.Fetch.Get(ctx, c.BaseURL+"?"+q.Encode())
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("%w: wikidata: invalid JSON: %v", external.ErrUnavailable, err)
	}
	return nil
}

type apiError struct {
	Code string `json:"code"`
	Info string `json:"info"`
}
