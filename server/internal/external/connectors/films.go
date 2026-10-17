package connectors

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/wikidata"
	"github.com/shinsaber/colec/internal/model"
)

// Films profile (P3-5, video.md): a film on Wikidata and its credits; its
// video releases at the BnF (DVD, Blu-ray, VHS of the legal deposit), listed
// from the BnF work record of the film (bib.work2bib) or found by EAN. A
// release found alone (barcode) brings its film: the Wikidata film carrying
// the BnF work number (P268), else a film built from the BnF record.

// Wikidata classes and properties of the films profile (labels checked on
// Wikidata on 2026-10-17, docs/sources-externes.md).
const (
	qFilm          = "Q11424"
	qAnimatedFilm  = "Q202866"
	qAnimatedFeat  = "Q29168811"
	qDocumentary   = "Q93204"
	qShortFilm     = "Q24862"
	qTelevisionFlm = "Q506240"

	pDirector        = "P57"
	pComposer        = "P86"
	pCinematographer = "P344"
	pCast            = "P161"
	pDuration        = "P2047"
	pOrigLanguage    = "P364"
	pIMDb            = "P345"

	unitMinute = "Q7727"
)

var (
	filmClasses    = []string{qFilm, qAnimatedFilm, qAnimatedFeat, qDocumentary, qShortFilm, qTelevisionFlm}
	filmClassesSet = map[string]bool{qFilm: true, qAnimatedFilm: true, qAnimatedFeat: true, qDocumentary: true, qShortFilm: true, qTelevisionFlm: true}
	filmKinds      = map[string]string{
		qAnimatedFilm: "animated", qAnimatedFeat: "animated", qDocumentary: "documentary", qShortFilm: "short", qTelevisionFlm: "tv_film",
	}
	filmGenres = map[string]string{
		"Q157443": "comedy", "Q130232": "drama", "Q859369": "comedy_drama", "Q1054574": "romance",
		"Q860626": "romantic_comedy", "Q118612349": "romantic_comedy", "Q188473": "action", "Q319221": "adventure",
		"Q202866": "animation", "Q29168811": "animation", "Q471839": "science_fiction", "Q157394": "fantasy",
		"Q200092": "horror", "Q2484376": "thriller", "Q959790": "crime", "Q1200678": "mystery", "Q369747": "war",
		"Q172980": "western", "Q93204": "documentary", "Q842256": "musical", "Q17013749": "historical",
		"Q645928": "biographical", "Q1361932": "family", "Q2143665": "children", "Q185867": "film_noir",
		"Q2297927": "spy", "Q1535153": "superhero", "Q846544": "disaster", "Q1339864": "sports",
		"Q1033891": "martial_arts", "Q1146335": "teen",
	}
	// filmCrew: Wikidata properties of the credits, in the order of the credits.
	filmCrew = []struct{ prop, role string }{
		{pDirector, "director"}, {pScreenwriter, "screenwriter"}, {pComposer, "composer"}, {pCinematographer, "cinematographer"},
	}
	// videoRoles: UNIMARC relator codes (7XX $4) of the credits of a video record.
	videoRoles = map[string]string{"300": "director", "690": "screenwriter", "230": "composer", "005": "actor", "160": "producer"}
)

// Limits of the credits: the first actors are selected, the next ones proposed unchecked.
const (
	maxCast            = 10
	selectedCast       = 5
	maxFilmEditionHits = 150
)

// ── Film ────────────────────────────────────────────────────────────

// addFilm adds a Wikidata film and its credits.
func (p *planner) addFilm(ctx context.Context, qid string) (string, error) {
	ents, err := p.c.WD.Entities(ctx, []string{qid}, fetchLangs, true)
	if err != nil {
		return "", err
	}
	e := ents[qid]
	if e == nil {
		return "", notFound("wikidata entity %s", qid)
	}
	if !e.HasID(pInstanceOf, filmClassesSet) {
		return "", notFound("wikidata entity %s is not a film", qid)
	}
	attrs, err := p.filmAttrs(ctx, e)
	if err != nil {
		return "", err
	}
	film := p.add(&node{
		key: wikidataKey(qid), role: model.NodeWork, typeID: p.d.typeID(typeFilm),
		attrs: attrs, keys: wikidataKeys(e), selected: true, sourceURL: wikidata.EntityURL(qid),
	})

	// Credits: one relation per person, with all their roles (director and screenwriter…).
	type credit struct {
		roles    []any
		order    int
		selected bool
	}
	credits := map[string]*credit{}
	var people []string
	addRole := func(id, role string, order int, selected bool) {
		c := credits[id]
		if c == nil {
			c = &credit{}
			credits[id] = c
			people = append(people, id)
		}
		if !slicesContainsAny(c.roles, role) {
			c.roles = append(c.roles, role)
		}
		if order > 0 && c.order == 0 {
			c.order = order
		}
		c.selected = c.selected || selected
	}
	for _, cr := range filmCrew {
		for _, id := range compactNonEmpty(e.IDs(cr.prop)) {
			addRole(id, cr.role, 0, true)
		}
	}
	for i, id := range compactNonEmpty(e.IDs(pCast)) {
		if i >= maxCast {
			break
		}
		addRole(id, "actor", i+1, i < selectedCast)
	}
	linked, err := p.c.WD.Entities(ctx, people, fetchLangs, true)
	if err != nil {
		return "", err
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
		pa, err := p.personAttrs(ctx, person, names)
		if err != nil {
			return "", err
		}
		c := credits[id]
		p.add(&node{
			key: wikidataKey(id), role: model.NodePerson, typeID: p.d.typeID(typePerson),
			attrs: pa, keys: wikidataKeys(person), selected: c.selected, parent: film.key,
			sourceURL: wikidata.EntityURL(id),
		})
		rel := map[string]any{"roles": c.roles}
		if c.order > 0 {
			rel["order"] = c.order
		}
		p.link(relCredited, film.key, wikidataKey(id), rel)
	}
	return film.key, nil
}

func (p *planner) filmAttrs(ctx context.Context, e *wikidata.Entity) (map[string]any, error) {
	ids := externalIDs(e)
	if v, _ := e.Text(pIMDb); strings.HasPrefix(v, "tt") {
		ids["imdb"] = v
	}
	a := map[string]any{
		"title":        title(e),
		"description":  i18n(e.Descriptions),
		"external_ids": ids,
		"sources":      p.sources("wikidata"),
		"kind":         "feature",
	}
	for _, id := range e.IDs(pInstanceOf) {
		if k := filmKinds[id]; k != "" {
			a["kind"] = k
			break
		}
	}
	if t := originalTitle(e); t != "" {
		a["original_title"] = t
	}
	if y, ok := e.EarliestYear(pPublication); ok {
		a["release_year"] = y
	}
	if m, ok := e.Quantity(pDuration, unitMinute); ok && m > 0 {
		a["duration"] = m
	}
	var genres []any
	for _, id := range e.IDs(pGenre) {
		if g := filmGenres[id]; g != "" && !slicesContainsAny(genres, g) {
			genres = append(genres, g)
		}
	}
	if len(genres) > 0 {
		a["genres"] = genres
	}
	var countries []any
	for _, c := range compactNonEmpty(e.IDs(pCountryOrigin)) {
		codes, err := p.c.WD.StringClaims(ctx, c, pISOAlpha2)
		if err != nil {
			return nil, err
		}
		if len(codes) > 0 && len(codes[0]) == 2 {
			countries = append(countries, strings.ToUpper(codes[0]))
		}
	}
	if len(countries) > 0 {
		a["countries"] = countries
	}
	if lang := firstID(e, pOrigLanguage); lang != "" {
		codes, err := p.c.WD.StringClaims(ctx, lang, pISO6391)
		if err != nil {
			return nil, err
		}
		if len(codes) > 0 && len(codes[0]) == 2 {
			a["original_language"] = strings.ToLower(codes[0])
		}
	}
	return dropEmpty(a), nil
}

// ── Video releases ──────────────────────────────────────────────────

// searchFilmEditions lists the video releases of a film (its BnF work record),
// or searches them by EAN or title; video records only.
func (c *Connector) searchFilmEditions(ctx context.Context, q model.ExternalEditionQuery) ([]model.ExternalEditionHit, error) {
	if q.ISBN != "" {
		return nil, InputError("this connector searches editions by title, barcode or film")
	}
	if q.Work != "" && !wikidata.IsQID(q.Work) {
		return nil, InputError("work must be a Wikidata identifier (Q…)")
	}
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	var recs []bnf.Record
	switch {
	case q.Work != "":
		ents, err := c.WD.Entities(ctx, []string{q.Work}, fetchLangs, true)
		if err != nil {
			return nil, err
		}
		e := ents[q.Work]
		if e == nil {
			return nil, notFound("wikidata entity %s", q.Work)
		}
		number := ""
		if ark, _ := externalIDs(e)["bnf_ark"].(string); ark != "" {
			number, _ = bnf.RecordNumber(ark)
		}
		if number != "" {
			recs, _, err = c.BnF.Search(ctx, "bib.work2bib all "+bnf.Quote(number), maxFilmEditionHits)
		} else {
			// Film unknown to the BnF: search its French (or English) title.
			recs, _, err = c.BnF.Search(ctx, videoQuery("bib.title all "+bnf.Quote(e.Label("fr", "en"))), maxEditionHits)
		}
		if err != nil {
			return nil, err
		}
	case q.Barcode != "":
		if recs, _, err = c.BnF.Search(ctx, "bib.ean all "+bnf.Quote(q.Barcode), maxEditionHits); err != nil {
			return nil, err
		}
	default:
		if recs, _, err = c.BnF.Search(ctx, videoQuery("bib.title all "+bnf.Quote(q.Title)), maxEditionHits); err != nil {
			return nil, err
		}
	}
	hits := []model.ExternalEditionHit{}
	var arks []string
	for i := range recs {
		r := &recs[i]
		if r.ARK == "" || !isVideo(r) || slicesContains(arks, r.ARK) {
			continue
		}
		v := parseVideo(r)
		h := model.ExternalEditionHit{
			ID: r.ARK, BnfArk: r.ARK, Title: editionTitle(r), Publishers: videoPublisherNames(r),
			Format: v.label(), Barcode: v.ean, Collection: collectionOf(r), SourceURL: bnf.RecordURL(r.ARK),
		}
		if v.year > 0 {
			h.StartYear = &v.year
		}
		hits = append(hits, h)
		arks = append(arks, r.ARK)
	}
	// Most recent first (DVD and Blu-ray before the VHS of the same film).
	sort.SliceStable(hits, func(i, j int) bool {
		yi, yj := 0, 0
		if hits[i].StartYear != nil {
			yi = *hits[i].StartYear
		}
		if hits[j].StartYear != nil {
			yj = *hits[j].StartYear
		}
		return yi > yj
	})
	existing, err := c.existing(ctx, map[uuid.UUID][]string{d.typeID(typeVideoRelease): arks}, []string{"external_ids", "bnf_ark"})
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].ExistingItemID = existing[hits[i].ID]
	}
	return hits, nil
}

// videoQuery restricts a CQL query to the video records (BnF document type « h »).
func videoQuery(cql string) string { return cql + ` and bib.doctype any "h"` }

// isVideo: a projected medium record (leader 6 « g »), or a record without type
// whose extent names a video medium (some box sets: « 21 DVD vidéo »).
func isVideo(r *bnf.Record) bool {
	if len(r.Leader) > 6 && r.Leader[6] != ' ' {
		return r.Leader[6] == 'g'
	}
	f := parseVideo(r).format
	return f != "" && f != "other"
}

// video is what a BnF video record says of its medium (215, 073, 101, 100).
type video struct {
	format              string // enum of VideoRelease
	name                string // « DVD », « Blu-ray »… for the hits
	discs, runtime      int
	region, aspectRatio string
	ean                 string
	year                int
	audio, subtitles    []any
}

var (
	reZone     = regexp.MustCompile(`(?i)\bzone (\w+)`)
	reDuration = regexp.MustCompile(`\((?:(\d+) h)?\s*(?:(\d+) min)?\)`)
	reBD       = regexp.MustCompile(`^\s*[0-9]*\s*bd\b`) // « 16 BD (50 h) » : Blu-ray sans « vidéo »
	reLeading  = regexp.MustCompile(`^\s*(\d+)\s`)
)

func parseVideo(r *bnf.Record) video {
	extent := r.Get("215", "a")
	low := strings.ToLower(extent)
	v := video{ean: external.NormalizeBarcode(r.Get("073", "a"))}
	switch {
	case strings.Contains(low, "4k"), strings.Contains(low, "ultra hd"), strings.Contains(low, "uhd"):
		v.format, v.name = "uhd_blu_ray", "Blu-ray 4K"
	case strings.Contains(low, "bd vidéo"), strings.Contains(low, "blu-ray"), reBD.MatchString(low):
		v.format, v.name = "blu_ray", "Blu-ray"
	case strings.Contains(low, "hd dvd"):
		v.format, v.name = "hd_dvd", "HD DVD"
	case strings.Contains(low, "dvd"):
		v.format, v.name = "dvd", "DVD"
	case strings.Contains(low, "vhs"), strings.Contains(low, "cass. vidéo"):
		v.format, v.name = "vhs", "VHS"
	case strings.Contains(low, "vidéodisque"), strings.Contains(low, "laserdisc"):
		v.format, v.name = "laserdisc", "LaserDisc"
	case extent != "":
		v.format = "other"
	}
	if m := reLeading.FindStringSubmatch(extent); m != nil {
		v.discs, _ = strconv.Atoi(m[1])
	}
	if m := reZone.FindStringSubmatch(extent); m != nil {
		v.region = strings.ToUpper(m[1])
	}
	for _, m := range reDuration.FindAllStringSubmatch(extent, -1) {
		h, _ := strconv.Atoi(m[1])
		mn, _ := strconv.Atoi(m[2])
		if h*60+mn > 0 {
			v.runtime = h*60 + mn
		}
	}
	if c := strings.TrimSpace(r.Get("215", "c")); strings.HasPrefix(c, "16/9") || strings.HasPrefix(c, "4/3") {
		v.aspectRatio = c[:4]
	}
	if start, _ := bnf.Years(r.Get("100", "a")); start > 0 {
		v.year = start
	}
	for _, f := range r.Tagged("101") {
		v.audio = appendLanguages(v.audio, f.All("a"))
		v.subtitles = appendLanguages(v.subtitles, f.All("j"))
	}
	return v
}

// label: « 2 DVD · zone 2 », « Blu-ray », « VHS ».
func (v video) label() string {
	name := v.name
	if name == "" {
		return ""
	}
	if v.discs > 1 {
		name = strconv.Itoa(v.discs) + " " + name
	}
	if v.region != "" {
		name += " · zone " + v.region
	}
	return name
}

func appendLanguages(out []any, codes []string) []any {
	for _, code := range codes {
		l := bnf.Language(code)
		if l == "" {
			switch code {
			case "", "xxx", "und", "zxx", "mul", "mis":
				continue
			}
			l = code
		}
		if !slicesContainsAny(out, l) {
			out = append(out, l)
		}
	}
	return out
}

// videoPublisher: a publisher or distributor of a video record.
type videoPublisher struct {
	name, ark string
	roles     []any
}

// videoPublishers reads 712 (relator 650 publisher, 310 distributor), else the
// names of 210 / 214 $c without their bracketed roles (« [TF1 vidéo [éd.]] »).
func videoPublishers(r *bnf.Record) []videoPublisher {
	var out []videoPublisher
	index := map[string]int{}
	add := func(name, ark, role string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		k := publisherKey(name)
		i, ok := index[k]
		if !ok {
			i = len(out)
			index[k] = i
			out = append(out, videoPublisher{name: name, ark: ark})
		}
		if !slicesContainsAny(out[i].roles, role) {
			out[i].roles = append(out[i].roles, role)
		}
	}
	for _, f := range r.Tagged("712") {
		role := map[string]string{"650": "publisher", "310": "distributor"}[f.Get("4")]
		if role != "" {
			add(f.Get("a"), bnf.ARK(f.Get("3")), role)
		}
	}
	if len(out) == 0 {
		for _, name := range publishers(r) {
			add(cleanPublisher(name), "", "publisher")
		}
	}
	return out
}

func videoPublisherNames(r *bnf.Record) []string {
	out := []string{}
	for _, p := range videoPublishers(r) {
		out = append(out, p.name)
	}
	return out
}

// cleanPublisher: « [TF1 vidéo [éd., distrib.]] » → « TF1 vidéo ».
func cleanPublisher(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(s, "["))
	if i := strings.Index(s, "["); i > 0 {
		s = s[:i]
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "]"))
}

// addVideoRelease adds a video release, its publishers and, without a chosen
// film, the film it records.
func (p *planner) addVideoRelease(ctx context.Context, ark, workKey string) error {
	rec, err := p.videoRecord(ctx, ark)
	if err != nil {
		return err
	}
	if workKey == "" {
		if workKey, err = p.filmOfRecord(ctx, rec); err != nil {
			return err
		}
	}
	p.addVideoReleaseNode(rec, ark, rec.Get("200", "a"), workKey)
	return nil
}

// videoRecord reads a BnF record that must describe a video.
func (p *planner) videoRecord(ctx context.Context, ark string) (*bnf.Record, error) {
	number, _ := bnf.RecordNumber(ark)
	rec, err := p.fetchRecord(ctx, number)
	if err != nil {
		return nil, err
	}
	if !isVideo(rec) {
		return nil, notFound("bnf record %s is not a video release", number)
	}
	return rec, nil
}

// addVideoReleaseNode adds the release of a video record (its medium,
// languages, contents…), released_as from the work and its publishers.
func (p *planner) addVideoReleaseNode(rec *bnf.Record, ark, title, workKey string) *node {
	v := parseVideo(rec)
	attrs := map[string]any{
		"title":        title,
		"external_ids": map[string]any{"bnf_ark": ark},
		"sources":      p.sources("bnf"),
	}
	set := func(k string, val any, ok bool) {
		if ok {
			attrs[k] = val
		}
	}
	set("format", v.format, v.format != "")
	set("disc_count", v.discs, v.discs > 0 && v.format != "vhs")
	set("region", v.region, v.region != "")
	set("runtime", v.runtime, v.runtime > 0)
	set("aspect_ratio", v.aspectRatio, v.aspectRatio != "")
	set("ean", v.ean, v.ean != "")
	set("release_year", v.year, v.year > 0)
	set("audio_languages", v.audio, len(v.audio) > 0)
	set("subtitle_languages", v.subtitles, len(v.subtitles) > 0)
	edition := strings.TrimSpace(rec.Get("200", "e"))
	if edition == "" {
		edition = strings.TrimSpace(rec.Get("205", "a"))
	}
	set("edition", edition, edition != "")
	if c := rec.Get("225", "a"); c != "" {
		attrs["collection"] = c
		set("collection_number", rec.Get("225", "v"), rec.Get("225", "v") != "")
	}
	var notes []string
	for _, n := range rec.All("300", "a") {
		if strings.HasPrefix(n, "Contient") {
			notes = append(notes, n)
		}
	}
	if extra := rec.All("215", "e"); len(extra) > 0 {
		notes = append(notes, "Avec : "+strings.Join(extra, ", "))
	}
	set("notes", strings.Join(notes, "\n"), len(notes) > 0)
	pubs := videoPublishers(rec)
	if len(pubs) > 0 {
		attrs["publisher"] = pubs[0].name
	}

	release := p.add(&node{
		key: bnfKey(ark), role: model.NodeEdition, typeID: p.d.typeID(typeVideoRelease),
		attrs: attrs, keys: bnfKeys(ark), selected: true, parent: workKey, sourceURL: bnf.RecordURL(ark),
	})
	if workKey != "" {
		p.link(relReleasedAs, workKey, release.key, nil)
	}
	for _, pub := range pubs {
		pa := map[string]any{"name": pub.name, "sources": p.sources("bnf")}
		keys := []matchKey{{path: []string{"name"}, value: pub.name, fold: true}}
		if pub.ark != "" {
			pa["external_ids"] = map[string]any{"bnf_ark": pub.ark}
			keys = append(bnfKeys(pub.ark), keys...)
		}
		node := p.add(&node{
			key: publisherKey(pub.name), role: model.NodePublisher, typeID: p.d.typeID(typePublisher),
			attrs: pa, keys: keys, selected: true, parent: release.key, sourceURL: release.sourceURL,
		})
		p.link(relVideoPublishedBy, release.key, node.key, map[string]any{"roles": pub.roles})
	}
	return release
}

// filmOfRecord returns the film a release records (500 $3, BnF work record):
// the Wikidata film carrying that work (P268), else a film built from the
// record and its credits (7XX). "" when the record names no work.
func (p *planner) filmOfRecord(ctx context.Context, rec *bnf.Record) (string, error) {
	workNumber := rec.Get("500", "3")
	workARK := bnf.ARK(workNumber)
	if workARK == "" {
		return "", nil
	}
	if key := bnfKey(workARK); p.byKey[key] != nil {
		return key, nil
	}
	qids, err := p.c.WD.Search(ctx, "haswbstatement:"+pBnF+"="+strings.TrimPrefix(bnf.ShortARK(workARK), "cb"), filmClasses, 1)
	if err != nil {
		return "", err
	}
	if len(qids) > 0 {
		if key := wikidataKey(qids[0]); p.byKey[key] != nil {
			return key, nil
		}
		return p.addFilm(ctx, qids[0])
	}
	// Film unknown to Wikidata: the BnF work title and the credits of the record.
	filmTitle := rec.Get("500", "a")
	if filmTitle == "" {
		filmTitle = rec.Get("200", "a")
	}
	lang := novelLanguage(rec)
	if l := bnf.Language(rec.Get("101", "c")); l != "" {
		lang = l
	}
	if lang == "" || len(lang) != 2 {
		lang = "fr"
	}
	attrs := map[string]any{
		"title":        map[string]any{lang: filmTitle},
		"external_ids": map[string]any{"bnf_ark": workARK},
		"sources":      p.sources("bnf"),
	}
	if l := bnf.Language(rec.Get("101", "c")); l != "" {
		attrs["original_language"] = l
	}
	film := p.add(&node{
		key: bnfKey(workARK), role: model.NodeWork, typeID: p.d.typeID(typeFilm),
		attrs: attrs, keys: bnfKeys(workARK), selected: true, sourceURL: bnf.RecordURL(workARK),
	})
	p.addVideoCredits(rec, film)
	return film.key, nil
}

// addVideoCredits adds the people of a video record (7XX with a relator code of
// videoRoles) to its film, one relation per person.
func (p *planner) addVideoCredits(rec *bnf.Record, film *node) {
	roles := map[string][]any{}
	people := map[string]map[string]any{}
	var order []string
	actors := 0
	actorOrder := map[string]int{}
	for _, tag := range []string{"700", "701", "702"} {
		for _, f := range rec.Tagged(tag) {
			last := strings.TrimSpace(f.Get("a"))
			if last == "" {
				continue
			}
			first := strings.TrimSpace(f.Get("b"))
			key := "bnf-person:" + fold(first+" "+last)
			attrs := map[string]any{"last_name": last, "sources": p.sources("bnf")}
			if first != "" {
				attrs["first_name"] = first
			}
			if n := f.Get("3"); bnf.ARK(n) != "" {
				key = "bnf:" + bnf.ShortARK(bnf.ARK(n))
				attrs["external_ids"] = map[string]any{"bnf_ark": bnf.ARK(n)}
			}
			for _, code := range f.All("4") {
				role := videoRoles[code]
				if role == "" {
					continue
				}
				if _, ok := people[key]; !ok {
					order = append(order, key)
					people[key] = attrs
				}
				if !slicesContainsAny(roles[key], role) {
					roles[key] = append(roles[key], role)
				}
				if role == "actor" && actorOrder[key] == 0 {
					actors++
					actorOrder[key] = actors
				}
			}
		}
	}
	for _, key := range order {
		attrs := people[key]
		var keys []matchKey
		if ids, ok := attrs["external_ids"].(map[string]any); ok {
			keys = bnfKeys(ids["bnf_ark"].(string))
		}
		person := p.add(&node{
			key: key, role: model.NodePerson, typeID: p.d.typeID(typePerson),
			attrs: attrs, keys: keys, selected: true, parent: film.key, sourceURL: film.sourceURL,
		})
		rel := map[string]any{"roles": roles[key]}
		if n := actorOrder[key]; n > 0 {
			rel["order"] = n
		}
		p.link(relCredited, film.key, person.key, rel)
	}
}
