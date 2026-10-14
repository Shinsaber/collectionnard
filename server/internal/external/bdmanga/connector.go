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
	"strings"
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

// Technical names of the domain (db/seed_bd_manga.sql).
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
)

// Limits of one request.
const (
	maxEditions       = 10
	maxVolumes        = 1000
	maxWorkHits       = 20
	maxEditionHits    = 50
	maxISBNVolumeHits = 20
)

// UnavailableError: the catalogue lacks types the connector writes.
type UnavailableError struct {
	Missing []string
}

func (e *UnavailableError) Error() string {
	return "BD / Manga types missing from the catalogue: " + strings.Join(e.Missing, ", ")
}

// InputError is an invalid request (unknown key, malformed identifier…).
type InputError string

func (e InputError) Error() string { return string(e) }

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
	for i := range its {
		d.types[its[i].Name] = &its[i]
		d.schemas[its[i].ID] = parseObject(its[i].AttributesSchema)
	}
	for _, rt := range rts {
		d.relations[rt.Name] = rt.ID
	}
	var missing []string
	for _, n := range itemTypeNames {
		if d.types[n] == nil {
			missing = append(missing, n)
		}
	}
	for _, n := range relationTypeNames {
		if _, ok := d.relations[n]; !ok {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return nil, &UnavailableError{Missing: missing}
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
