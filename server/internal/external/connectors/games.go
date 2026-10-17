package connectors

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/wikidata"
	"github.com/shinsaber/colec/internal/model"
)

// Games profile (P3-7, jeux-video.md): a video game on Wikidata, its platforms
// (each linked to its manufacturer) and companies; its French releases at the
// BnF (legal deposit), found by title or EAN. A release found alone brings its
// game (Wikidata, same title) and its platform (215: « 1 cartouche (Nintendo Switch) »).

// Wikidata classes and properties of the games profile (labels checked on
// Wikidata on 2026-10-17, docs/sources-externes.md).
const (
	qVideoGame      = "Q7889"
	qHomeConsole    = "Q8076"
	qHomeConsoleMod = "Q115187350"
	qHandheld       = "Q941818"
	qPersonalComp   = "Q16338"

	pPlatform     = "P400"
	pDeveloper    = "P178"
	pGamePublish  = "P123"
	pGameMode     = "P404"
	pPEGI         = "P908"
	pManufacturer = "P176"
	pDiscontinued = "P2669"
	pCountry      = "P17"
	pWebsite      = "P856"

	maxGameEditionHits = 150
	maxMakers          = 3
)

var (
	gameClasses    = []string{qVideoGame}
	gameClassesSet = map[string]bool{qVideoGame: true}
	gameGenres     = map[string]string{
		"Q270948": "action", "Q23916": "adventure", "Q343568": "action_adventure", "Q744038": "rpg", "Q5923834": "rpg",
		"Q105906396": "rpg", "Q1422746": "action_rpg", "Q1529437": "tactical_rpg", "Q828322": "platform", "Q54767": "puzzle",
		"Q472055": "strategy", "Q2176159": "strategy", "Q1610017": "simulation", "Q1199309": "life_simulation",
		"Q860750": "racing", "Q868217": "sports", "Q846224": "fighting", "Q401831": "beat_em_up", "Q1163960": "hack_and_slash",
		"Q4282636": "shooter", "Q185029": "fps", "Q1044478": "shoot_em_up", "Q11338014": "horror", "Q333967": "survival_horror",
		"Q19643088": "metroidvania", "Q1143132": "roguelike", "Q105098453": "roguelike", "Q125693245": "roguelike",
		"Q11590806": "point_and_click", "Q689445": "visual_novel", "Q2632782": "rhythm", "Q7888616": "party",
		"Q1050023": "casual", "Q15613992": "arcade", "Q27670585": "science_fiction",
	}
	gameModes = map[string]string{
		"Q208850": "single_player", "Q6895044": "multiplayer", "Q5252671": "multiplayer", "Q1630770": "multiplayer",
		"Q1758804": "cooperative", "Q107494132": "cooperative", "Q61005756": "split_screen", "Q719798": "pvp",
	}
	pegiRatings = map[string]string{
		"Q14915512": "3", "Q14915514": "7", "Q14915515": "12", "Q14915516": "16", "Q14915517": "18",
	}
	platformKinds = map[string]string{
		qHomeConsole: "home_console", qHomeConsoleMod: "home_console", qHandheld: "handheld", qPersonalComp: "computer",
	}
	// gameRoles: UNIMARC relator codes (712 $4) of the companies of a game record.
	gameRoles = map[string]string{"650": "publisher", "310": "distributor", "635": "developer"}
)

// ── Game ────────────────────────────────────────────────────────────

// addGame adds a Wikidata game, its platforms (with their manufacturer) and its companies.
func (p *planner) addGame(ctx context.Context, qid string) (string, error) {
	ents, err := p.c.WD.Entities(ctx, []string{qid}, fetchLangs, true)
	if err != nil {
		return "", err
	}
	e := ents[qid]
	if e == nil {
		return "", notFound("wikidata entity %s", qid)
	}
	if !e.HasID(pInstanceOf, gameClassesSet) {
		return "", notFound("wikidata entity %s is not a video game", qid)
	}
	game := p.add(&node{
		key: wikidataKey(qid), role: model.NodeWork, typeID: p.d.typeID(typeGame),
		attrs: p.gameAttrs(e), keys: wikidataKeys(e), selected: true, sourceURL: wikidata.EntityURL(qid),
	})

	platforms := compactNonEmpty(e.IDs(pPlatform))
	roles := map[string][]any{}
	var companies []string
	addRole := func(id, role string) {
		if _, ok := roles[id]; !ok {
			companies = append(companies, id)
		}
		if !slicesContainsAny(roles[id], role) {
			roles[id] = append(roles[id], role)
		}
	}
	for _, id := range compactNonEmpty(e.IDs(pDeveloper)) {
		addRole(id, "developer")
	}
	for _, id := range compactNonEmpty(e.IDs(pGamePublish)) {
		addRole(id, "publisher")
	}
	linked, err := p.c.WD.Entities(ctx, compactNonEmpty(append(slicesClone(platforms), companies...)), fetchLangs, true)
	if err != nil {
		return "", err
	}
	for _, id := range companies {
		key, err := p.company(ctx, linked[id], game.key)
		if err != nil {
			return "", err
		}
		if key != "" {
			p.link(relProducedBy, game.key, key, map[string]any{"roles": roles[id]})
		}
	}
	for _, id := range platforms {
		key, err := p.platform(ctx, linked[id], game.key)
		if err != nil {
			return "", err
		}
		if key != "" {
			p.link(relOnPlatform, game.key, key, nil)
		}
	}
	return game.key, nil
}

func (p *planner) gameAttrs(e *wikidata.Entity) map[string]any {
	a := map[string]any{
		"title":        title(e),
		"description":  i18n(e.Descriptions),
		"external_ids": externalIDs(e),
		"sources":      p.sources("wikidata"),
	}
	if t := originalTitle(e); t != "" {
		a["original_title"] = t
	}
	if y, ok := e.EarliestYear(pPublication); ok {
		a["release_year"] = y
	}
	collect := func(prop string, m map[string]string) []any {
		var out []any
		for _, id := range e.IDs(prop) {
			if v := m[id]; v != "" && !slicesContainsAny(out, v) {
				out = append(out, v)
			}
		}
		return out
	}
	if g := collect(pGenre, gameGenres); len(g) > 0 {
		a["genres"] = g
	}
	if m := collect(pGameMode, gameModes); len(m) > 0 {
		a["game_modes"] = m
	}
	for _, id := range e.IDs(pPEGI) {
		if r := pegiRatings[id]; r != "" {
			a["pegi"] = r
			break
		}
	}
	return dropEmpty(a)
}

// company adds a Wikidata company (type Publisher of the « people » package).
func (p *planner) company(ctx context.Context, e *wikidata.Entity, parent string) (string, error) {
	if e == nil {
		return "", nil
	}
	key := wikidataKey(e.ID)
	if p.byKey[key] != nil {
		return key, nil
	}
	name := e.Label("fr", "en", "mul")
	a := map[string]any{
		"name":         name,
		"description":  i18n(e.Descriptions),
		"external_ids": map[string]any{"wikidata": e.ID},
		"sources":      p.sources("wikidata"),
	}
	if w, _ := e.Text(pWebsite); strings.HasPrefix(w, "http") {
		a["website"] = w
	}
	if c := firstID(e, pCountry); c != "" {
		codes, err := p.c.WD.StringClaims(ctx, c, pISOAlpha2)
		if err != nil {
			return "", err
		}
		if len(codes) > 0 && len(codes[0]) == 2 {
			a["country"] = strings.ToUpper(codes[0])
		}
	}
	p.add(&node{
		key: key, role: model.NodePublisher, typeID: p.d.typeID(typePublisher), attrs: dropEmpty(a),
		keys: append(wikidataKeys(e), matchKey{path: []string{"name"}, value: name, fold: true}), selected: true,
		parent: parent, sourceURL: wikidata.EntityURL(e.ID),
	})
	return key, nil
}

// platform adds a Wikidata platform and its manufacturer (produced_by {manufacturer}).
func (p *planner) platform(ctx context.Context, e *wikidata.Entity, parent string) (string, error) {
	if e == nil {
		return "", nil
	}
	key := wikidataKey(e.ID)
	if p.byKey[key] != nil {
		return key, nil
	}
	name := e.Label("fr", "en", "mul")
	a := map[string]any{"name": name, "external_ids": map[string]any{"wikidata": e.ID}, "sources": p.sources("wikidata")}
	for _, id := range e.IDs(pInstanceOf) {
		if k := platformKinds[id]; k != "" {
			a["kind"] = k
			break
		}
	}
	if y, ok := e.EarliestYear(pInception); ok {
		a["release_year"] = y
	} else if y, ok := e.EarliestYear(pPublication); ok {
		a["release_year"] = y
	}
	if y, ok := e.Year(pDiscontinued); ok {
		a["discontinued_year"] = y
	}
	p.add(&node{
		key: key, role: model.NodeUniverse, typeID: p.d.typeID(typePlatform), attrs: a,
		keys: append(wikidataKeys(e), matchKey{path: []string{"name"}, value: name, fold: true}), selected: true,
		parent: parent, sourceURL: wikidata.EntityURL(e.ID),
	})
	// Every manufacturer (Switch: Nintendo and its hardware division).
	makers := compactNonEmpty(e.IDs(pManufacturer))
	if len(makers) > maxMakers {
		makers = makers[:maxMakers]
	}
	if len(makers) > 0 {
		ents, err := p.c.WD.Entities(ctx, makers, fetchLangs, true)
		if err != nil {
			return "", err
		}
		for _, id := range makers {
			mk, err := p.company(ctx, ents[id], key)
			if err != nil {
				return "", err
			}
			if mk != "" {
				p.link(relProducedBy, key, mk, map[string]any{"roles": []any{"manufacturer"}})
			}
		}
	}
	return key, nil
}

// ── Releases ────────────────────────────────────────────────────────

// searchGameEditions searches French releases at the BnF by EAN, by title, or
// those titled like a Wikidata game; game records only.
func (c *Connector) searchGameEditions(ctx context.Context, q model.ExternalEditionQuery) ([]model.ExternalEditionHit, error) {
	if q.ISBN != "" {
		return nil, InputError("this connector searches editions by title, barcode or game")
	}
	if q.Work != "" && !wikidata.IsQID(q.Work) {
		return nil, InputError("work must be a Wikidata identifier (Q…)")
	}
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	var recs []bnf.Record
	var titles []string
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
		titles = compactNonEmpty([]string{fold(e.Label("fr", "mul")), fold(e.Label("en", "mul"))})
		if recs, _, err = c.BnF.Search(ctx, "bib.title all "+bnf.Quote(e.Label("fr", "en", "mul")), maxGameEditionHits); err != nil {
			return nil, err
		}
	case q.Barcode != "":
		if recs, _, err = c.BnF.Search(ctx, eanQuery(q.Barcode), maxEditionHits); err != nil {
			return nil, err
		}
	default:
		if recs, _, err = c.BnF.Search(ctx, "bib.title all "+bnf.Quote(q.Title), maxGameEditionHits); err != nil {
			return nil, err
		}
	}
	hits := []model.ExternalEditionHit{}
	var arks []string
	for i := range recs {
		r := &recs[i]
		if r.ARK == "" || !isGame(r) || slicesContains(arks, r.ARK) {
			continue
		}
		if len(titles) > 0 && !titledLike(fold(gameTitle(r)), titles) {
			continue // another work found by the title search
		}
		g := parseGame(r)
		h := model.ExternalEditionHit{
			ID: r.ARK, BnfArk: r.ARK, Title: gameTitle(r), Publishers: gameCompanyNames(r),
			Format: g.label(), Barcode: g.ean, SourceURL: bnf.RecordURL(r.ARK),
		}
		if g.year > 0 {
			h.StartYear = &g.year
		}
		hits = append(hits, h)
		arks = append(arks, r.ARK)
	}
	// Most recent first.
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
	existing, err := c.existing(ctx, map[uuid.UUID][]string{d.typeID(typeGameRelease): arks}, []string{"external_ids", "bnf_ark"})
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].ExistingItemID = existing[hits[i].ID]
	}
	return hits, nil
}

// gameTitle: 200 $a, its part ($i : « The legend of Zelda : Breath of the wild ») and
// other title information ($e : « Mario kart 8 : deluxe »).
func gameTitle(r *bnf.Record) string {
	t := r.Get("200", "a")
	for _, code := range []string{"i", "e"} {
		if v := strings.TrimSpace(r.Get("200", code)); v != "" {
			t += " : " + v
		}
	}
	return t
}

// eanQuery: the BnF sometimes stores a UPC without its leading zero (« 45496333133 »).
func eanQuery(code string) string {
	q := "bib.ean all " + bnf.Quote(code)
	if t := strings.TrimLeft(code, "0"); t != code && t != "" {
		q += " or bib.ean all " + bnf.Quote(t)
	}
	return q
}

// titledLike: the record title contains one of the game titles (« Mario kart 8 :
// deluxe » for « Mario Kart 8 Deluxe », not « Mario kart 8 » on Wii U).
func titledLike(t string, titles []string) bool {
	for _, x := range titles {
		if x != "" && strings.Contains(t, x) {
			return true
		}
	}
	return false
}

var reGameMedium = regexp.MustCompile(`(?i)cartouche|disque optique|ressource dématérialisée|carte de jeu|cd-rom|dvd-rom|disquette`)

// isGame: a game record — electronic resource (leader 6 « l ») described as a
// game (230 « Jeu », 608 « jeu vidéo »), or a brief record without type whose
// extent is a game medium (« 1 cartouche informatique »).
func isGame(r *bnf.Record) bool {
	if len(r.Leader) > 6 && r.Leader[6] == 'l' {
		if strings.HasPrefix(strings.ToLower(r.Get("230", "a")), "jeu") {
			return true
		}
		for _, g := range r.All("608", "a") {
			if strings.Contains(strings.ToLower(g), "jeu vidéo") {
				return true
			}
		}
		return false
	}
	return (len(r.Leader) < 7 || r.Leader[6] == ' ') && reGameMedium.MatchString(r.Get("215", "a"))
}

// gameRecord is what a BnF game record says of its medium and platform.
type gameRecord struct {
	format, name, platform, ean string
	year                        int
	languages                   []any
}

var (
	reParenPlatform = regexp.MustCompile(`\(([^()]+)\)`)
	reMediumPrefix  = regexp.MustCompile(`(?i)^\s*\d*\s*(cartouches?|disques? optiques? numériques?|disques?|cartes? de jeu)\s+`)
)

func parseGame(r *bnf.Record) gameRecord {
	extent := r.Get("215", "a")
	low := strings.ToLower(extent)
	g := gameRecord{ean: gameEAN(r.Get("073", "a"))}
	switch {
	case strings.Contains(low, "cartouche"):
		g.format, g.name = "cartridge", "Cartouche"
	case strings.Contains(low, "disque optique"), strings.Contains(low, "cd-rom"), strings.Contains(low, "dvd-rom"):
		g.format, g.name = "optical_disc", "Disque"
	case strings.Contains(low, "carte de jeu"):
		g.format, g.name = "card", "Carte de jeu"
	case strings.Contains(low, "dématérialisée"):
		g.format, g.name = "download_code", "Code de téléchargement"
	case extent != "":
		g.format = "other"
	}
	// Platform: « 1 cartouche (Nintendo Switch) », « 1 cartouche Super Nintendo », else 500 $q.
	if m := reParenPlatform.FindStringSubmatch(extent); m != nil && !strings.Contains(strings.ToLower(m[1]), "min") {
		g.platform = strings.TrimSpace(m[1])
	} else if rest := reMediumPrefix.ReplaceAllString(extent, ""); rest != extent {
		if rest = strings.TrimSpace(rest); rest != "" && !strings.HasPrefix(strings.ToLower(rest), "informatique") {
			g.platform = rest
		}
	}
	if g.platform == "" {
		g.platform = strings.TrimSpace(r.Get("500", "q"))
	}
	if start, _ := bnf.Years(r.Get("100", "a")); start > 0 {
		g.year = start
	}
	for _, f := range r.Tagged("101") {
		g.languages = appendLanguages(g.languages, f.All("a"))
	}
	return g
}

// gameEAN normalises an EAN, restoring the leading zero the BnF sometimes drops.
func gameEAN(s string) string {
	if e := external.NormalizeBarcode(s); e != "" {
		return e
	}
	return external.NormalizeBarcode("0" + strings.TrimSpace(s))
}

// label: « Cartouche · Nintendo Switch ».
func (g gameRecord) label() string {
	return strings.Join(compactNonEmpty([]string{g.name, g.platform}), " · ")
}

// gameCompanies reads 712 (650 publisher, 310 distributor, 635 developer), else 210 / 214 $c.
func gameCompanies(r *bnf.Record) []videoPublisher {
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
		if role := gameRoles[f.Get("4")]; role != "" {
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

func gameCompanyNames(r *bnf.Record) []string {
	out := []string{}
	for _, c := range gameCompanies(r) {
		out = append(out, c.name)
	}
	return out
}

// addGameRelease adds a French release, its companies and platform and,
// without a chosen game, the game it records (Wikidata, same title).
func (p *planner) addGameRelease(ctx context.Context, ark, workKey string) error {
	number, _ := bnf.RecordNumber(ark)
	rec, err := p.fetchRecord(ctx, number)
	if err != nil {
		return err
	}
	if !isGame(rec) {
		return notFound("bnf record %s is not a video game", number)
	}
	if workKey == "" {
		if workKey, err = p.gameOfRecord(ctx, rec); err != nil {
			return err
		}
	}
	g := parseGame(rec)
	attrs := map[string]any{
		"title":        gameTitle(rec),
		"external_ids": map[string]any{"bnf_ark": ark},
		"sources":      p.sources("bnf"),
	}
	set := func(k string, v any, ok bool) {
		if ok {
			attrs[k] = v
		}
	}
	set("format", g.format, g.format != "")
	set("ean", g.ean, g.ean != "")
	set("release_year", g.year, g.year > 0)
	set("languages", g.languages, len(g.languages) > 0)
	edition := strings.TrimSpace(rec.Get("205", "a"))
	set("edition", edition, edition != "")
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

	release := p.add(&node{
		key: bnfKey(ark), role: model.NodeEdition, typeID: p.d.typeID(typeGameRelease),
		attrs: attrs, keys: bnfKeys(ark), selected: true, parent: workKey, sourceURL: bnf.RecordURL(ark),
	})
	if workKey != "" {
		p.link(relGameReleasedAs, workKey, release.key, nil)
	}
	if key := p.releasePlatform(g.platform, workKey, release.key); key != "" {
		p.link(relOnPlatform, release.key, key, nil)
	}
	for _, c := range gameCompanies(rec) {
		// The same company already proposed (« Nintendo co. » of the BnF, « Nintendo » of Wikidata).
		if same := p.sameCompany(c.name); same != "" {
			p.link(relProducedBy, release.key, same, map[string]any{"roles": c.roles})
			continue
		}
		pa := map[string]any{"name": c.name, "sources": p.sources("bnf")}
		keys := []matchKey{{path: []string{"name"}, value: c.name, fold: true}}
		if c.ark != "" {
			pa["external_ids"] = map[string]any{"bnf_ark": c.ark}
			keys = append(bnfKeys(c.ark), keys...)
		}
		n := p.add(&node{
			key: publisherKey(c.name), role: model.NodePublisher, typeID: p.d.typeID(typePublisher),
			attrs: pa, keys: keys, selected: true, parent: release.key, sourceURL: release.sourceURL,
		})
		p.link(relProducedBy, release.key, n.key, map[string]any{"roles": c.roles})
	}
	return nil
}

// companySuffixes are dropped when comparing company names.
var companySuffixes = map[string]bool{"co": true, "ltd": true, "inc": true, "sa": true, "sas": true, "sarl": true, "gmbh": true, "corp": true, "corporation": true, "company": true, "kk": true, "llc": true}

// companyName folds a company name without its legal suffix (« Nintendo co. Ltd » → « nintendo »).
func companyName(s string) string {
	words := strings.Fields(fold(strings.NewReplacer(".", " ", ",", " ").Replace(s)))
	for len(words) > 1 && companySuffixes[words[len(words)-1]] {
		words = words[:len(words)-1]
	}
	return strings.Join(words, " ")
}

// sameCompany returns the key of a company of the plan with the same name, or "".
func (p *planner) sameCompany(name string) string {
	want := companyName(name)
	for _, n := range p.nodes {
		if n.typeID == p.d.typeID(typePublisher) && companyName(stringAt(n.attrs, []string{"name"})) == want {
			return n.key
		}
	}
	return ""
}

// releasePlatform returns the platform of a release: a platform of the game
// with the same name (« Switch » matches « Nintendo Switch »), the only
// platform of the game for a brief record without one, else a platform named
// after the record.
func (p *planner) releasePlatform(name, gameKey, releaseKey string) string {
	var own []*node
	for _, n := range p.nodes {
		if n.typeID == p.d.typeID(typePlatform) && n.parent == gameKey && gameKey != "" {
			own = append(own, n)
		}
	}
	if name == "" {
		if len(own) == 1 {
			return own[0].key
		}
		return ""
	}
	want := fold(name)
	for _, n := range own {
		if pn := fold(stringAt(n.attrs, []string{"name"})); pn == want || strings.HasSuffix(pn, " "+want) || strings.HasSuffix(want, " "+pn) {
			return n.key
		}
	}
	key := "platform:" + want
	if p.byKey[key] == nil {
		p.add(&node{
			key: key, role: model.NodeUniverse, typeID: p.d.typeID(typePlatform),
			attrs: map[string]any{"name": name, "sources": p.sources("bnf")},
			keys:  []matchKey{{path: []string{"name"}, value: name, fold: true}}, selected: true, parent: releaseKey,
		})
	}
	return key
}

// gameOfRecord finds on Wikidata the game a release is titled after (200 $a),
// with the same French, English or multilingual label; "" if none.
func (p *planner) gameOfRecord(ctx context.Context, rec *bnf.Record) (string, error) {
	t := gameTitle(rec)
	if t == "" {
		return "", nil
	}
	ids, err := p.c.WD.Search(ctx, rec.Get("200", "a"), gameClasses, 5)
	if err != nil || len(ids) == 0 {
		return "", err
	}
	ents, err := p.c.WD.Entities(ctx, ids, fetchLangs, false)
	if err != nil {
		return "", err
	}
	want := []string{fold(t), fold(rec.Get("200", "a"))}
	for _, id := range ids {
		e := ents[id]
		if e == nil {
			continue
		}
		for _, l := range []string{e.Label("fr"), e.Label("en"), e.Label("mul")} {
			if l != "" && slicesContains(want, fold(l)) {
				if p.byKey[wikidataKey(id)] != nil {
					return wikidataKey(id), nil
				}
				return p.addGame(ctx, id)
			}
		}
	}
	return "", nil
}
