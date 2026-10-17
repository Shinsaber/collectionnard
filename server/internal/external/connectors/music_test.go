package connectors_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/connectors/connectorstest"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/validator"
)

// Réponses réelles de MusicBrainz enregistrées le 2026-10-17 (testdata/mb_*.json).
const (
	radiohead  = "a74b1b7f-71a5-4011-9441-d0b5e4122711"
	parlophone = "df7d1c7f-ef95-425f-8eef-445b3d7bcbd9"
	airbag     = "4a7fea2e-545b-4c63-bc9a-9943cc3a29d7"
	fitter     = "5838f978-0822-4e28-874f-e1511324ec3a" // Fitter Happier : face C, 2e disque du vinyle
)

func mbKey(kind, id string) string { return "mb:" + kind + ":" + id }

func TestMusicSearchWorks(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.MusicConnector(connectorstest.NewCatalog(t))
	if conn.ID() != "music" || conn.Domain() != "music" || conn.EditionCode() != "barcode" || !conn.EditionsByWork() {
		t.Errorf("id %s, domain %s", conn.ID(), conn.Domain())
	}
	hits, err := conn.SearchWorks(context.Background(), "OK Computer Radiohead")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 { // OK Computer and a tribute: every word in the title or the artist
		t.Fatalf("hits: %d", len(hits))
	}
	h := hits[0]
	if h.ID != connectorstest.OKComputer || h.Label != "OK Computer" || h.Description != "Radiohead · Album" ||
		h.ItemTypeID != connectorstest.TypeID(t, "Album") || h.StartYear == nil || *h.StartYear != 1997 || h.WikidataID != "" {
		t.Errorf("first hit: %+v", h)
	}
}

func TestMusicSearchEditions(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.MusicConnector(connectorstest.NewCatalog(t))
	ctx := context.Background()

	hits, err := conn.SearchEditions(ctx, model.ExternalEditionQuery{Work: connectorstest.OKComputer})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 39 {
		t.Fatalf("releases of the album: %d", len(hits))
	}
	// Oldest first: the Japanese CD of 1997-05-21.
	if hits[0].Country != "JP" || hits[0].StartYear == nil || *hits[0].StartYear != 1997 {
		t.Errorf("first: %+v", hits[0])
	}
	i := slices.IndexFunc(hits, func(h model.ExternalEditionHit) bool { return h.ID == connectorstest.OKComputerVinyl })
	if i < 0 {
		t.Fatal("vinyl missing")
	}
	v := hits[i]
	if v.Format != `2×12" Vinyl` || v.Barcode != "724385522918" || v.CatalogNumber != "NODATA 02" ||
		!slices.Equal(v.Publishers, []string{"Parlophone"}) || v.SourceURL != "https://musicbrainz.org/release/"+connectorstest.OKComputerVinyl {
		t.Errorf("vinyl: %+v", v)
	}

	hits, err = conn.SearchEditions(ctx, model.ExternalEditionQuery{Barcode: connectorstest.OKComputerUPC})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 || hits[0].Artist != "Radiohead" || hits[0].Format != "CD" {
		t.Errorf("by barcode: %+v", hits)
	}

	for _, q := range []model.ExternalEditionQuery{{ISBN: "9782723433358"}, {Work: "Q202996"}} {
		var input external.InputError
		if _, err := conn.SearchEditions(ctx, q); !errors.As(err, &input) {
			t.Errorf("%+v: %v, want InputError", q, err)
		}
	}
}

func TestMusicPreview(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.MusicConnector(connectorstest.NewCatalog(t))
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{
		Work: connectorstest.OKComputer, Editions: []string{connectorstest.OKComputerVinyl},
	})
	if err != nil {
		t.Fatal(err)
	}
	album := nodeByKey(t, plan, mbKey("release-group", connectorstest.OKComputer))
	a := attrs(t, album.Attributes)
	if album.Role != model.NodeWork || a["title"] != "OK Computer" || a["primary_type"] != "album" ||
		a["first_release_year"] != 1997.0 || jsonPath(a, "external_ids", "wikidata") != "Q202996" {
		t.Errorf("album: %v", a)
	}
	artist := nodeByKey(t, plan, mbKey("artist", radiohead))
	a = attrs(t, artist.Attributes)
	if a["name"] != "Radiohead" || a["kind"] != "group" || a["country"] != "GB" || a["begin_year"] != 1991.0 ||
		jsonPath(a, "external_ids", "wikidata") != "Q44190" || a["sort_name"] != nil {
		t.Errorf("artist: %v", a)
	}
	release := nodeByKey(t, plan, mbKey("release", connectorstest.OKComputerVinyl))
	a = attrs(t, release.Attributes)
	if release.Role != model.NodeEdition || release.Parent != album.Key || a["format"] != "vinyl" ||
		a["media_count"] != 2.0 || a["release_date"] != "1997-06-16" || a["release_year"] != 1997.0 ||
		a["country"] != "GB" || a["barcode"] != "724385522918" || a["catalog_number"] != "NODATA 02" ||
		a["status"] != "official" || a["packaging"] != "Cardboard/Paper Sleeve" {
		t.Errorf("release: %v", a)
	}
	if relation(plan, album.Key, release.Key) == nil || relation(plan, album.Key, artist.Key) == nil {
		t.Error("issued_as / performed_by of the album missing")
	}
	if relation(plan, release.Key, artist.Key) != nil {
		t.Error("the release has the credit of the album: no performed_by of its own")
	}
	label := nodeByKey(t, plan, mbKey("label", parlophone))
	if r := relation(plan, release.Key, label.Key); label.Role != model.NodePublisher || r == nil ||
		jsonValue(t, r.Attributes, "catalog_number") != "NODATA 02" {
		t.Errorf("label: %+v %+v", label, r)
	}

	var tracks []model.ExternalNode
	for _, n := range plan.Nodes {
		if n.Role == model.NodeVolume {
			tracks = append(tracks, n)
		}
	}
	if len(tracks) != 12 {
		t.Fatalf("tracks: %d", len(tracks))
	}
	a = attrs(t, nodeByKey(t, plan, mbKey("recording", airbag)).Attributes)
	if a["title"] != "Airbag" || a["length"] != 284.0 || a["isrc"] != "GBAYE9701274" {
		t.Errorf("Airbag: %v", a)
	}
	r := relation(plan, release.Key, mbKey("recording", fitter))
	if r == nil || jsonValue(t, r.Attributes, "disc") != 2.0 || jsonValue(t, r.Attributes, "position") != 1.0 ||
		jsonValue(t, r.Attributes, "order") != 7.0 {
		t.Errorf("Fitter Happier: %+v", r)
	}

	// Every proposed item and relation is valid against its schema.
	its, rts := connectorstest.Domain(t)
	for _, n := range plan.Nodes {
		i := slices.IndexFunc(its, func(it model.ItemType) bool { return it.ID == n.ItemTypeID })
		if err := validator.ValidateAttributes(its[i].AttributesSchema, n.Attributes); err != nil {
			t.Errorf("node %s invalid: %v", n.Key, err)
		}
	}
	for _, r := range plan.Relations {
		i := slices.IndexFunc(rts, func(rt model.RelationType) bool { return rt.ID == r.RelationTypeID })
		if err := validator.ValidateAttributes(rts[i].AttributesSchema, r.Attributes); err != nil {
			t.Errorf("relation %s → %s invalid: %v", r.From, r.To, err)
		}
	}
}

// Two releases of the same album share their tracks (the same recordings) and
// their album, found from the release when no work is chosen.
func TestMusicPreviewEditionsOnly(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.MusicConnector(connectorstest.NewCatalog(t))
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{
		Editions: []string{connectorstest.OKComputerVinyl, connectorstest.OKComputerCD},
	})
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	for _, n := range plan.Nodes {
		count[n.Role]++
	}
	if count[model.NodeWork] != 1 || count[model.NodeEdition] != 2 || count[model.NodeVolume] != 12 || count[model.NodePublisher] != 2 {
		t.Errorf("nodes by role: %v", count)
	}
	cd := mbKey("release", connectorstest.OKComputerCD)
	if relation(plan, cd, mbKey("recording", fitter)) == nil || relation(plan, mbKey("release-group", connectorstest.OKComputer), cd) == nil {
		t.Error("CD links missing")
	}
}

func TestMusicMatching(t *testing.T) {
	src := connectorstest.NewSources(t)
	cat := connectorstest.NewCatalog(t)
	artistID, labelID := uuid.New(), uuid.New()
	cat.Items = []model.Item{
		{ID: artistID, ItemTypeID: connectorstest.TypeID(t, "Artist"),
			Attributes: []byte(`{"name":"Radiohead","country":"GB","external_ids":{"musicbrainz":"` + radiohead + `"}}`)},
		{ID: labelID, ItemTypeID: connectorstest.TypeID(t, "Label"), Attributes: []byte(`{"name":"PARLOPHONE"}`)},
	}
	conn := src.MusicConnector(cat)
	plan, err := conn.Preview(context.Background(), model.ExternalImportRequest{Editions: []string{connectorstest.OKComputerVinyl}})
	if err != nil {
		t.Fatal(err)
	}
	artist := nodeByKey(t, plan, mbKey("artist", radiohead))
	if artist.ExistingItemID == nil || *artist.ExistingItemID != artistID || artist.Status != model.NodeUpdate {
		t.Errorf("artist: %+v", artist)
	}
	label := nodeByKey(t, plan, mbKey("label", parlophone))
	if label.ExistingItemID == nil || *label.ExistingItemID != labelID || attrs(t, label.Attributes)["name"] != "PARLOPHONE" {
		t.Errorf("label matched by name, keeping it: %+v", label)
	}
}

func TestMusicInvalidRequests(t *testing.T) {
	src := connectorstest.NewSources(t)
	conn := src.MusicConnector(connectorstest.NewCatalog(t))
	for _, req := range []model.ExternalImportRequest{
		{Work: "Q202996"},
		{Editions: []string{"ark:/12148/cb343384394"}},
		{Work: "B1392450-E666-3926-A536-22C65F834433"},
	} {
		var input external.InputError
		if _, err := conn.Preview(context.Background(), req); !errors.As(err, &input) {
			t.Errorf("%+v: %v, want InputError", req, err)
		}
	}
	// Unknown release: 404 of the source.
	if _, err := conn.Preview(context.Background(), model.ExternalImportRequest{Editions: []string{uuid.NewString()}}); !errors.Is(err, external.ErrNotFound) {
		t.Errorf("unknown release: %v", err)
	}
}

func jsonPath(m map[string]any, path ...string) any {
	var cur any = m
	for _, k := range path {
		obj, _ := cur.(map[string]any)
		cur = obj[k]
	}
	return cur
}
