package bdmanga

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/wikidata"
	"github.com/shinsaber/colec/internal/model"
	"golang.org/x/text/unicode/norm"
)

// Novels profile (P3-3): a novel on Wikidata, its cycle and authors; its
// editions at the BnF, listed from the BnF work record (bib.work2bib) when
// Wikidata knows it (P268). An edition is the book itself (single volume), or
// a set of volumes when its volumes link (461) to a set titled like them.

// Wikidata classes and properties of the novels profile (labels checked on
// Wikidata on 2026-10-17).
const (
	qLiteraryWork = "Q7725634"
	qNovel        = "Q8261"

	pForm      = "P7937" // forme de l'œuvre
	pGenre     = "P136"
	pLanguage  = "P407" // langue de l'œuvre
	pSeries    = "P179" // fait partie de la série
	pSeriesOrd = "P1545"
	pISO6391   = "P218"
)

var (
	novelClasses    = []string{qLiteraryWork, qNovel}
	novelClassesSet = map[string]bool{qLiteraryWork: true, qNovel: true}
	// sagaClasses: a P179 series becomes a Saga only when it is a series of works.
	sagaClasses = map[string]bool{
		"Q1667921":  true, // suite romanesque
		"Q277759":   true, // série de livres
		"Q867335":   true, // cycle littéraire
		"Q13593966": true, // trilogie littéraire
		"Q17710980": true, // tétralogie littéraire
		"Q614101":   true, // heptalogie
		"Q7725310":  true, // série d'œuvres
	}
	novelForms = map[string]string{
		qNovel: "novel", "Q149537": "novella", "Q1279564": "short_stories",
		"Q35760": "essay", "Q25379": "play", "Q482": "poetry",
	}
	novelGenres = map[string]string{
		"Q132311": "fantasy", "Q24925": "science_fiction", "Q208505": "crime", "Q186424": "crime",
		"Q182015": "thriller", "Q858330": "romance", "Q1196408": "historical", "Q193606": "horror",
		"Q1233720": "young_adult", "Q111984153": "young_adult", "Q131539": "children",
	}
	// contributorRoles: UNIMARC relator codes (7XX $4) of the people of an edition.
	contributorRoles = map[string]string{"730": "translator", "440": "illustrator", "080": "preface"}
)

// maxNovelEditionHits bounds the editions listed for a novel (Les Misérables:
// 312 records at the BnF); French editions come first.
const maxNovelEditionHits = 150

// addNovelWork adds a novel, its cycle (Saga), its franchise and its authors.
func (p *planner) addNovelWork(ctx context.Context, qid string) (string, error) {
	ents, err := p.c.WD.Entities(ctx, []string{qid}, fetchLangs, true)
	if err != nil {
		return "", err
	}
	e := ents[qid]
	if e == nil {
		return "", notFound("wikidata entity %s", qid)
	}
	if !e.HasID(pInstanceOf, novelClassesSet) {
		return "", notFound("wikidata entity %s is not a literary work", qid)
	}
	attrs, err := p.novelAttrs(ctx, e)
	if err != nil {
		return "", err
	}
	work := p.add(&node{
		key: wikidataKey(qid), role: model.NodeWork, typeID: p.d.typeID(typeNovel),
		attrs: attrs, keys: wikidataKeys(e), selected: true, sourceURL: wikidata.EntityURL(qid),
	})

	people := compactNonEmpty(e.IDs(pAuthor))
	series := ""
	order := ""
	if sts := e.Claims[pSeries]; len(sts) > 0 {
		series = sts[0].Value.ID
		if q := sts[0].Qualifiers[pSeriesOrd]; len(q) > 0 {
			order = q[0].Text
		}
	}
	universes := e.IDs(pFranchise)
	if len(universes) > 1 {
		universes = universes[:1]
	}
	linked, err := p.c.WD.Entities(ctx, compactNonEmpty(append(append(slicesClone(universes), series), people...)), fetchLangs, true)
	if err != nil {
		return "", err
	}
	if s := linked[series]; s != nil && s.HasID(pInstanceOf, sagaClasses) {
		p.add(&node{
			key: wikidataKey(series), role: model.NodeUniverse, typeID: p.d.typeID(typeSaga),
			attrs: map[string]any{
				"title": title(s), "description": i18n(s.Descriptions),
				"external_ids": externalIDs(s), "sources": p.sources("wikidata"),
			},
			keys: wikidataKeys(s), selected: true, parent: work.key, sourceURL: wikidata.EntityURL(series),
		})
		var attrs map[string]any
		if n, ok := bnf.FirstNumber(order); ok {
			attrs = map[string]any{"order": n}
		}
		p.link(relContainsWork, wikidataKey(series), work.key, attrs)
	}
	for _, id := range universes {
		if u := linked[id]; u != nil {
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
		p.link(relCreatedBy, work.key, wikidataKey(id), map[string]any{"roles": []any{"author"}})
	}
	return work.key, nil
}

func (p *planner) novelAttrs(ctx context.Context, e *wikidata.Entity) (map[string]any, error) {
	a := map[string]any{
		"title":        title(e),
		"description":  i18n(e.Descriptions),
		"external_ids": externalIDs(e),
		"sources":      p.sources("wikidata"),
	}
	if t := originalTitle(e); t != "" {
		a["original_title"] = t
	}
	if y, ok := e.Year(pPublication, pInception); ok {
		a["first_publication_year"] = y
	}
	for _, id := range e.IDs(pForm) {
		if f := novelForms[id]; f != "" {
			a["form"] = f
			break
		}
	}
	if _, ok := a["form"]; !ok && e.HasID(pInstanceOf, map[string]bool{qNovel: true}) {
		a["form"] = "novel"
	}
	var genres []any
	seen := map[string]bool{}
	for _, id := range e.IDs(pGenre) {
		if g := novelGenres[id]; g != "" && !seen[g] {
			seen[g] = true
			genres = append(genres, g)
		}
	}
	if len(genres) > 0 {
		a["genres"] = genres
	}
	if lang := firstID(e, pLanguage); lang != "" {
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

// ── Editions ────────────────────────────────────────────────────────

// searchNovelEditions lists editions by work (BnF work record of the novel),
// title or ISBN. A record linked to a set titled like it stands for that set
// (multi-volume edition); any other record is a single-volume edition.
func (c *Connector) searchNovelEditions(ctx context.Context, q model.ExternalEditionQuery) ([]model.ExternalEditionHit, error) {
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	var recs []bnf.Record
	// anchor: the novel searched for ; a set stands for its volumes only when its title contains it.
	anchor := q.Title
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
		anchor = e.Label("fr", "en")
		number := ""
		if ark, _ := externalIDs(e)["bnf_ark"].(string); ark != "" {
			number, _ = bnf.RecordNumber(ark)
		}
		if number != "" {
			recs, _, err = c.BnF.Search(ctx, "bib.work2bib all "+bnf.Quote(number), maxNovelEditionHits)
		} else {
			// Work unknown to the BnF: search its French (or English) title.
			recs, _, err = c.BnF.Search(ctx, "bib.title all "+bnf.Quote(e.Label("fr", "en")), maxEditionHits)
		}
		if err != nil {
			return nil, err
		}
	case q.ISBN != "":
		if recs, _, err = c.BnF.Search(ctx, "bib.fuzzyIsbn all "+bnf.Quote(q.ISBN), maxISBNVolumeHits); err != nil {
			return nil, err
		}
	default:
		if recs, _, err = c.BnF.Search(ctx, "bib.title all "+bnf.Quote(q.Title), maxEditionHits); err != nil {
			return nil, err
		}
	}

	// Sets the records link to (461), fetched once to compare their titles.
	var setNumbers []string
	for _, r := range recs {
		for _, f := range r.Tagged("461") {
			if n := f.Get("0"); bnf.ARK(n) != "" && !slicesContains(setNumbers, n) {
				setNumbers = append(setNumbers, n)
			}
		}
	}
	sets := map[string]*bnf.Record{}
	if len(setNumbers) > 0 {
		found, _, err := c.BnF.Search(ctx, "bib.recordid any "+bnf.Quote(joinSpace(setNumbers)), len(setNumbers))
		if err != nil {
			return nil, err
		}
		for i := range found {
			if n, ok := bnf.RecordNumber(found[i].ARK); ok {
				sets[n] = &found[i]
			}
		}
	}

	hits := []model.ExternalEditionHit{}
	seen := map[string]bool{}
	for i := range recs {
		r := &recs[i]
		if r.ARK == "" || !r.IsText() || novelTitle(r) == "" {
			continue // notice quasi vide
		}
		if set := novelSet(r, sets, anchor); set != nil {
			if seen[set.ARK] {
				continue
			}
			seen[set.ARK] = true
			h := editionHit(set)
			h.Collection = collectionOf(r)
			if q.ISBN != "" {
				h.MatchedVolume = &model.ExternalVolumeHit{BnfArk: r.ARK, Title: r.Get("200", "a"), ISBN: q.ISBN}
				if n, ok := volumeNumber(*r, setNumberOf(set)); ok {
					h.MatchedVolume.Number = &n
				}
			}
			hits = append(hits, h)
			continue
		}
		if seen[r.ARK] {
			continue
		}
		seen[r.ARK] = true
		h := editionHit(r)
		h.Collection = collectionOf(r)
		h.ISBN = recordISBN(r)
		hits = append(hits, h)
	}
	// French editions first (legal deposit: the catalogue mixes translations), then the most recent.
	sort.SliceStable(hits, func(i, j int) bool {
		fi, fj := hits[i].Language == "fr", hits[j].Language == "fr"
		if fi != fj {
			return fi
		}
		yi, yj := 0, 0
		if hits[i].StartYear != nil {
			yi = *hits[i].StartYear
		}
		if hits[j].StartYear != nil {
			yj = *hits[j].StartYear
		}
		return yi > yj
	})

	arks := make([]string, len(hits))
	for i := range hits {
		arks[i] = hits[i].BnfArk
	}
	existing, err := c.existing(ctx, map[uuid.UUID][]string{d.typeID(typeEdition): arks}, []string{"external_ids", "bnf_ark"})
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].ExistingItemID = existing[hits[i].BnfArk]
	}
	return hits, nil
}

// novelTitle: the title of the record (200), else that of the first work it
// contains (423$t, compilation without collective title), else its uniform
// title (500$a) ; empty for the near-empty records of the catalogue.
func novelTitle(r *bnf.Record) string {
	if t := editionTitle(r); t != "" {
		return t
	}
	if t := r.Get("423", "t"); t != "" {
		return t
	}
	return r.Get("500", "a")
}

func editionHit(r *bnf.Record) model.ExternalEditionHit {
	h := model.ExternalEditionHit{
		BnfArk: r.ARK, Title: novelTitle(r), Publishers: publishers(r), SourceURL: bnf.RecordURL(r.ARK),
		Language: novelLanguage(r),
	}
	if h.Publishers == nil {
		h.Publishers = []string{}
	}
	if start, _ := bnf.Years(r.Get("100", "a")); start > 0 {
		h.StartYear = &start
	}
	return h
}

// novelSet returns the set a record belongs to when the set is titled like the novel
// searched for (anchor : « Les Misérables » for the Pocket set) ; a volume of
// collected works (« Œuvres complètes ») stays a single-volume edition. Without
// anchor (ISBN search), the set must be titled like the record itself.
func novelSet(r *bnf.Record, sets map[string]*bnf.Record, anchor string) *bnf.Record {
	own := fold(r.Get("200", "a"))
	if a := fold(anchor); a != "" {
		own = a
	}
	for _, f := range r.Tagged("461") {
		set := sets[f.Get("0")]
		if set == nil || own == "" {
			continue
		}
		if t := fold(set.Get("200", "a")); t != "" && (strings.Contains(own, t) || strings.Contains(t, own)) {
			return set
		}
	}
	return nil
}

func setNumberOf(set *bnf.Record) string {
	n, _ := bnf.RecordNumber(set.ARK)
	return n
}

// collectionOf: the editorial collection and the number in it (225$a $v): « Folio 2 ».
func collectionOf(r *bnf.Record) string {
	return strings.TrimSpace(r.Get("225", "a") + " " + r.Get("225", "v"))
}

func recordISBN(r *bnf.Record) string {
	if isbn := bnf.NormalizeISBN(r.Get("073", "a")); isbn != "" {
		return isbn
	}
	return bnf.NormalizeISBN(r.Get("010", "a"))
}

// fold lowercases and strips the accents and punctuation of a title.
func fold(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '\'' || r == '-':
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func slicesContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// addNovelEdition adds an edition: the set and its volumes when the record has
// volumes (bib.ens2bib), otherwise the book itself; plus its publishers and the
// translators, illustrators and preface writers of the record.
func (p *planner) addNovelEdition(ctx context.Context, ark, workKey string) error {
	number, _ := bnf.RecordNumber(ark)
	rec, err := p.fetchRecord(ctx, number)
	if err != nil {
		return err
	}
	vols, _, err := p.c.BnF.Search(ctx, "bib.ens2bib all "+bnf.Quote(number), maxVolumes)
	if err != nil {
		return err
	}
	lang := novelLanguage(rec)
	if lang == "" {
		lang = "fr"
	}
	attrs := map[string]any{
		"title":        map[string]any{lang: novelTitle(rec)},
		"language":     lang, // hors de la liste du schéma (traduction en breton…) : écarté par fit
		"external_ids": map[string]any{"bnf_ark": ark},
		"sources":      p.sources("bnf"),
	}
	if start, end := bnf.Years(rec.Get("100", "a")); start > 0 {
		attrs["start_year"] = start
		if end > 0 {
			attrs["end_year"] = end
		}
	}
	if c := rec.Get("225", "a"); c != "" {
		attrs["collection"] = c
		if n := rec.Get("225", "v"); n != "" {
			attrs["collection_number"] = n
		}
	}
	keys := bnfKeys(ark)
	if len(vols) == 0 {
		// A single volume: the edition is the book.
		if isbn := recordISBN(rec); isbn != "" {
			attrs["isbn"] = isbn
			keys = append(keys, matchKey{path: []string{"isbn"}, value: isbn})
		}
		if n, ok := bnf.PageCount(rec.Get("215", "a")); ok {
			attrs["page_count"] = n
		}
		if price, ok := bnf.PriceEUR(rec.Get("010", "d")); ok {
			attrs["retail_price"] = price
		}
		if f := bindingFormat(rec.Get("010", "b")); f != "" {
			attrs["format"] = f
		}
	}
	edition := p.add(&node{
		key: bnfKey(ark), role: model.NodeEdition, typeID: p.d.typeID(typeEdition),
		attrs: attrs, keys: keys, selected: true, parent: workKey, sourceURL: bnf.RecordURL(ark),
	})
	if workKey != "" {
		p.link(relPublishedAs, workKey, edition.key, nil)
	}
	for _, name := range publishers(rec) {
		pub := p.add(&node{
			key: publisherKey(name), role: model.NodePublisher, typeID: p.d.typeID(typePublisher),
			attrs: map[string]any{"name": name, "sources": p.sources("bnf")},
			keys:  []matchKey{{path: []string{"name"}, value: name, fold: true}}, selected: true,
			parent: edition.key, sourceURL: bnf.RecordURL(ark),
		})
		p.link(relEditedBy, edition.key, pub.key, nil)
	}
	p.addContributors(rec, edition)
	if len(vols) > 0 {
		p.addVolumes(vols, number, lang, edition)
		if len(vols) > 0 {
			edition.attrs["volume_count"] = len(p.childrenOf(edition.key, model.NodeVolume))
		}
	}
	return nil
}

// novelLanguage: the language of the text (101$a), as ISO 639-1 when known, else the
// BnF code itself (« bre », « oci ») ; never the French default of the BD profile:
// the catalogue of a novel mixes many translations.
func novelLanguage(r *bnf.Record) string {
	code := r.Get("101", "a")
	if l := bnf.Language(code); l != "" {
		return l
	}
	switch code {
	case "xxx", "und", "zxx", "mul", "mis":
		return "" // indéterminée, sans contenu linguistique, multiple…
	}
	if len(code) == 3 {
		return code
	}
	return ""
}

// bindingFormat maps the binding of 010$b to the formats of an edition.
func bindingFormat(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.HasPrefix(s, "br"):
		return "paperback"
	case strings.HasPrefix(s, "rel"), strings.HasPrefix(s, "cart"):
		return "hardcover"
	}
	return ""
}

// addContributors adds the translators, illustrators and preface writers of an
// edition (7XX with a relator code of contributorRoles), one relation per person.
func (p *planner) addContributors(rec *bnf.Record, edition *node) {
	roles := map[string][]any{}
	var order []string
	people := map[string]map[string]any{}
	for _, tag := range []string{"700", "701", "702"} {
		for _, f := range rec.Tagged(tag) {
			role := contributorRoles[f.Get("4")]
			last := strings.TrimSpace(f.Get("a"))
			if role == "" || last == "" {
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
			if _, ok := people[key]; !ok {
				order = append(order, key)
				people[key] = attrs
			}
			if !slicesContainsAny(roles[key], role) {
				roles[key] = append(roles[key], role)
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
			attrs: attrs, keys: keys, selected: true, parent: edition.key, sourceURL: edition.sourceURL,
		})
		p.link(relCreatedBy, edition.key, person.key, map[string]any{"roles": roles[key]})
	}
}

func slicesContainsAny(s []any, v any) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// childrenOf returns the nodes of a role whose parent is key.
func (p *planner) childrenOf(key, role string) []*node {
	var out []*node
	for _, n := range p.nodes {
		if n.parent == key && n.role == role {
			out = append(out, n)
		}
	}
	return out
}
