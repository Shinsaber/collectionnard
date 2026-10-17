// Package connectors holds the connectors to the external sources, one profile
// per kind of collection: BD / manga and novels (Wikidata + BnF, domain
// « books », les.collection.md), music (MusicBrainz, domain « music »,
// musique.md). They share the planner: the proposed items are matched against
// the catalogue (existing items keep their values, the source only fills the
// empty fields), then written through the catalogue import.
//
// A profile is the only place that knows its domain: it finds its item types
// and relation types by package id, then by technical name, and reports the
// missing ones instead of guessing.
package connectors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/external/musicbrainz"
	"github.com/shinsaber/colec/internal/external/wikidata"
	"github.com/shinsaber/colec/internal/model"
)

// Catalog is the part of the store the connector reads.
type Catalog interface {
	ListItemTypes(ctx context.Context, pg model.Pagination) ([]model.ItemType, int, error)
	ListRelationTypes(ctx context.Context, pg model.Pagination, sourceTypeID, targetTypeID *uuid.UUID) ([]model.RelationType, int, error)
	FindItemsByAttribute(ctx context.Context, itemTypeID uuid.UUID, path, values []string, fold bool) ([]model.Item, error)
	ListRelationsBetween(ctx context.Context, ids []uuid.UUID) ([]model.ItemRelation, error)
}

// profile: the kind of collection a connector imports. All share the planner
// (nodes, matching, import format).
type profile int

const (
	profileBDManga profile = iota // manga and comics: work → multi-volume edition → volumes
	profileNovels                 // novels: work → edition (the book, or a set of volumes)
	profileMusic                  // music (domain « music »): album → releases → tracks
	profileFilms                  // films (domain « video »): film → video releases (DVD, Blu-ray…)
)

type Connector struct {
	Catalog Catalog
	WD      *wikidata.Client
	BnF     *bnf.Client
	MB      *musicbrainz.Client
	Now     func() time.Time
	profile profile
}

// New returns the BD / Manga connector.
func New(cat Catalog, fetch *external.Client) *Connector {
	return &Connector{Catalog: cat, WD: wikidata.New(fetch), BnF: bnf.New(fetch), Now: time.Now}
}

// NewNovels returns the novels connector (same sources: Wikidata and the BnF).
func NewNovels(cat Catalog, fetch *external.Client) *Connector {
	c := New(cat, fetch)
	c.profile = profileNovels
	return c
}

// NewFilms returns the films connector (Wikidata for the film, BnF for its video releases).
func NewFilms(cat Catalog, fetch *external.Client) *Connector {
	c := New(cat, fetch)
	c.profile = profileFilms
	return c
}

// NewMusic returns the music connector (source: MusicBrainz).
func NewMusic(cat Catalog, fetch *external.Client) *Connector {
	return &Connector{Catalog: cat, MB: musicbrainz.New(fetch), Now: time.Now, profile: profileMusic}
}

// Technical names of the domain (package « books », server/internal/domains/packages).
const (
	typeManga     = "Manga"
	typeNovel     = "Novel"
	typeSaga      = "Saga"
	typeComic     = "Comic"
	typeUniverse  = "Universe"
	typeEdition   = "Edition"
	typeVolume    = "Volume"
	typePerson    = "Person"
	typePublisher = "Publisher"

	relContainsWork   = "contains_work"
	relCreatedBy      = "created_by"
	relPublishedAs    = "published_as"
	relEditedBy       = "edited_by"
	relContainsVolume = "contains_volume"

	// Package « music » (musique.md).
	typeArtist  = "Artist"
	typeAlbum   = "Album"
	typeRelease = "Release"
	typeTrack   = "Track"
	typeLabel   = "Label"

	relPerformedBy   = "performed_by"
	relIssuedAs      = "issued_as"
	relContainsTrack = "contains_track"
	relReleasedBy    = "released_by"

	// Package « video » (video.md) ; people and publishers: package « people ».
	typeFilm         = "Film"
	typeVideoRelease = "VideoRelease"

	relReleasedAs       = "released_as"
	relCredited         = "credited"
	relVideoPublishedBy = "published_by"
)

var (
	// Types required by each profile.
	itemTypeNames = map[profile][]string{
		profileBDManga: {typeManga, typeComic, typeUniverse, typeEdition, typeVolume, typePerson, typePublisher},
		profileNovels:  {typeNovel, typeSaga, typeUniverse, typeEdition, typeVolume, typePerson, typePublisher},
		profileMusic:   {typeAlbum, typeRelease, typeTrack, typeArtist, typeLabel},
		profileFilms:   {typeFilm, typeVideoRelease, typePerson, typePublisher},
	}
	relationTypeNames = map[profile][]string{
		profileBDManga: {relContainsWork, relCreatedBy, relPublishedAs, relEditedBy, relContainsVolume},
		profileNovels:  {relContainsWork, relCreatedBy, relPublishedAs, relEditedBy, relContainsVolume},
		profileMusic:   {relPerformedBy, relIssuedAs, relContainsTrack, relReleasedBy},
		profileFilms:   {relReleasedAs, relCredited, relVideoPublishedBy},
	}

	// packageIDs: fixed ids of the types in the domain packages. A type is found by this id
	// first (it may have been renamed on the instance), then by its technical name (type
	// created by hand, outside the package).
	packageIDs = map[string]uuid.UUID{
		typeUniverse:        uuid.MustParse("b0000000-0000-0000-0000-000000000010"),
		typeSaga:            uuid.MustParse("b0000000-0000-0000-0000-000000000011"),
		typeManga:           uuid.MustParse("b0000000-0000-0000-0000-000000000012"),
		typeComic:           uuid.MustParse("b0000000-0000-0000-0000-000000000013"),
		typeEdition:         uuid.MustParse("b0000000-0000-0000-0000-000000000014"),
		typeVolume:          uuid.MustParse("b0000000-0000-0000-0000-000000000015"),
		typePerson:          uuid.MustParse("b0000000-0000-0000-0000-000000000016"),
		typePublisher:       uuid.MustParse("b0000000-0000-0000-0000-000000000017"),
		typeNovel:           uuid.MustParse("b0000000-0000-0000-0000-000000000018"),
		relContainsWork:     uuid.MustParse("c0000000-0000-0000-0000-000000000011"),
		relCreatedBy:        uuid.MustParse("c0000000-0000-0000-0000-000000000012"),
		relPublishedAs:      uuid.MustParse("c0000000-0000-0000-0000-000000000013"),
		relEditedBy:         uuid.MustParse("c0000000-0000-0000-0000-000000000014"),
		relContainsVolume:   uuid.MustParse("c0000000-0000-0000-0000-000000000015"),
		typeArtist:          uuid.MustParse("b0000000-0000-0000-0000-000000000020"),
		typeAlbum:           uuid.MustParse("b0000000-0000-0000-0000-000000000021"),
		typeRelease:         uuid.MustParse("b0000000-0000-0000-0000-000000000022"),
		typeTrack:           uuid.MustParse("b0000000-0000-0000-0000-000000000023"),
		typeLabel:           uuid.MustParse("b0000000-0000-0000-0000-000000000024"),
		relPerformedBy:      uuid.MustParse("c0000000-0000-0000-0000-000000000020"),
		relIssuedAs:         uuid.MustParse("c0000000-0000-0000-0000-000000000021"),
		relContainsTrack:    uuid.MustParse("c0000000-0000-0000-0000-000000000022"),
		relReleasedBy:       uuid.MustParse("c0000000-0000-0000-0000-000000000023"),
		typeFilm:            uuid.MustParse("b0000000-0000-0000-0000-000000000030"),
		typeVideoRelease:    uuid.MustParse("b0000000-0000-0000-0000-000000000005"),
		relReleasedAs:       uuid.MustParse("c0000000-0000-0000-0000-000000000004"),
		relCredited:         uuid.MustParse("c0000000-0000-0000-0000-000000000030"),
		relVideoPublishedBy: uuid.MustParse("c0000000-0000-0000-0000-000000000031"),
	}
)

// Limits of one request.
const (
	maxEditions       = 10
	maxVolumes        = 1000
	maxWorkHits       = 20
	maxEditionHits    = 50
	maxISBNVolumeHits = 20
)

// UnavailableError and InputError are the errors shared by every connector.
type (
	UnavailableError = external.UnavailableError
	InputError       = external.InputError
)

// ID and Domain identify the connector (external.Connector).
func (c *Connector) ID() string {
	switch c.profile {
	case profileNovels:
		return "novels"
	case profileMusic:
		return "music"
	case profileFilms:
		return "films"
	}
	return "bd-manga"
}

func (c *Connector) Domain() string {
	switch c.profile {
	case profileMusic:
		return "music"
	case profileFilms:
		return "video"
	}
	return "books"
}

// EditionsByWork: the novels connector lists the editions of a Wikidata work
// (BnF work record, bib.work2bib), the music connector the releases of an album, the films connector the
// video releases of a film (BnF work record).
func (c *Connector) EditionsByWork() bool { return c.profile != profileBDManga }

// EditionCode is the code editions are searched by: the ISBN of a book, the barcode of a release.
func (c *Connector) EditionCode() string {
	if c.profile == profileMusic || c.profile == profileFilms {
		return "barcode"
	}
	return "isbn"
}

// domain holds the resolved types.
type domain struct {
	types     map[string]*model.ItemType // by technical name
	schemas   map[uuid.UUID]map[string]any
	relations map[string]uuid.UUID
}

func (c *Connector) domain(ctx context.Context) (*domain, error) {
	all := model.Pagination{Page: 1, PerPage: 1000}
	its, _, err := c.Catalog.ListItemTypes(ctx, all)
	if err != nil {
		return nil, err
	}
	rts, _, err := c.Catalog.ListRelationTypes(ctx, all, nil, nil)
	if err != nil {
		return nil, err
	}
	d := &domain{types: map[string]*model.ItemType{}, schemas: map[uuid.UUID]map[string]any{}, relations: map[string]uuid.UUID{}}
	typeByName, typeByID := map[string]*model.ItemType{}, map[uuid.UUID]*model.ItemType{}
	for i := range its {
		typeByName[its[i].Name] = &its[i]
		typeByID[its[i].ID] = &its[i]
		d.schemas[its[i].ID] = parseObject(its[i].AttributesSchema)
	}
	relByName, relByID := map[string]uuid.UUID{}, map[uuid.UUID]bool{}
	for _, rt := range rts {
		relByName[rt.Name] = rt.ID
		relByID[rt.ID] = true
	}
	var missing []string
	for _, n := range itemTypeNames[c.profile] {
		t := typeByID[packageIDs[n]]
		if t == nil {
			t = typeByName[n]
		}
		if t == nil {
			missing = append(missing, n)
			continue
		}
		d.types[n] = t
	}
	for _, n := range relationTypeNames[c.profile] {
		if relByID[packageIDs[n]] {
			d.relations[n] = packageIDs[n]
		} else if id, ok := relByName[n]; ok {
			d.relations[n] = id
		} else {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return nil, &UnavailableError{Domain: c.Domain(), Missing: missing}
	}
	return d, nil
}

func (d *domain) typeID(name string) uuid.UUID { return d.types[name].ID }

// today is the retrieval date written in `sources` (attribution of the sources).
func (c *Connector) today() string {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	return now().Format(time.DateOnly)
}

// IsNotFound reports whether err means an unknown external entity.
func IsNotFound(err error) bool { return errors.Is(err, external.ErrNotFound) }

func notFound(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{external.ErrNotFound}, args...)...)
}
