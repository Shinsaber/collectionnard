package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Formats and version of the export files (GET /export, GET /collections/{id}/export).
const (
	CatalogFormat    = "collectionnard.catalog"
	CollectionFormat = "collectionnard.collection"
	ExportVersion    = 1
)

// CatalogExport is the whole catalogue: meta-types, items (with tag names) and
// relations, each list sorted by id. It is also the body of POST /import.
type CatalogExport struct {
	Format          string               `json:"format"`
	Version         int                  `json:"version"`
	ExportedAt      *time.Time           `json:"exported_at,omitempty"`
	ItemTypes       []ExportItemType     `json:"item_types"`
	CollectionTypes []ExportMetaType     `json:"collection_types"`
	RelationTypes   []ExportRelationType `json:"relation_types"`
	Items           []ExportItem         `json:"items"`
	Relations       []ExportRelation     `json:"relations"`
}

// ExportItemType is an ItemType in an export. DisplayOrder absent on import: 100
// for a new type, the stored value is kept otherwise.
type ExportItemType struct {
	ExportMetaType
	DisplayOrder *int `json:"display_order,omitempty"`
}

// ExportMetaType is a CollectionType in an export, and the common part of the other meta-types.
type ExportMetaType struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	AttributesSchema json.RawMessage `json:"attributes_schema"`
}

// ExportRelationType is a RelationType with its allowed source / target item types.
type ExportRelationType struct {
	ExportMetaType
	SourceItemTypeIDs []uuid.UUID `json:"source_item_type_ids"`
	TargetItemTypeIDs []uuid.UUID `json:"target_item_type_ids"`
}

type ExportItem struct {
	ID         uuid.UUID       `json:"id"`
	ItemTypeID uuid.UUID       `json:"item_type_id"`
	Attributes json.RawMessage `json:"attributes"`
	Tags       []string        `json:"tags,omitempty"`
}

type ExportRelation struct {
	ID             uuid.UUID       `json:"id"`
	RelationTypeID uuid.UUID       `json:"relation_type_id"`
	SourceItemID   uuid.UUID       `json:"source_item_id"`
	TargetItemID   uuid.UUID       `json:"target_item_id"`
	Attributes     json.RawMessage `json:"attributes"`
}

// Entity names used in import reports.
const (
	EntityFile            = "file"
	EntityItemTypes       = "item_types"
	EntityCollectionTypes = "collection_types"
	EntityRelationTypes   = "relation_types"
	EntityItems           = "items"
	EntityRelations       = "relations"
	EntityCollection      = "collection"
)

// ImportCounts tallies the outcome of the entities of one list.
type ImportCounts struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Removed   int `json:"removed,omitempty"` // collection import: items no longer in the file
}

// ImportError locates one rejected entity (Index -1: the file itself).
type ImportError struct {
	Entity  string     `json:"entity"`
	Index   int        `json:"index"`
	ID      *uuid.UUID `json:"id,omitempty"`
	Message string     `json:"message"`
}

// MaxImportErrors caps the errors listed in a report.
const MaxImportErrors = 100

// ImportReport is the result of POST /import.
type ImportReport struct {
	DryRun       bool                     `json:"dry_run"`
	Applied      bool                     `json:"applied"`
	CollectionID *uuid.UUID               `json:"collection_id,omitempty"`
	Counts       map[string]*ImportCounts `json:"counts"`
	Errors       []ImportError            `json:"errors"`
}

// NewImportReport returns an empty report with every list counted.
func NewImportReport(dryRun bool) *ImportReport {
	r := &ImportReport{DryRun: dryRun, Counts: map[string]*ImportCounts{}, Errors: []ImportError{}}
	for _, e := range []string{EntityItemTypes, EntityCollectionTypes, EntityRelationTypes, EntityItems, EntityRelations} {
		r.Counts[e] = &ImportCounts{}
	}
	return r
}

// NewCollectionImportReport returns an empty report for POST /collections/import.
func NewCollectionImportReport(dryRun bool) *ImportReport {
	return &ImportReport{DryRun: dryRun, Errors: []ImportError{}, Counts: map[string]*ImportCounts{
		EntityCollection: {}, EntityItems: {},
	}}
}

// Fail records an error (beyond MaxImportErrors, errors are no longer listed
// but the import still fails).
func (r *ImportReport) Fail(entity string, index int, id *uuid.UUID, message string) {
	if len(r.Errors) < MaxImportErrors {
		r.Errors = append(r.Errors, ImportError{Entity: entity, Index: index, ID: id, Message: message})
	}
}

// CollectionExport is one collection of a user and all its items.
type CollectionExport struct {
	Format     string                 `json:"format"`
	Version    int                    `json:"version"`
	ExportedAt time.Time              `json:"exported_at"`
	Collection CollectionExportHeader `json:"collection"`
	Items      []CollectionExportItem `json:"items"`
}

type CollectionExportHeader struct {
	ID             uuid.UUID      `json:"id"`
	Name           string         `json:"name"`
	Description    *string        `json:"description,omitempty"`
	CollectionType ExportMetaType `json:"collection_type"`
}

type CollectionExportItem struct {
	ItemID         uuid.UUID       `json:"item_id"`
	ItemType       string          `json:"item_type"`
	Label          string          `json:"label"`
	ItemAttributes json.RawMessage `json:"item_attributes"`
	Attributes     json.RawMessage `json:"attributes"`
	AddedAt        time.Time       `json:"added_at"`
}
