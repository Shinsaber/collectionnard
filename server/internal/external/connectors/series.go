package connectors

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/tvmaze"
	"github.com/shinsaber/colec/internal/external/wikidata"
	"github.com/shinsaber/colec/internal/model"
)

// Series profile (P3-6, video.md § Séries): a TV series on Wikidata, its
// broadcaster, seasons and episodes (TVmaze for the seasons Wikidata leaves
// without episodes); its box sets at the BnF (video records found by title or
// EAN), linked to the episodes of the seasons they contain (200 $h).

// Wikidata classes and properties of the series profile (labels checked on
// Wikidata on 2026-10-17).
const (
	qTVSeries        = "Q5398426"
	qMiniseries      = "Q1259759"
	qWebSeries       = "Q526877"
	qAnimeSeries     = "Q63952888"
	qAnimatedTV      = "Q117467246"
	qAnimatedSeries  = "Q581714"
	qStreamingVideo  = "Q59152282"
	qCableChannel    = "Q5009242"
	pHasParts        = "P527"
	pBroadcaster     = "P449"
	pCreator         = "P170"
	pSeasonCount     = "P2437"
	pEpisodeCount    = "P1113"
	pTVmazeSeries    = "P8600"
	pSeriesOrdinal   = "P1545"
	maxSeriesSeasons = 60
)

var (
	seriesClasses    = []string{qTVSeries, qMiniseries, qWebSeries, qAnimeSeries, qAnimatedTV, qAnimatedSeries}
	seriesClassesSet = map[string]bool{qTVSeries: true, qMiniseries: true, qWebSeries: true, qAnimeSeries: true, qAnimatedTV: true, qAnimatedSeries: true}
)

// ── Series ──────────────────────────────────────────────────────────

// addSeries adds a Wikidata series, its credits, broadcasters, seasons and episodes.
func (p *planner) addSeries(ctx context.Context, qid string) (string, error) {
	ents, err := p.c.WD.Entities(ctx, []string{qid}, fetchLangs, true)
	if err != nil {
		return "", err
	}
	e := ents[qid]
	if e == nil {
		return "", notFound("wikidata entity %s", qid)
	}
	if !e.HasID(pInstanceOf, seriesClassesSet) {
		return "", notFound("wikidata entity %s is not a TV series", qid)
	}
	attrs, err := p.seriesAttrs(ctx, e)
	if err != nil {
		return "", err
	}
	series := p.add(&node{
		key: wikidataKey(qid), role: model.NodeWork, typeID: p.d.typeID(typeSeries),
		attrs: attrs, keys: wikidataKeys(e), selected: true, sourceURL: wikidata.EntityURL(qid),
	})

	// Seasons (P527, ordered by P1545), their episodes, the broadcasters of both.
	type part struct {
		id    string
		order int
	}
	var seasonIDs []part
	for _, st := range e.Claims[pHasParts] {
		if st.Value.ID == "" {
			continue
		}
		n := 0
		if q := st.Qualifiers[pSeriesOrdinal]; len(q) > 0 {
			n, _ = strconv.Atoi(q[0].Text)
		}
		seasonIDs = append(seasonIDs, part{st.Value.ID, n})
	}
	if len(seasonIDs) > maxSeriesSeasons {
		seasonIDs = seasonIDs[:maxSeriesSeasons]
	}
	ids := make([]string, len(seasonIDs))
	for i, s := range seasonIDs {
		ids[i] = s.id
	}
	seasons, err := p.c.WD.Entities(ctx, ids, fetchLangs, true)
	if err != nil {
		return "", err
	}
	var episodeIDs []string
	for _, s := range seasonIDs {
		if se := seasons[s.id]; se != nil {
			for _, st := range se.Claims[pHasParts] {
				if st.Value.ID != "" {
					episodeIDs = append(episodeIDs, st.Value.ID)
				}
			}
		}
	}
	people := p.seriesPeople(e)
	broadcasters := compactNonEmpty(e.IDs(pBroadcaster))
	for _, s := range seasons {
		broadcasters = compactNonEmpty(append(broadcasters, s.IDs(pBroadcaster)...))
	}
	linked, err := p.c.WD.Entities(ctx, compactNonEmpty(append(append(slicesClone(episodeIDs), broadcasters...), people.ids...)), fetchLangs, true)
	if err != nil {
		return "", err
	}

	defaultBroadcast := ""
	if len(broadcasters) > 0 {
		defaultBroadcast = p.broadcast(series, linked[e.IDs(pBroadcaster)[0]], attrs)
	}
	withEpisodes := map[int]string{} // season number → season key, when Wikidata lists its episodes
	seasonKeys := map[int]string{}
	for i, s := range seasonIDs {
		se := seasons[s.id]
		if se == nil {
			continue
		}
		number := s.order
		if number == 0 {
			number = i + 1
		}
		sa := map[string]any{
			"name":         seasonName(number),
			"number":       number,
			"external_ids": map[string]any{"wikidata": se.ID},
			"sources":      p.sources("wikidata"),
		}
		if y, ok := se.Year(pStart, pPublication); ok {
			sa["year"] = y
		}
		if n, ok := se.Quantity(pEpisodeCount, "1"); ok && n > 0 {
			sa["episode_count"] = n
		}
		parent := defaultBroadcast
		if b := firstID(se, pBroadcaster); b != "" && linked[b] != nil {
			parent = p.broadcast(series, linked[b], attrs)
		}
		season := p.add(&node{
			key: wikidataKey(se.ID), role: model.NodeSeason, typeID: p.d.typeID(typeSeason),
			attrs: sa, keys: wikidataKeys(se), selected: true, parent: series.key, sourceURL: wikidata.EntityURL(se.ID),
		})
		seasonKeys[number] = season.key
		if parent != "" {
			p.link(relContainsSeason, parent, season.key, nil)
		}
		for j, st := range se.Claims[pHasParts] {
			ep := linked[st.Value.ID]
			if ep == nil {
				continue
			}
			n := j + 1
			if q := st.Qualifiers[pSeriesOrdinal]; len(q) > 0 {
				if v, err := strconv.Atoi(q[0].Text); err == nil {
					n = v
				}
			}
			p.addEpisode(season.key, n, wikidataKey(ep.ID), wikidata.EntityURL(ep.ID), wikidataKeys(ep), p.wikidataEpisode(ep, n))
			withEpisodes[number] = season.key
		}
	}

	// Seasons without episodes on Wikidata: TVmaze, found by the IMDb id of the series.
	if imdb, _ := e.Text(pIMDb); imdb != "" && (len(seasonIDs) == 0 || len(withEpisodes) < len(seasonIDs)) {
		if err := p.tvmazeEpisodes(ctx, series, imdb, defaultBroadcast, seasonKeys, withEpisodes); err != nil && !IsNotFound(err) {
			return "", err
		}
	}

	names, err := p.personNames(ctx, people.ids, linked)
	if err != nil {
		return "", err
	}
	for _, id := range people.ids {
		person := linked[id]
		if person == nil {
			continue
		}
		pa, err := p.personAttrs(ctx, person, names)
		if err != nil {
			return "", err
		}
		c := people.credits[id]
		p.add(&node{
			key: wikidataKey(id), role: model.NodePerson, typeID: p.d.typeID(typePerson),
			attrs: pa, keys: wikidataKeys(person), selected: c.selected, parent: series.key,
			sourceURL: wikidata.EntityURL(id),
		})
		rel := map[string]any{"roles": c.roles}
		if c.order > 0 {
			rel["order"] = c.order
		}
		p.link(relCredited, series.key, wikidataKey(id), rel)
	}
	return series.key, nil
}

func (p *planner) seriesAttrs(ctx context.Context, e *wikidata.Entity) (map[string]any, error) {
	ids := externalIDs(e)
	if v, _ := e.Text(pIMDb); strings.HasPrefix(v, "tt") {
		ids["imdb"] = v
	}
	if v, _ := e.Text(pTVmazeSeries); v != "" {
		ids["tvmaze"] = v
	}
	label := e.Label("fr", "en", "mul")
	a := map[string]any{
		"title":        label,
		"synopsis":     i18n(e.Descriptions),
		"external_ids": ids,
		"sources":      p.sources("wikidata"),
	}
	if t := originalTitle(e); t != "" && t != label {
		a["original_title"] = t
	} else if en := e.Label("en"); en != "" && en != label {
		a["original_title"] = en
	}
	if y, ok := e.Year(pStart, pPublication); ok {
		a["start_year"] = y
	}
	if y, ok := e.Year(pEnd); ok {
		a["end_year"] = y
	}
	if n, ok := e.Quantity(pSeasonCount, "1"); ok && n > 0 {
		a["season_count"] = n
	}
	if n, ok := e.Quantity(pEpisodeCount, "1"); ok && n > 0 {
		a["episode_count"] = n
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

type seriesCredit struct {
	roles    []any
	order    int
	selected bool
}

type seriesCredits struct {
	ids     []string
	credits map[string]*seriesCredit
}

// seriesPeople: creators (selected), then the first actors (the first ones selected).
func (p *planner) seriesPeople(e *wikidata.Entity) seriesCredits {
	out := seriesCredits{credits: map[string]*seriesCredit{}}
	add := func(id, role string, order int, selected bool) {
		c := out.credits[id]
		if c == nil {
			c = &seriesCredit{}
			out.credits[id] = c
			out.ids = append(out.ids, id)
		}
		if !slicesContainsAny(c.roles, role) {
			c.roles = append(c.roles, role)
		}
		if order > 0 && c.order == 0 {
			c.order = order
		}
		c.selected = c.selected || selected
	}
	for _, id := range compactNonEmpty(e.IDs(pCreator)) {
		add(id, "creator", 0, true)
	}
	for i, id := range compactNonEmpty(e.IDs(pCast)) {
		if i >= maxCast {
			break
		}
		add(id, "actor", i+1, i < selectedCast)
	}
	return out
}

// broadcast adds the broadcast of a series by a broadcaster (« AMC »), with
// broadcast_on from the series; returns its key.
func (p *planner) broadcast(series *node, network *wikidata.Entity, seriesAttrs map[string]any) string {
	if network == nil {
		return ""
	}
	key := "broadcast:" + series.key + ":" + network.ID
	if p.byKey[key] != nil {
		return key
	}
	name := network.Label("fr", "en", "mul")
	attrs := map[string]any{"name": name, "channel": name, "sources": p.sources("wikidata")}
	switch {
	case network.HasID(pInstanceOf, map[string]bool{qStreamingVideo: true}):
		attrs["type"] = "streaming"
	case network.HasID(pInstanceOf, map[string]bool{qCableChannel: true}):
		attrs["type"] = "cable"
	default:
		attrs["type"] = "tv"
	}
	if start, ok := seriesAttrs["start_year"].(int); ok {
		years := strconv.Itoa(start)
		if end, ok := seriesAttrs["end_year"].(int); ok && end != start {
			years += "-" + strconv.Itoa(end)
		}
		attrs["name"] = name + " " + years
	}
	// The broadcast belongs to its series: matched through broadcast_on (same channel), never by name alone.
	p.add(&node{
		key: key, role: model.NodeUniverse, typeID: p.d.typeID(typeBroadcast), via: &viaMatch{relBroadcastOn, "channel"},
		attrs: attrs, selected: true, parent: series.key, sourceURL: wikidata.EntityURL(network.ID),
	})
	var rel map[string]any
	if start, ok := seriesAttrs["start_year"].(int); ok {
		rel = map[string]any{"broadcast_start_year": start}
	}
	p.link(relBroadcastOn, series.key, key, rel)
	return key
}

func seasonName(n int) string { return "Saison " + strconv.Itoa(n) }

// wikidataEpisode: attributes of an episode entity (French title, else English).
func (p *planner) wikidataEpisode(e *wikidata.Entity, number int) map[string]any {
	ids := map[string]any{"wikidata": e.ID}
	if v, _ := e.Text(pIMDb); strings.HasPrefix(v, "tt") {
		ids["imdb"] = v
	}
	title := e.Label("fr", "en", "mul")
	a := map[string]any{
		"number":       number,
		"title":        title,
		"external_ids": ids,
		"sources":      p.sources("wikidata"),
	}
	if en := e.Label("en"); en != "" && en != title {
		a["original_title"] = en
	}
	if m, ok := e.Quantity(pDuration, unitMinute); ok && m > 0 {
		a["duration"] = m
	}
	if d, ok := e.EarliestDate(pPublication); ok {
		a["air_date"] = d
	}
	return a
}

// addEpisode adds an episode under its season (contains_episode, order = number).
func (p *planner) addEpisode(seasonKey string, number int, key, url string, match []matchKey, attrs map[string]any) {
	ep := p.add(&node{
		key: key, role: model.NodeVolume, typeID: p.d.typeID(typeEpisode),
		attrs: dropEmpty(attrs), keys: match, selected: true, parent: seasonKey, sourceURL: url,
	})
	p.link(relContainsEpisode, seasonKey, ep.key, map[string]any{"order": number})
}

// tvmazeEpisodes completes from TVmaze the seasons Wikidata leaves without
// episodes (seasons it does not know are created).
func (p *planner) tvmazeEpisodes(ctx context.Context, series *node, imdb, broadcast string, seasonKeys, withEpisodes map[int]string) error {
	show, err := p.c.TV.LookupIMDb(ctx, imdb)
	if err != nil {
		return err
	}
	ids, _ := series.attrs["external_ids"].(map[string]any)
	if ids != nil && ids["tvmaze"] == nil {
		ids["tvmaze"] = strconv.Itoa(show.ID)
	}
	seasons, err := p.c.TV.Seasons(ctx, show.ID)
	if err != nil {
		return err
	}
	episodes, err := p.c.TV.Episodes(ctx, show.ID)
	if err != nil {
		return err
	}
	source := tvmaze.ShowURL(show.ID)
	for _, s := range seasons {
		if s.Number <= 0 || withEpisodes[s.Number] != "" {
			continue
		}
		key := seasonKeys[s.Number]
		if key == "" {
			sa := map[string]any{
				"name": seasonName(s.Number), "number": s.Number,
				"external_ids": map[string]any{"tvmaze": strconv.Itoa(s.ID)}, "sources": p.sources("tvmaze"),
			}
			if y, _ := strconv.Atoi(firstN(s.PremiereDate, 4)); y > 0 {
				sa["year"] = y
			}
			if s.EpisodeOrder > 0 {
				sa["episode_count"] = s.EpisodeOrder
			}
			season := p.add(&node{
				key: "tvmaze-season:" + strconv.Itoa(s.ID), role: model.NodeSeason, typeID: p.d.typeID(typeSeason),
				attrs: sa, keys: []matchKey{{path: []string{"external_ids", "tvmaze"}, value: strconv.Itoa(s.ID)}},
				selected: true, parent: series.key, sourceURL: source,
			})
			key = season.key
			seasonKeys[s.Number] = key
			if broadcast != "" {
				p.link(relContainsSeason, broadcast, key, nil)
			}
		}
		for _, ep := range episodes {
			if ep.Season != s.Number || ep.Number <= 0 || ep.Type != "regular" {
				continue
			}
			a := map[string]any{
				"number": ep.Number, "title": ep.Name,
				"external_ids": map[string]any{"tvmaze": strconv.Itoa(ep.ID)}, "sources": p.sources("tvmaze"),
			}
			if ep.Runtime > 0 {
				a["duration"] = ep.Runtime
			}
			if len(ep.Airdate) == len("2006-01-02") {
				a["air_date"] = ep.Airdate
			}
			p.addEpisode(key, ep.Number, "tvmaze:"+strconv.Itoa(ep.ID), source,
				[]matchKey{{path: []string{"external_ids", "tvmaze"}, value: strconv.Itoa(ep.ID)}}, a)
		}
	}
	return nil
}

func firstN(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}

// ── Box sets ────────────────────────────────────────────────────────

// searchSeriesEditions searches box sets at the BnF by title or EAN, or those
// of a series (titled like it: box sets have no BnF work record).
func (c *Connector) searchSeriesEditions(ctx context.Context, q model.ExternalEditionQuery) ([]model.ExternalEditionHit, error) {
	if q.ISBN != "" {
		return nil, InputError("this connector searches editions by title, barcode or series")
	}
	if q.Work != "" && !wikidata.IsQID(q.Work) {
		return nil, InputError("work must be a Wikidata identifier (Q…)")
	}
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	var recs []bnf.Record
	var titles []string // a box set of the series is titled like it (200 $a)
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
		titles = compactNonEmpty([]string{fold(e.Label("fr")), fold(e.Label("en"))})
		if recs, _, err = c.BnF.Search(ctx, videoQuery("bib.title all "+bnf.Quote(e.Label("fr", "en"))), maxFilmEditionHits); err != nil {
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
		if len(titles) > 0 && !slicesContains(titles, fold(r.Get("200", "a"))) {
			continue // another work found by the title search (« El Camino »)
		}
		v := parseVideo(r)
		h := model.ExternalEditionHit{
			ID: r.ARK, BnfArk: r.ARK, Title: boxSetTitle(r), Publishers: videoPublisherNames(r),
			Format: v.label(), Barcode: v.ean, Collection: collectionOf(r), SourceURL: bnf.RecordURL(r.ARK),
		}
		if v.year > 0 {
			h.StartYear = &v.year
		}
		hits = append(hits, h)
		arks = append(arks, r.ARK)
	}
	// By title: the seasons in order, then the complete sets.
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Title < hits[j].Title })
	existing, err := c.existing(ctx, map[uuid.UUID][]string{d.typeID(typeVideoRelease): arks}, []string{"external_ids", "bnf_ark"})
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].ExistingItemID = existing[hits[i].ID]
	}
	return hits, nil
}

// boxSetTitle: « Breaking bad — Intégrale de la saison 1 » (200 $a, $h, $i, $e).
func boxSetTitle(r *bnf.Record) string {
	t := r.Get("200", "a")
	var parts []string
	for _, code := range []string{"h", "i", "e"} {
		if v := strings.TrimSpace(r.Get("200", code)); v != "" {
			parts = append(parts, v)
		}
	}
	if len(parts) > 0 {
		t += " — " + strings.Join(parts, ", ")
	}
	return t
}

var (
	reSeasons  = regexp.MustCompile(`(?i)saisons?\s+([0-9][0-9 ,àaet\-]*)`)
	reNumbers  = regexp.MustCompile(`[0-9]+`)
	reRange    = regexp.MustCompile(`([0-9]+)\s*(?:à|-)\s*([0-9]+)`)
	reFinal    = regexp.MustCompile(`(?i)saison finale|derni[èe]re saison`)
	reComplete = regexp.MustCompile(`(?i)int[ée]grale de la s[ée]rie|s[ée]rie compl[èe]te|coffret int[ée]gral`)
)

// boxSetSeasons reads the seasons a box set contains (200 $h $i $e): « Intégrale
// de la saison 1 » → [1], « Intégrale des saisons 1, 2 et 3 » → [1 2 3],
// « Saisons 1 à 4 » → [1 2 3 4]; all = « L'intégrale de la série » ; final = « Saison
// finale » without a number (the last season of the series).
func boxSetSeasons(r *bnf.Record) (seasons []int, all, final bool) {
	text := strings.Join([]string{r.Get("200", "h"), r.Get("200", "i"), r.Get("200", "e")}, " ")
	if reComplete.MatchString(text) {
		return nil, true, false
	}
	// « Saison finale [Saison 5] » : the bracketed number counts.
	for _, m := range reSeasons.FindAllStringSubmatch(text, -1) {
		if rg := reRange.FindStringSubmatch(m[1]); rg != nil {
			from, _ := strconv.Atoi(rg[1])
			to, _ := strconv.Atoi(rg[2])
			for n := from; n <= to && n-from < maxSeriesSeasons; n++ {
				seasons = append(seasons, n)
			}
			continue
		}
		for _, s := range reNumbers.FindAllString(m[1], -1) {
			if n, err := strconv.Atoi(s); err == nil && n > 0 && !containsInt(seasons, n) {
				seasons = append(seasons, n)
			}
		}
	}
	return seasons, false, len(seasons) == 0 && reFinal.MatchString(text)
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// addBoxSet adds a series box set, the series it belongs to (found on
// Wikidata by its title when none is chosen), and includes_episode towards
// the episodes of the seasons it contains.
func (p *planner) addBoxSet(ctx context.Context, ark, workKey string) error {
	rec, err := p.videoRecord(ctx, ark)
	if err != nil {
		return err
	}
	if workKey == "" {
		if workKey, err = p.seriesOfRecord(ctx, rec); err != nil {
			return err
		}
	}
	release := p.addVideoReleaseNode(rec, ark, boxSetTitle(rec), workKey)
	if workKey == "" {
		return nil
	}
	numbers, all, final := boxSetSeasons(rec)
	seasons := p.childrenOf(workKey, model.NodeSeason)
	if final {
		last := 0
		for _, s := range seasons {
			if n, _ := s.attrs["number"].(int); n > last {
				last = n
			}
		}
		numbers = []int{last}
	}
	for _, season := range seasons {
		n, _ := season.attrs["number"].(int)
		if !all && !containsInt(numbers, n) {
			continue
		}
		episodes := p.childrenOf(season.key, model.NodeVolume)
		sort.SliceStable(episodes, func(i, j int) bool {
			a, _ := episodes[i].attrs["number"].(int)
			b, _ := episodes[j].attrs["number"].(int)
			return a < b
		})
		for _, ep := range episodes {
			p.link(relIncludesEpisode, release.key, ep.key, nil)
		}
	}
	return nil
}

// seriesOfRecord finds on Wikidata the series a box set is titled after (200 $a),
// among the series classes, with the same French or English label; "" if none.
func (p *planner) seriesOfRecord(ctx context.Context, rec *bnf.Record) (string, error) {
	t := rec.Get("200", "a")
	if t == "" {
		return "", nil
	}
	ids, err := p.c.WD.Search(ctx, t, seriesClasses, 5)
	if err != nil || len(ids) == 0 {
		return "", err
	}
	ents, err := p.c.WD.Entities(ctx, ids, fetchLangs, false)
	if err != nil {
		return "", err
	}
	for _, id := range ids {
		if e := ents[id]; e != nil && (fold(e.Label("fr")) == fold(t) || fold(e.Label("en")) == fold(t)) {
			if p.byKey[wikidataKey(id)] != nil {
				return wikidataKey(id), nil
			}
			return p.addSeries(ctx, id)
		}
	}
	return "", nil
}
