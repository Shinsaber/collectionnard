// Package bnf queries the general catalogue of the Bibliothèque nationale de
// France through its SRU API (UNIMARC records). The descriptive metadata is
// under the Licence Ouverte (Etalab): reuse must cite the source and the
// retrieval date.
package bnf

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/shinsaber/colec/internal/external"
)

// DefaultBaseURL is the SRU endpoint of the general catalogue.
const DefaultBaseURL = "https://catalogue.bnf.fr/api/SRU"

// pageSize is the number of records asked per SRU call (the server accepts 1000).
const pageSize = 500

type Client struct {
	Fetch   *external.Client
	BaseURL string
}

func New(fetch *external.Client) *Client {
	return &Client{Fetch: fetch, BaseURL: DefaultBaseURL}
}

// Search runs a CQL query and returns at most limit UNIMARC records, with the
// total number of matches.
func (c *Client) Search(ctx context.Context, cql string, limit int) ([]Record, int, error) {
	var out []Record
	total := 0
	for start := 1; len(out) < limit; {
		q := url.Values{
			"version":        {"1.2"},
			"operation":      {"searchRetrieve"},
			"query":          {cql},
			"recordSchema":   {"unimarcXchange"},
			"maximumRecords": {strconv.Itoa(min(pageSize, limit-len(out)))},
			"startRecord":    {strconv.Itoa(start)},
		}
		body, err := c.Fetch.Get(ctx, c.BaseURL+"?"+q.Encode())
		if err != nil {
			return nil, 0, err
		}
		var resp sruResponse
		if err := xml.Unmarshal(body, &resp); err != nil {
			return nil, 0, fmt.Errorf("%w: bnf: invalid XML: %v", external.ErrUnavailable, err)
		}
		if len(resp.Diagnostics) > 0 {
			return nil, 0, fmt.Errorf("%w: bnf: %s %s", external.ErrUnavailable,
				resp.Diagnostics[0].Message, resp.Diagnostics[0].Details)
		}
		total = resp.Total
		for _, r := range resp.Records {
			out = append(out, r.record())
		}
		if resp.Next <= start || len(resp.Records) == 0 {
			break
		}
		start = resp.Next
	}
	return out, total, nil
}

// Quote makes a user value safe inside a CQL quoted term.
func Quote(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '"', '\\', '*', '?', '^':
			return ' '
		}
		return r
	}, s)
	return `"` + strings.Join(strings.Fields(s), " ") + `"`
}

// ── Records ─────────────────────────────────────────────────────────

// Record is a UNIMARC bibliographic record.
type Record struct {
	ARK    string // "ark:/12148/cb343384394"
	Leader string
	Fields []Field
}

// Field is a control field (Value) or a data field (Subfields).
type Field struct {
	Tag       string
	Value     string
	Subfields []Subfield
}

type Subfield struct {
	Code, Value string
}

// Get returns the first value of a subfield in a data field.
func (f Field) Get(code string) string {
	for _, s := range f.Subfields {
		if s.Code == code {
			return strings.TrimSpace(s.Value)
		}
	}
	return ""
}

// All returns every value of a subfield in a data field.
func (f Field) All(code string) []string {
	var out []string
	for _, s := range f.Subfields {
		if s.Code == code {
			if v := strings.TrimSpace(s.Value); v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// Get returns the first value of tag$code in the record.
func (r Record) Get(tag, code string) string {
	for _, f := range r.Fields {
		if f.Tag == tag {
			if v := f.Get(code); v != "" {
				return v
			}
		}
	}
	return ""
}

// All returns every value of tag$code in the record.
func (r Record) All(tag, code string) []string {
	var out []string
	for _, f := range r.Fields {
		if f.Tag == tag {
			out = append(out, f.All(code)...)
		}
	}
	return out
}

// Tagged returns the fields with this tag.
func (r Record) Tagged(tag string) []Field {
	var out []Field
	for _, f := range r.Fields {
		if f.Tag == tag {
			out = append(out, f)
		}
	}
	return out
}

// Control returns the value of a control field (001…009, 100 is a data field).
func (r Record) Control(tag string) string {
	for _, f := range r.Fields {
		if f.Tag == tag {
			return f.Value
		}
	}
	return ""
}

// IsText reports whether the record may describe printed text: it excludes the
// other types of record (leader 6: music, maps, video, sound, images,
// multimedia, objects). Some set records leave the type blank: they are kept.
func (r Record) IsText() bool {
	return len(r.Leader) < 7 || !strings.ContainsRune("cdefgijklmr", rune(r.Leader[6]))
}

// ── XML ─────────────────────────────────────────────────────────────

type sruResponse struct {
	Total       int         `xml:"numberOfRecords"`
	Next        int         `xml:"nextRecordPosition"`
	Records     []sruRecord `xml:"records>record"`
	Diagnostics []struct {
		Message string `xml:"message"`
		Details string `xml:"details"`
	} `xml:"diagnostics>diagnostic"`
}

type sruRecord struct {
	Identifier string `xml:"recordIdentifier"`
	Data       struct {
		Record struct {
			Leader   string `xml:"leader"`
			Controls []struct {
				Tag   string `xml:"tag,attr"`
				Value string `xml:",chardata"`
			} `xml:"controlfield"`
			Datafields []struct {
				Tag       string `xml:"tag,attr"`
				Subfields []struct {
					Code  string `xml:"code,attr"`
					Value string `xml:",chardata"`
				} `xml:"subfield"`
			} `xml:"datafield"`
		} `xml:"record"`
	} `xml:"recordData"`
}

// nonSorting strips the control characters that delimit the non-sorting part
// of a title ("\u0088Une \u0089Aventure").
var nonSorting = strings.NewReplacer("\u0088", "", "\u0089", "", "\u0098", "", "\u009c", "")

func (s sruRecord) record() Record {
	r := Record{ARK: strings.TrimSpace(s.Identifier), Leader: s.Data.Record.Leader}
	for _, c := range s.Data.Record.Controls {
		r.Fields = append(r.Fields, Field{Tag: c.Tag, Value: strings.TrimSpace(c.Value)})
	}
	for _, d := range s.Data.Record.Datafields {
		f := Field{Tag: d.Tag}
		for _, sf := range d.Subfields {
			f.Subfields = append(f.Subfields, Subfield{Code: sf.Code, Value: nonSorting.Replace(sf.Value)})
		}
		r.Fields = append(r.Fields, f)
	}
	return r
}
