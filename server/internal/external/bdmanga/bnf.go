package bdmanga

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/model"
)

func bnfKey(ark string) string { return "bnf:" + bnf.ShortARK(ark) }

func bnfKeys(ark string) []matchKey {
	return []matchKey{{path: []string{"external_ids", "bnf_ark"}, value: ark}}
}

// fetchRecord reads one catalogue record by number.
func (p *planner) fetchRecord(ctx context.Context, number string) (*bnf.Record, error) {
	recs, _, err := p.c.BnF.Search(ctx, "bib.recordid all "+bnf.Quote(number), 1)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, notFound("bnf record %s", number)
	}
	return &recs[0], nil
}

// editionTitle: title proper, plus the other title information ("full color").
func editionTitle(r *bnf.Record) string {
	t := r.Get("200", "a")
	if e := r.Get("200", "e"); e != "" {
		t += " : " + e
	}
	return t
}

// publishers of a record (210 / 214 $c), without duplicates.
func publishers(r *bnf.Record) []string {
	var out []string
	seen := map[string]bool{}
	for _, tag := range []string{"210", "214"} {
		for _, name := range r.All(tag, "c") {
			name = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(name, ","), ";"))
			if k := publisherKey(name); name != "" && !seen[k] {
				seen[k] = true
				out = append(out, name)
			}
		}
	}
	return out
}

func publisherKey(name string) string {
	return "publisher:" + strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func recordLanguage(r *bnf.Record) string {
	if l := bnf.Language(r.Get("101", "a")); l != "" {
		return l
	}
	return "fr" // legal deposit of the French national library
}

// addEdition adds a BnF multi-volume set, its publishers and its volumes.
func (p *planner) addEdition(ctx context.Context, ark, workKey string) error {
	number, _ := bnf.RecordNumber(ark)
	set, err := p.fetchRecord(ctx, number)
	if err != nil {
		return err
	}
	lang := recordLanguage(set)
	attrs := map[string]any{
		"title":        map[string]any{lang: editionTitle(set)},
		"language":     lang,
		"external_ids": map[string]any{"bnf_ark": ark},
		"sources":      p.sources("bnf"),
	}
	start, end := bnf.Years(set.Get("100", "a"))
	if start > 0 {
		attrs["start_year"] = start
	}
	if end > 0 {
		attrs["end_year"] = end
	}
	edition := p.add(&node{
		key: bnfKey(ark), role: model.NodeEdition, typeID: p.d.typeID(typeEdition),
		attrs: attrs, keys: bnfKeys(ark), selected: true, parent: workKey, sourceURL: bnf.RecordURL(ark),
	})
	if workKey != "" {
		p.link(relPublishedAs, workKey, edition.key, nil)
	}
	for _, name := range publishers(set) {
		pub := p.add(&node{
			key: publisherKey(name), role: model.NodePublisher, typeID: p.d.typeID(typePublisher),
			attrs: map[string]any{"name": name, "sources": p.sources("bnf")},
			keys:  []matchKey{{path: []string{"name"}, value: name, fold: true}}, selected: true,
			parent: edition.key, sourceURL: bnf.RecordURL(ark),
		})
		p.link(relEditedBy, edition.key, pub.key, nil)
	}

	// A multi-volume set (461 links) or an editorial collection (410 links).
	vols, _, err := p.c.BnF.Search(ctx, "bib.ens2bib all "+bnf.Quote(number)+" or bib.col2bib all "+bnf.Quote(number), maxVolumes)
	if err != nil {
		return err
	}
	p.addVolumes(vols, number, lang, edition)
	return nil
}

type volume struct {
	rec    bnf.Record
	number int
	ok     bool // number found
	year   int
}

func (p *planner) addVolumes(recs []bnf.Record, setNumber, lang string, edition *node) {
	vols := make([]volume, 0, len(recs))
	for _, r := range recs {
		if r.ARK == "" {
			continue
		}
		n, ok := volumeNumber(r, setNumber)
		y, _ := bnf.Years(r.Get("100", "a"))
		vols = append(vols, volume{rec: r, number: n, ok: ok, year: y})
	}
	// By number, then oldest record first: the first of each number is the
	// original printing, the next ones are reprints.
	sort.SliceStable(vols, func(i, j int) bool {
		a, b := vols[i], vols[j]
		if a.ok != b.ok {
			return a.ok
		}
		if a.number != b.number {
			return a.number < b.number
		}
		if a.year != b.year {
			return a.year < b.year
		}
		return a.rec.ARK < b.rec.ARK
	})
	seen := map[int]bool{}
	for _, v := range vols {
		r := v.rec
		attrs := map[string]any{
			"number":       v.number,
			"title":        map[string]any{lang: volumeTitle(r, edition, v)},
			"external_ids": map[string]any{"bnf_ark": r.ARK},
			"sources":      p.sources("bnf"),
		}
		if s := r.Get("200", "e"); s != "" {
			attrs["subtitle"] = s
		}
		isbn := bnf.NormalizeISBN(r.Get("073", "a"))
		if isbn == "" {
			isbn = bnf.NormalizeISBN(r.Get("010", "a"))
		}
		if isbn != "" {
			attrs["isbn"] = isbn
		}
		if n, ok := bnf.PageCount(r.Get("215", "a")); ok {
			attrs["page_count"] = n
		}
		if price, ok := bnf.PriceEUR(r.Get("010", "d")); ok {
			attrs["retail_price"] = price
		}
		keys := bnfKeys(r.ARK)
		if isbn != "" {
			keys = append(keys, matchKey{path: []string{"isbn"}, value: isbn})
		}
		n := &node{
			key: bnfKey(r.ARK), role: model.NodeVolume, typeID: p.d.typeID(typeVolume),
			attrs: attrs, keys: keys, selected: true, parent: edition.key, sourceURL: bnf.RecordURL(r.ARK),
		}
		switch {
		case !v.ok:
			n.selected, n.note = false, model.NoteNoNumber
		case seen[v.number]:
			n.selected, n.note = false, model.NoteReprint
		}
		seen[v.number] = true
		n = p.add(n)
		p.link(relContainsVolume, edition.key, n.key, map[string]any{"order": v.number}).orderOfTarget = true
	}
}

// setLinks are the links of a volume to its edition: multi-volume set (461)
// or editorial collection (410).
func setLinks(r bnf.Record) []bnf.Field {
	return append(r.Tagged("461"), r.Tagged("410")...)
}

// volumeNumber: the number in the edition ($v of the link to it), then the
// series statement (225$v), then the part number of the title (200$h).
func volumeNumber(r bnf.Record, setNumber string) (int, bool) {
	for _, f := range setLinks(r) {
		if f.Get("0") == setNumber {
			if n, ok := bnf.FirstNumber(f.Get("v")); ok {
				return n, true
			}
		}
	}
	for _, s := range []string{r.Get("225", "v"), r.Get("200", "h")} {
		if n, ok := bnf.FirstNumber(s); ok {
			return n, true
		}
	}
	return 0, false
}

// volumeTitle: the name of the part (200$i) when the title proper is the
// series name, else the title proper; failing both, "<edition> <number>".
func volumeTitle(r bnf.Record, edition *node, v volume) string {
	if t := r.Get("200", "i"); t != "" {
		return t
	}
	if t := r.Get("200", "a"); t != "" {
		return t
	}
	t := stringAt(edition.attrs, []string{"title", recordLanguage(&r)})
	if v.ok {
		t += " " + strconv.Itoa(v.number)
	}
	return strings.TrimSpace(t)
}
