package connectors

import (
	"net/url"
	"strings"

	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/musicbrainz"
	"github.com/shinsaber/colec/internal/external/tvmaze"
	"github.com/shinsaber/colec/internal/external/wikidata"
)

// Links (x-widget: links) to the records an imported item comes from: its
// Wikidata entity and Wikipedia articles, BnF record, MusicBrainz, TVmaze and
// IMDb pages. Added to every node whose type has a `links` field; an existing
// item keeps its links and gains the missing ones (by URL).

// musicbrainzKinds: path of a MusicBrainz page by item type.
var musicbrainzKinds = map[string]string{
	typeAlbum: "release-group", typeRelease: "release", typeTrack: "recording", typeArtist: "artist", typeLabel: "label",
}

// tvmazeKinds: path of a TVmaze page by item type.
var tvmazeKinds = map[string]string{typeSeries: "shows", typeSeason: "seasons", typeEpisode: "episodes"}

// sourceLabels: label of a link by host, for the source page of a node.
var sourceLabels = map[string]string{
	"www.wikidata.org": "Wikidata", "catalogue.bnf.fr": "BnF", "musicbrainz.org": "MusicBrainz",
	"www.tvmaze.com": "TVmaze", "www.imdb.com": "IMDb",
}

func link(label, u string) map[string]any { return map[string]any{"label": label, "url": u} }

// typeName returns the technical name of a node type ("" if unknown).
func (p *planner) typeName(n *node) string {
	for name, t := range p.d.types {
		if t.ID == n.typeID {
			return name
		}
	}
	return ""
}

// sourceLinks lists the links of a node: identifiers first, then its source page.
func (p *planner) sourceLinks(n *node) []any {
	var out []any
	seen := map[string]bool{}
	add := func(label, u string) {
		if u != "" && !seen[u] {
			seen[u] = true
			out = append(out, link(label, u))
		}
	}
	ids, _ := n.attrs["external_ids"].(map[string]any)
	str := func(k string) string { s, _ := ids[k].(string); return s }
	if q := str("wikidata"); wikidata.IsQID(q) {
		add("Wikidata", wikidata.EntityURL(q))
		if p.c.WD != nil {
			sl := p.c.WD.Sitelinks(q)
			add("Wikipédia (fr)", sl["frwiki"])
			add("Wikipedia (en)", sl["enwiki"])
		}
	}
	if a := str("bnf_ark"); a != "" {
		if _, ok := bnf.RecordNumber(a); ok {
			add("BnF", bnf.RecordURL(a))
		}
	}
	typ := p.typeName(n)
	if m := str("musicbrainz"); musicbrainz.IsMBID(m) && musicbrainzKinds[typ] != "" {
		add("MusicBrainz", musicbrainz.EntityURL(musicbrainzKinds[typ], m))
	}
	if t := str("tvmaze"); t != "" && tvmazeKinds[typ] != "" {
		if typ == typeSeries {
			add("TVmaze", tvmaze.ShowURL(atoiOr(t)))
		} else {
			add("TVmaze", "https://www.tvmaze.com/"+tvmazeKinds[typ]+"/"+url.PathEscape(t))
		}
	}
	if i := str("imdb"); strings.HasPrefix(i, "tt") {
		add("IMDb", "https://www.imdb.com/title/"+url.PathEscape(i)+"/")
	}
	if u, err := url.Parse(n.sourceURL); err == nil && sourceLabels[u.Host] != "" {
		add(sourceLabels[u.Host], n.sourceURL)
	}
	return out
}

func atoiOr(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// addSourceLinks sets the links of every node whose type has a `links` field
// (attributes complete: called once the plan is built, before matching).
func (p *planner) addSourceLinks() {
	for _, n := range p.nodes {
		props, _ := p.d.schemas[n.typeID]["properties"].(map[string]any)
		if _, ok := props["links"]; !ok {
			continue
		}
		if links := p.sourceLinks(n); len(links) > 0 {
			n.attrs["links"] = mergeLinks(n.attrs["links"], links)
		}
	}
}

// mergeLinks adds to existing the links whose URL it does not have yet.
func mergeLinks(existing, source any) any {
	el, _ := existing.([]any)
	sl, ok := source.([]any)
	if !ok {
		return existing
	}
	seen := map[any]bool{}
	for _, e := range el {
		if m, ok := e.(map[string]any); ok {
			seen[m["url"]] = true
		}
	}
	out := append([]any{}, el...)
	for _, s := range sl {
		if m, ok := s.(map[string]any); ok && !seen[m["url"]] {
			out = append(out, s)
			seen[m["url"]] = true
		}
	}
	return out
}
