// Package bdmanga is the BD / Manga connector: it searches works on Wikidata
// and French editions on the BnF, then proposes the matching items of the
// domain of db/seed_bd_manga.sql (les.collection.md), matched against the
// catalogue: existing items keep their values, the source only fills the empty
// fields.
//
// The connector is the only place that knows the domain: it finds its item
// types and relation types by technical name (Manga, Comic…, created_by…), and
// reports the missing ones instead of guessing.
package bdmanga

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/bnf"
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

type Connector struct {
	Catalog Catalog
	WD      *wikidata.Client
	BnF     *bnf.Client
	Now     func() time.Time
}

func New(cat Catalog, fetch *external.Client) *Connector {
	return &Connector{Catalog: cat, WD: wikidata.New(fetch), BnF: bnf.New(fetch), Now: time.Now}
}

// Technical names of the domain (package « books », server/internal/domains/packages).
const (
	typeManga     = "Manga"
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
)

var (
	itemTypeNames     = []string{typeManga, typeComic, typeUniverse, typeEdition, typeVolume, typePerson, typePublisher}
	relationTypeNames = []string{relContainsWork, relCreatedBy, relPublishedAs, relEditedBy, relContainsVolume}

	// packageIDs: fixed ids of the types in the « books » package. A type is found by this id
	// first (it may have been renamed on the instance), then by its technical name (type
	// created by hand, outside the package).
	packageIDs = map[string]uuid.UUID{
		typeUniverse:      uuid.MustParse("b0000000-0000-0000-0000-000000000010"),
		typeManga:         uuid.MustParse("b0000000-0000-0000-0000-000000000012"),
		typeComic:         uuid.MustParse("b0000000-0000-0000-0000-000000000013"),
		typeEdition:       uuid.MustParse("b0000000-0000-0000-0000-000000000014"),
		typeVolume:        uuid.MustParse("b0000000-0000-0000-0000-000000000015"),
		typePerson:        uuid.MustParse("b0000000-0000-0000-0000-000000000016"),
		typePublisher:     uuid.MustParse("b0000000-0000-0000-0000-000000000017"),
		relContainsWork:   uuid.MustParse("c0000000-0000-0000-0000-000000000011"),
		relCreatedBy:      uuid.MustParse("c0000000-0000-0000-0000-000000000012"),
		relPublishedAs:    uuid.MustParse("c0000000-0000-0000-0000-000000000013"),
		relEditedBy:       uuid.MustParse("c0000000-0000-0000-0000-000000000014"),
		relContainsVolume: uuid.MustParse("c0000000-0000-0000-0000-000000000015"),
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
func (c *Connector) ID() string     { return "bd-manga" }
func (c *Connector) Domain() string { return "books" }

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
	for _, n := range itemTypeNames {
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
	for _, n := range relationTypeNames {
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

// today is the retrieval date written in `sources` (attribution of the BnF).
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
