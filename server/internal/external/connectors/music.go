package connectors

import (
	"cmp"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external/musicbrainz"
	"github.com/shinsaber/colec/internal/model"
)

// Music profile (musique.md): an album is a MusicBrainz release group, an
// edition a release (the object one owns), a track a recording (the same item
// on every release that carries it). Artists and labels are matched by their
// MusicBrainz identifier; labels also by name, like the publishers of books.
// Only the core data (CC0) is read: no genre, tag or rating.

const (
	maxReleases     = 100 // releases listed for one album
	maxAlbumArtists = 5   // artists of an album looked up for their details
)

var barcodeRE = regexp.MustCompile(`^[0-9]{8,14}$`)

func musicbrainzKey(kind, id string) string { return "mb:" + kind + ":" + id }

func mbidKey(id string) matchKey {
	return matchKey{path: []string{"external_ids", "musicbrainz"}, value: id}
}

// ── Search ──────────────────────────────────────────────────────────

// searchAlbums searches release groups by title or artist.
func (c *Connector) searchAlbums(ctx context.Context, text string) ([]model.ExternalWorkHit, error) {
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	rgs, err := c.MB.SearchReleaseGroups(ctx, text, maxWorkHits)
	if err != nil {
		return nil, err
	}
	album := d.typeID(typeAlbum)
	hits := []model.ExternalWorkHit{}
	var ids []string
	for _, rg := range rgs {
		h := model.ExternalWorkHit{
			ID: rg.ID, Label: rg.Title, Description: albumDescription(&rg), ItemTypeID: album,
			SourceURL: musicbrainz.EntityURL("release-group", rg.ID),
		}
		if y, ok := year(rg.FirstReleaseDate); ok {
			h.StartYear = &y
		}
		hits = append(hits, h)
		ids = append(ids, rg.ID)
	}
	existing, err := c.existing(ctx, map[uuid.UUID][]string{album: ids}, []string{"external_ids", "musicbrainz"})
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].ExistingItemID = existing[hits[i].ID]
	}
	return hits, nil
}

// albumDescription: « Radiohead · Album », « Various Artists · Album, Compilation ».
func albumDescription(rg *musicbrainz.ReleaseGroup) string {
	types := slices.DeleteFunc(append([]string{rg.PrimaryType}, rg.SecondaryTypes...), func(s string) bool { return s == "" })
	parts := slices.DeleteFunc([]string{musicbrainz.Credit(rg.ArtistCredit), strings.Join(types, ", ")},
		func(s string) bool { return s == "" })
	return strings.Join(parts, " · ")
}

// searchReleases searches releases by title or barcode, or lists those of an album.
func (c *Connector) searchReleases(ctx context.Context, q model.ExternalEditionQuery) ([]model.ExternalEditionHit, error) {
	if q.ISBN != "" {
		return nil, InputError("this connector searches editions by title, barcode or album")
	}
	d, err := c.domain(ctx)
	if err != nil {
		return nil, err
	}
	var rels []musicbrainz.Release
	switch {
	case q.Work != "":
		if !musicbrainz.IsMBID(q.Work) {
			return nil, InputError("work must be a MusicBrainz release group identifier")
		}
		if rels, err = c.MB.ReleasesOf(ctx, q.Work, maxReleases); err != nil {
			return nil, err
		}
		// Oldest first, then by country: the first pressing, then the reissues.
		slices.SortStableFunc(rels, func(a, b musicbrainz.Release) int {
			if (a.Date == "") != (b.Date == "") {
				return cmp.Compare(btoi(a.Date == ""), btoi(b.Date == ""))
			}
			return cmp.Or(cmp.Compare(sortDate(a.Date), sortDate(b.Date)), cmp.Compare(a.Country, b.Country))
		})
	case q.Barcode != "":
		rels, err = c.MB.SearchReleases(ctx, "barcode:"+q.Barcode, maxEditionHits)
	default:
		rels, err = c.MB.SearchReleases(ctx, "release:("+musicbrainz.Escape(q.Title)+")", maxEditionHits)
	}
	if err != nil {
		return nil, err
	}
	hits := []model.ExternalEditionHit{}
	var ids []string
	for i := range rels {
		r := &rels[i]
		h := model.ExternalEditionHit{
			ID: r.ID, Title: r.Title, Publishers: labelNames(r), Artist: musicbrainz.Credit(r.ArtistCredit),
			Format: mediaSummary(r.Media), Country: r.Country, Barcode: r.Barcode, CatalogNumber: catalogNumber(r),
			SourceURL: musicbrainz.EntityURL("release", r.ID),
		}
		if y, ok := year(r.Date); ok {
			h.StartYear = &y
		}
		hits = append(hits, h)
		ids = append(ids, r.ID)
	}
	release := d.typeID(typeRelease)
	existing, err := c.existing(ctx, map[uuid.UUID][]string{release: ids}, []string{"external_ids", "musicbrainz"})
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].ExistingItemID = existing[hits[i].ID]
	}
	return hits, nil
}

func labelNames(r *musicbrainz.Release) []string {
	out := []string{}
	for _, li := range r.LabelInfo {
		if li.Label != nil && li.Label.ID != musicbrainz.NoLabel && !slices.Contains(out, li.Label.Name) {
			out = append(out, li.Label.Name)
		}
	}
	return out
}

func catalogNumber(r *musicbrainz.Release) string {
	for _, li := range r.LabelInfo {
		if n := strings.TrimSpace(li.CatalogNumber); n != "" && !strings.EqualFold(n, "[none]") {
			return n
		}
	}
	return ""
}

// mediaSummary: « CD », « 2×12" Vinyl », « 2×CD + DVD-Video ».
func mediaSummary(media []musicbrainz.Medium) string {
	var parts []string
	for i := 0; i < len(media); {
		j := i
		for j < len(media) && media[j].Format == media[i].Format {
			j++
		}
		f := media[i].Format
		if f == "" {
			f = "?"
		}
		if j-i > 1 {
			f = strconv.Itoa(j-i) + "×" + f
		}
		parts = append(parts, f)
		i = j
	}
	return strings.Join(parts, " + ")
}

// ── Plan ────────────────────────────────────────────────────────────

// addAlbum adds the album of a release group and its artists (looked up for
// their dates and Wikidata identifier).
func (p *planner) addAlbum(ctx context.Context, id string) (string, error) {
	rg, err := p.c.MB.ReleaseGroup(ctx, id)
	if err != nil {
		return "", err
	}
	return p.album(ctx, rg, true), nil
}

// album registers the node of a release group; lookup fetches the details of its artists.
func (p *planner) album(ctx context.Context, rg *musicbrainz.ReleaseGroup, lookup bool) string {
	attrs := map[string]any{
		"title":        rg.Title,
		"external_ids": withoutEmpty(map[string]any{"musicbrainz": rg.ID, "wikidata": musicbrainz.Wikidata(rg.Relations)}),
		"sources":      p.sources("musicbrainz"),
	}
	if t := enumValue(rg.PrimaryType); t != "" {
		attrs["primary_type"] = t
	}
	var secondary []any
	for _, s := range rg.SecondaryTypes {
		secondary = append(secondary, enumValue(s))
	}
	if len(secondary) > 0 {
		attrs["secondary_types"] = secondary
	}
	if y, ok := year(rg.FirstReleaseDate); ok {
		attrs["first_release_year"] = y
	}
	keys := []matchKey{mbidKey(rg.ID)}
	if qid := musicbrainz.Wikidata(rg.Relations); qid != "" {
		keys = append(keys, matchKey{path: []string{"external_ids", "wikidata"}, value: qid})
	}
	key := p.add(&node{
		key: musicbrainzKey("release-group", rg.ID), role: model.NodeWork, typeID: p.d.typeID(typeAlbum),
		attrs: attrs, keys: keys, selected: true, sourceURL: musicbrainz.EntityURL("release-group", rg.ID),
	}).key
	for i, ac := range rg.ArtistCredit {
		p.credit(ctx, key, ac, lookup && i < maxAlbumArtists)
	}
	return key
}

// credit links a node to a credited artist (performed_by), adding the artist.
func (p *planner) credit(ctx context.Context, from string, ac musicbrainz.ArtistCredit, lookup bool) {
	a := ac.Artist
	if a.ID == "" || a.ID == musicbrainz.VariousArtists {
		return // compilation: the artists are credited on the tracks
	}
	key := musicbrainzKey("artist", a.ID)
	if p.byKey[key] == nil && lookup {
		if full, err := p.c.MB.Artist(ctx, a.ID); err == nil {
			a = *full // details are optional: the credit is enough when the lookup fails
		}
	}
	p.add(&node{
		key: key, role: model.NodePerson, typeID: p.d.typeID(typeArtist), parent: from,
		attrs: p.artistAttrs(&a), keys: []matchKey{mbidKey(a.ID)}, selected: true,
		sourceURL: musicbrainz.EntityURL("artist", a.ID),
	})
	attrs := map[string]any{}
	if ac.Name != "" && ac.Name != a.Name {
		attrs["credited_as"] = ac.Name
	}
	p.link(relPerformedBy, from, key, attrs)
}

func (p *planner) artistAttrs(a *musicbrainz.Artist) map[string]any {
	attrs := map[string]any{
		"name":         a.Name,
		"external_ids": withoutEmpty(map[string]any{"musicbrainz": a.ID, "wikidata": musicbrainz.Wikidata(a.Relations)}),
		"sources":      p.sources("musicbrainz"),
	}
	if a.SortName != "" && a.SortName != a.Name {
		attrs["sort_name"] = a.SortName
	}
	if k := enumValue(a.Type); k != "" {
		attrs["kind"] = k
	}
	if isCountry(a.Country) {
		attrs["country"] = a.Country
	}
	if y, ok := year(a.LifeSpan.Begin); ok {
		attrs["begin_year"] = y
	}
	if y, ok := year(a.LifeSpan.End); ok {
		attrs["end_year"] = y
	}
	if a.Disambiguation != "" {
		attrs["disambiguation"] = a.Disambiguation
	}
	return attrs
}

// addRelease adds a release with its album (the one chosen, or the release
// group of the release), its labels and its tracks.
func (p *planner) addRelease(ctx context.Context, id, _ string) error {
	r, err := p.c.MB.Release(ctx, id)
	if err != nil {
		return err
	}
	if r.ReleaseGroup == nil {
		return notFound("release group of the MusicBrainz release %s", id)
	}
	albumKey := musicbrainzKey("release-group", r.ReleaseGroup.ID)
	if p.byKey[albumKey] == nil {
		p.album(ctx, r.ReleaseGroup, true)
	}
	key := musicbrainzKey("release", r.ID)
	p.add(&node{
		key: key, role: model.NodeEdition, typeID: p.d.typeID(typeRelease), parent: albumKey,
		attrs: p.releaseAttrs(r), keys: []matchKey{mbidKey(r.ID)}, selected: true,
		sourceURL: musicbrainz.EntityURL("release", r.ID),
	})
	p.link(relIssuedAs, albumKey, key, nil)

	// The artists of the release, then of a track, only when they differ (« feat. », compilations).
	if creditIDs(r.ArtistCredit) != creditIDs(r.ReleaseGroup.ArtistCredit) {
		for _, ac := range r.ArtistCredit {
			p.credit(ctx, key, ac, false)
		}
	}
	for _, li := range r.LabelInfo {
		l := li.Label
		if l == nil || l.ID == "" || l.ID == musicbrainz.NoLabel {
			continue
		}
		lk := musicbrainzKey("label", l.ID)
		attrs := map[string]any{
			"name": l.Name, "external_ids": map[string]any{"musicbrainz": l.ID}, "sources": p.sources("musicbrainz"),
		}
		if l.Disambiguation != "" {
			attrs["disambiguation"] = l.Disambiguation
		}
		p.add(&node{
			key: lk, role: model.NodePublisher, typeID: p.d.typeID(typeLabel), parent: key, attrs: attrs,
			keys:     []matchKey{mbidKey(l.ID), {path: []string{"name"}, value: l.Name, fold: true}},
			selected: true, sourceURL: musicbrainz.EntityURL("label", l.ID),
		})
		relAttrs := map[string]any{}
		if n := strings.TrimSpace(li.CatalogNumber); n != "" && !strings.EqualFold(n, "[none]") {
			relAttrs["catalog_number"] = n
		}
		p.link(relReleasedBy, key, lk, relAttrs)
	}
	order := 0
	for _, m := range r.Media {
		for _, t := range m.Tracks {
			if t.Recording.ID == "" {
				continue
			}
			order++
			if order > maxVolumes {
				return InputError(fmt.Sprintf("more than %d tracks", maxVolumes))
			}
			tk := p.track(&t, key)
			p.link(relContainsTrack, key, tk, map[string]any{"order": order, "disc": m.Position, "position": t.Position})
			if creditIDs(t.ArtistCredit) != creditIDs(r.ArtistCredit) {
				for _, ac := range t.ArtistCredit {
					p.credit(ctx, tk, ac, false)
				}
			}
		}
	}
	return nil
}

func (p *planner) track(t *musicbrainz.Track, release string) string {
	rec := t.Recording
	title := rec.Title
	if title == "" {
		title = t.Title
	}
	attrs := map[string]any{
		"title": title, "external_ids": map[string]any{"musicbrainz": rec.ID}, "sources": p.sources("musicbrainz"),
	}
	length := rec.Length
	if length == nil {
		length = t.Length
	}
	if length != nil && *length > 0 {
		attrs["length"] = (*length + 500) / 1000 // milliseconds → seconds
	}
	if len(rec.ISRCs) > 0 {
		attrs["isrc"] = rec.ISRCs[0]
	}
	return p.add(&node{
		key: musicbrainzKey("recording", rec.ID), role: model.NodeVolume, typeID: p.d.typeID(typeTrack),
		parent: release, attrs: attrs, keys: []matchKey{mbidKey(rec.ID)}, selected: true,
		sourceURL: musicbrainz.EntityURL("recording", rec.ID),
	}).key
}

func (p *planner) releaseAttrs(r *musicbrainz.Release) map[string]any {
	attrs := map[string]any{
		"title":        r.Title,
		"external_ids": map[string]any{"musicbrainz": r.ID},
		"sources":      p.sources("musicbrainz"),
	}
	if len(r.Media) > 0 {
		attrs["format"] = mediumFormat(r.Media[0].Format)
		attrs["media_count"] = len(r.Media)
	}
	if len(r.Date) == len("2006-01-02") {
		attrs["release_date"] = r.Date
	}
	if y, ok := year(r.Date); ok {
		attrs["release_year"] = y
	}
	if isCountry(r.Country) {
		attrs["country"] = r.Country // also XE (Europe) and XW (worldwide)
	}
	if barcodeRE.MatchString(r.Barcode) {
		attrs["barcode"] = r.Barcode
	}
	if n := catalogNumber(r); n != "" {
		attrs["catalog_number"] = n
	}
	if s := enumValue(r.Status); s != "" {
		attrs["status"] = s
	}
	if r.Packaging != "" && r.Packaging != "None" {
		attrs["packaging"] = r.Packaging
	}
	return attrs
}

// mediumFormat maps a MusicBrainz medium format (« 12" Vinyl », « Enhanced
// CD », « Hybrid SACD »…) onto the `format` enum of a release; mixed releases
// take the format of their first medium.
func mediumFormat(f string) string {
	switch {
	case strings.Contains(f, "SACD"):
		return "sacd"
	case strings.Contains(f, "Blu-ray"):
		return "blu_ray"
	case strings.Contains(f, "DVD"):
		return "dvd"
	case strings.Contains(f, "Vinyl"), strings.HasSuffix(f, `"`):
		return "vinyl"
	case strings.Contains(f, "CD"):
		return "cd"
	case strings.Contains(f, "Cassette"):
		return "cassette"
	case f == "Digital Media", f == "Download Card":
		return "digital"
	}
	return "other"
}

// enumValue turns a MusicBrainz type into an enum value: « Pseudo-Release » →
// pseudo_release, « DJ-mix » → dj_mix, « Mixtape/Street » → mixtape.
func enumValue(s string) string {
	s, _, _ = strings.Cut(strings.ToLower(s), "/")
	return strings.NewReplacer("-", "_", " ", "_").Replace(s)
}

// creditIDs identifies a credit by its artists (names credited aside).
func creditIDs(ac []musicbrainz.ArtistCredit) string {
	ids := make([]string, len(ac))
	for i, c := range ac {
		ids[i] = c.Artist.ID
	}
	return strings.Join(ids, ",")
}

func isCountry(s string) bool {
	return len(s) == 2 && s[0] >= 'A' && s[0] <= 'Z' && s[1] >= 'A' && s[1] <= 'Z'
}

// year reads the year of a MusicBrainz date (« 1997 », « 1997-06 », « 1997-06-16 »).
func year(date string) (int, bool) {
	if len(date) < 4 {
		return 0, false
	}
	y, err := strconv.Atoi(date[:4])
	return y, err == nil && y > 0
}

func withoutEmpty(m map[string]any) map[string]any {
	for k, v := range m {
		if isEmpty(v) {
			delete(m, k)
		}
	}
	return m
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// sortDate puts a partial date (« 1997 ») after the full dates of its year.
func sortDate(d string) string {
	if len(d) < 4 || len(d) > 10 {
		return d
	}
	return d + "-99-99"[len(d)-4:]
}
