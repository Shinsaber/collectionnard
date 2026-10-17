package connectors

import (
	"context"
	"strings"

	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/wikidata"
	"github.com/shinsaber/colec/internal/model"
)

// Languages of the i18n texts taken from Wikidata (those of the interface) and
// languages read as a fallback for names and original titles.
var (
	textLangs  = []string{"fr", "en"}
	fetchLangs = []string{"fr", "en", "mul", "ja", "ko", "zh"}
)

// Wikidata classes (P31) of the works the search accepts.
const (
	qMangaSeries   = "Q21198342"
	qManga         = "Q8274"
	qManhwa        = "Q562214"
	qManhua        = "Q754669"
	qWebtoon       = "Q74262765"
	qComicSeries   = "Q14406742"
	qGraphicNovel  = "Q725377"
	qComicStrip    = "Q21191134"
	qLimitedSeries = "Q3297186"
	qComicBook     = "Q1760610"
)

var (
	workClasses  = []string{qMangaSeries, qComicSeries, qGraphicNovel, qComicStrip, qWebtoon, qLimitedSeries, qComicBook, qManga, qManhwa, qManhua}
	mangaClasses = map[string]bool{qMangaSeries: true, qManga: true, qManhwa: true, qManhua: true, qWebtoon: true}
	comicClasses = map[string]bool{qComicSeries: true, qGraphicNovel: true, qComicStrip: true, qLimitedSeries: true, qComicBook: true}
)

// Countries (P495) used to qualify a work.
const (
	qJapan      = "Q17"
	qSouthKorea = "Q884"
	qChina      = "Q148"
	qTaiwan     = "Q865"
	qHongKong   = "Q8646"
	qFrance     = "Q142"
	qBelgium    = "Q31"
	qSwitzerl   = "Q39"
	qUSA        = "Q30"
)

// originLangs: language of the original title by country of origin.
var originLangs = map[string]string{qJapan: "ja", qSouthKorea: "ko", qChina: "zh", qTaiwan: "zh", qHongKong: "zh"}

// demographics maps the intended public (P2360) to the enum of Manga.
var demographics = map[string]string{
	"Q231302": "shonen", "Q242492": "shojo", "Q237338": "seinen", "Q503106": "josei", "Q478804": "kodomo",
}

// Wikidata properties.
const (
	pInstanceOf    = "P31"
	pAuthor        = "P50"
	pScreenwriter  = "P58"
	pIllustrator   = "P110"
	pFranchise     = "P8345"
	pCountryOrigin = "P495"
	pAudience      = "P2360"
	pTitle         = "P1476"
	pNativeName    = "P1559"
	pStart         = "P580"
	pInception     = "P571"
	pPublication   = "P577"
	pEnd           = "P582"
	pParts         = "P2635"
	pBnF           = "P268"
	pFamilyName    = "P734"
	pGivenName     = "P735"
	pCitizenship   = "P27"
	pBirth         = "P569"
	pISOAlpha2     = "P297"

	unitVolume = "Q1238720" // tankōbon / volume
)

// workType returns the item type of a work entity for the profile of c ("" when not one of its works).
func (c *Connector) workType(e *wikidata.Entity) string {
	if c.profile == profileSeries {
		if e.HasID(pInstanceOf, seriesClassesSet) {
			return typeSeries
		}
		return ""
	}
	if c.profile == profileFilms {
		if e.HasID(pInstanceOf, filmClassesSet) {
			return typeFilm
		}
		return ""
	}
	if c.profile == profileNovels {
		if e.HasID(pInstanceOf, novelClassesSet) {
			return typeNovel
		}
		return ""
	}
	return workType(e)
}

// workType returns the item type of a work entity ("" when not a work).
func workType(e *wikidata.Entity) string {
	switch {
	case e.HasID(pInstanceOf, mangaClasses):
		return typeManga
	case e.HasID(pInstanceOf, comicClasses):
		return typeComic
	}
	return ""
}

func wikidataKey(id string) string { return "wikidata:" + id }

// i18n keeps the texts of the interface languages.
func i18n(m map[string]string) map[string]any {
	out := map[string]any{}
	for _, l := range textLangs {
		if v := m[l]; v != "" {
			out[l] = v
		}
	}
	return out
}

// title is the i18n label of an entity; without a French or English label it
// falls back on the multilingual (or original script) label.
func title(e *wikidata.Entity) map[string]any {
	t := i18n(e.Labels)
	if len(t) == 0 {
		if l := e.Label("mul", "ja", "ko", "zh"); l != "" {
			t["en"] = l
		} else {
			for _, v := range e.Labels {
				t["en"] = v
				break
			}
		}
	}
	return t
}

func externalIDs(e *wikidata.Entity) map[string]any {
	ids := map[string]any{"wikidata": e.ID}
	if v, _ := e.Text(pBnF); v != "" {
		ark := "ark:/12148/cb" + v
		if _, ok := bnf.RecordNumber(ark); ok {
			ids["bnf_ark"] = ark
		}
	}
	return ids
}

// addWork adds the work, its universe and its creators.
func (p *planner) addWork(ctx context.Context, qid string) (string, error) {
	ents, err := p.c.WD.Entities(ctx, []string{qid}, fetchLangs, true)
	if err != nil {
		return "", err
	}
	e := ents[qid]
	if e == nil {
		return "", notFound("wikidata entity %s", qid)
	}
	typeName := workType(e)
	if typeName == "" {
		return "", notFound("wikidata entity %s is not a manga or comic work", qid)
	}
	work := p.add(&node{
		key: wikidataKey(qid), role: model.NodeWork, typeID: p.d.typeID(typeName),
		attrs: p.workAttrs(e, typeName), keys: wikidataKeys(e), selected: true,
		sourceURL: wikidata.EntityURL(qid),
	})

	// Creators: one relation per person, with all their roles.
	var people []string
	roles := map[string][]any{}
	for _, pr := range []struct{ prop, role string }{{pAuthor, "author"}, {pScreenwriter, "writer"}, {pIllustrator, "artist"}} {
		for _, id := range e.IDs(pr.prop) {
			if _, seen := roles[id]; !seen {
				people = append(people, id)
			}
			roles[id] = append(roles[id], pr.role)
		}
	}
	universes := e.IDs(pFranchise)
	if len(universes) > 1 {
		universes = universes[:1]
	}
	linked, err := p.c.WD.Entities(ctx, append(slicesClone(universes), people...), fetchLangs, true)
	if err != nil {
		return "", err
	}
	for _, id := range universes {
		u := linked[id]
		if u == nil {
			continue
		}
		p.add(&node{
			key: wikidataKey(id), role: model.NodeUniverse, typeID: p.d.typeID(typeUniverse),
			attrs: map[string]any{
				"title": title(u), "description": i18n(u.Descriptions),
				"external_ids": externalIDs(u), "sources": p.sources("wikidata"),
			},
			keys: wikidataKeys(u), selected: true, parent: work.key, sourceURL: wikidata.EntityURL(id),
		})
		p.link(relContainsWork, wikidataKey(id), work.key, nil)
	}

	names, err := p.personNames(ctx, people, linked)
	if err != nil {
		return "", err
	}
	for _, id := range people {
		person := linked[id]
		if person == nil {
			continue
		}
		attrs, err := p.personAttrs(ctx, person, names)
		if err != nil {
			return "", err
		}
		p.add(&node{
			key: wikidataKey(id), role: model.NodePerson, typeID: p.d.typeID(typePerson),
			attrs: attrs, keys: wikidataKeys(person), selected: true, parent: work.key,
			sourceURL: wikidata.EntityURL(id),
		})
		p.link(relCreatedBy, work.key, wikidataKey(id), map[string]any{"roles": roles[id]})
	}
	return work.key, nil
}

func wikidataKeys(e *wikidata.Entity) []matchKey {
	keys := []matchKey{{path: []string{"external_ids", "wikidata"}, value: e.ID}}
	if ark, ok := externalIDs(e)["bnf_ark"].(string); ok {
		keys = append(keys, matchKey{path: []string{"external_ids", "bnf_ark"}, value: ark})
	}
	return keys
}

func (p *planner) workAttrs(e *wikidata.Entity, typeName string) map[string]any {
	a := map[string]any{
		"title":        title(e),
		"description":  i18n(e.Descriptions),
		"external_ids": externalIDs(e),
		"sources":      p.sources("wikidata"),
	}
	if t := originalTitle(e); t != "" {
		a["original_title"] = t
	}
	if y, ok := e.Year(pStart, pInception, pPublication); ok {
		a["start_year"] = y
	}
	if y, ok := e.Year(pEnd); ok {
		a["end_year"] = y
		a["status"] = "completed"
	}
	if n, ok := e.Quantity(pParts, unitVolume); ok && n > 0 {
		a["volumes_vo"] = n
	}
	countries := map[string]bool{}
	for _, c := range e.IDs(pCountryOrigin) {
		countries[c] = true
	}
	classes := map[string]bool{}
	for _, c := range e.IDs(pInstanceOf) {
		classes[c] = true
	}
	if typeName == typeManga {
		switch {
		case classes[qWebtoon]:
			a["sub_type"] = "webtoon"
		case classes[qManhwa] || countries[qSouthKorea]:
			a["sub_type"] = "manhwa"
		case classes[qManhua] || countries[qChina] || countries[qTaiwan] || countries[qHongKong]:
			a["sub_type"] = "manhua"
		case classes[qMangaSeries] || classes[qManga] || countries[qJapan]:
			a["sub_type"] = "manga"
		}
		for _, id := range e.IDs(pAudience) {
			if d := demographics[id]; d != "" {
				a["demographic"] = d
				break
			}
		}
	} else {
		switch {
		case classes[qGraphicNovel]:
			a["sub_type"] = "graphic_novel"
		case classes[qComicStrip]:
			a["sub_type"] = "strip"
		case countries[qFrance] || countries[qBelgium] || countries[qSwitzerl]:
			a["sub_type"] = "franco_belge"
		case countries[qUSA]:
			a["sub_type"] = "comic"
		}
	}
	return dropEmpty(a)
}

// originalTitle: the title in the original language, when it is not the
// French or English one.
func originalTitle(e *wikidata.Entity) string {
	if t, lang := e.Text(pTitle); t != "" && lang != "fr" && lang != "en" {
		return t
	}
	if t, _ := e.Text(pNativeName); t != "" {
		return t
	}
	// Label in the language of the country of origin (Asian works only: a
	// French or Belgian work already has its original title in `title`).
	for _, c := range e.IDs(pCountryOrigin) {
		if l := originLangs[c]; l != "" {
			return e.Label(l)
		}
	}
	return ""
}

// personNames fetches the labels of the family and given names of people.
func (p *planner) personNames(ctx context.Context, people []string, linked map[string]*wikidata.Entity) (map[string]*wikidata.Entity, error) {
	var ids []string
	for _, id := range people {
		if e := linked[id]; e != nil {
			ids = append(ids, firstID(e, pFamilyName), firstID(e, pGivenName))
		}
	}
	ids = compactNonEmpty(ids)
	if len(ids) == 0 {
		return map[string]*wikidata.Entity{}, nil
	}
	return p.c.WD.Entities(ctx, ids, fetchLangs, false)
}

func (p *planner) personAttrs(ctx context.Context, e *wikidata.Entity, names map[string]*wikidata.Entity) (map[string]any, error) {
	label := e.Label("fr", "en", "mul")
	last, first := "", ""
	if n := names[firstID(e, pFamilyName)]; n != nil {
		last = n.Label("en", "fr", "mul")
	}
	if n := names[firstID(e, pGivenName)]; n != nil {
		first = n.Label("en", "fr", "mul")
	}
	if last == "" {
		// No family name statement: "Given Family" label, or a pen name kept whole.
		last = label
		if first != "" && strings.HasPrefix(label, first+" ") {
			last = strings.TrimPrefix(label, first+" ")
		} else if first == "" && strings.Contains(label, " ") {
			i := strings.LastIndex(label, " ")
			first, last = label[:i], label[i+1:]
		}
	}
	a := map[string]any{
		"last_name":    last,
		"first_name":   first,
		"biography":    i18n(e.Descriptions),
		"external_ids": externalIDs(e),
		"sources":      p.sources("wikidata"),
	}
	if t, _ := e.Text(pNativeName); t != "" && t != label && t != strings.TrimSpace(first+" "+last) {
		a["native_name"] = t
	}
	if d, ok := e.Date(pBirth); ok {
		a["birth_date"] = d
	}
	if c := firstID(e, pCitizenship); c != "" {
		codes, err := p.c.WD.StringClaims(ctx, c, pISOAlpha2)
		if err != nil {
			return nil, err
		}
		if len(codes) > 0 && len(codes[0]) == 2 {
			a["nationality"] = strings.ToUpper(codes[0])
		}
	}
	return dropEmpty(a), nil
}

func firstID(e *wikidata.Entity, prop string) string {
	if ids := e.IDs(prop); len(ids) > 0 {
		return ids[0]
	}
	return ""
}

func compactNonEmpty(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func slicesClone(s []string) []string { return append([]string(nil), s...) }

// dropEmpty removes empty strings and objects (no "first_name": "").
func dropEmpty(a map[string]any) map[string]any {
	for k, v := range a {
		if isEmpty(v) {
			delete(a, k)
		}
	}
	return a
}
