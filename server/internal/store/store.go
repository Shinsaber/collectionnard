package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/itemquery"
	"github.com/shinsaber/colec/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// ItemFilters are the criteria of ListItems. Attrs and SortAttr come from the
// itemquery package, already validated against the schema of ItemTypeID.
type ItemFilters struct {
	ItemTypeID *uuid.UUID
	Search     string // full text (prefix terms, accent- and case-insensitive)
	Tag        string
	Sort       string // created_at | updated_at | relevance, optional "-" prefix
	SortAttr   *itemquery.Sort
	Attrs      []itemquery.Filter
	// RelatedTo keeps the items linked to this item, in either direction,
	// optionally through RelationTypeID only.
	RelatedTo      *uuid.UUID
	RelationTypeID *uuid.UUID
}

// Store defines all data operations.
type Store interface {
	// Health
	Ping(ctx context.Context) error

	// User
	GetUserByUsername(ctx context.Context, username string) (*model.User, string, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)

	// RefreshToken (hash SHA-256 du token, jamais le token brut)
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	ConsumeRefreshToken(ctx context.Context, tokenHash string) (uuid.UUID, error)
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
	DeleteExpiredRefreshTokens(ctx context.Context) (int64, error)

	// ItemType
	ListItemTypes(ctx context.Context, pg model.Pagination) ([]model.ItemType, int, error)
	GetItemType(ctx context.Context, id uuid.UUID) (*model.ItemType, error)
	CreateItemType(ctx context.Context, in model.ItemTypeCreate) (*model.ItemType, error)
	UpdateItemType(ctx context.Context, id uuid.UUID, in model.ItemTypeUpdate) (*model.ItemType, error)
	DeleteItemType(ctx context.Context, id uuid.UUID) error

	// RelationType
	ListRelationTypes(ctx context.Context, pg model.Pagination, sourceTypeID, targetTypeID *uuid.UUID) ([]model.RelationType, int, error)
	GetRelationType(ctx context.Context, id uuid.UUID) (*model.RelationType, error)
	CreateRelationType(ctx context.Context, in model.RelationTypeCreate) (*model.RelationType, error)
	UpdateRelationType(ctx context.Context, id uuid.UUID, in model.RelationTypeUpdate) (*model.RelationType, error)
	DeleteRelationType(ctx context.Context, id uuid.UUID) error

	// CollectionType
	ListCollectionTypes(ctx context.Context, pg model.Pagination) ([]model.CollectionType, int, error)
	GetCollectionType(ctx context.Context, id uuid.UUID) (*model.CollectionType, error)
	CreateCollectionType(ctx context.Context, in model.CollectionTypeCreate) (*model.CollectionType, error)
	UpdateCollectionType(ctx context.Context, id uuid.UUID, in model.CollectionTypeUpdate) (*model.CollectionType, error)
	DeleteCollectionType(ctx context.Context, id uuid.UUID) error

	// Item
	ListItems(ctx context.Context, pg model.Pagination, f ItemFilters) ([]model.Item, int, error)
	GetItem(ctx context.Context, id uuid.UUID) (*model.Item, error)
	CreateItem(ctx context.Context, in model.ItemCreate) (*model.Item, error)
	UpdateItem(ctx context.Context, id uuid.UUID, in model.ItemUpdate) (*model.Item, error)
	DeleteItem(ctx context.Context, id uuid.UUID) error

	// Relation
	ListItemRelations(ctx context.Context, itemID uuid.UUID, relationTypeID *uuid.UUID) ([]model.ItemRelation, error)
	GetRelation(ctx context.Context, id uuid.UUID) (*model.ItemRelation, error)
	CreateRelation(ctx context.Context, in model.ItemRelationCreate) (*model.ItemRelation, error)
	UpdateRelation(ctx context.Context, id uuid.UUID, in model.ItemRelationUpdate) (*model.ItemRelation, error)
	DeleteRelation(ctx context.Context, id uuid.UUID) error

	// Graph
	GetItemGraph(ctx context.Context, itemID uuid.UUID, maxDepth, maxNodes int, relationTypeIDs []uuid.UUID) (*model.GraphResponse, error)

	// Collection
	ListCollections(ctx context.Context, userID uuid.UUID, pg model.Pagination, collectionTypeID *uuid.UUID) ([]model.Collection, int, error)
	GetCollection(ctx context.Context, id uuid.UUID) (*model.Collection, error)
	CreateCollection(ctx context.Context, userID uuid.UUID, in model.CollectionCreate) (*model.Collection, error)
	UpdateCollection(ctx context.Context, id uuid.UUID, in model.CollectionUpdate) (*model.Collection, error)
	DeleteCollection(ctx context.Context, id uuid.UUID) error

	// CollectionItem
	ListCollectionItems(ctx context.Context, collectionID uuid.UUID, pg model.Pagination, itemTypeID *uuid.UUID, sort string) ([]model.CollectionItem, int, error)
	AddCollectionItem(ctx context.Context, collectionID uuid.UUID, in model.CollectionItemAdd) (*model.CollectionItem, error)
	UpdateCollectionItem(ctx context.Context, collectionID, itemID uuid.UUID, in model.CollectionItemUpdate) (*model.CollectionItem, error)
	RemoveCollectionItem(ctx context.Context, collectionID, itemID uuid.UUID) error

	// Item collection membership
	ListItemCollections(ctx context.Context, userID, itemID uuid.UUID) ([]model.ItemCollectionMembership, error)
	// ListRelatedItemCollections: items linked to itemID (either direction) that are in
	// collections of userID, with those collections.
	ListRelatedItemCollections(ctx context.Context, userID, itemID uuid.UUID) ([]model.RelatedItemCollections, error)

	// Tag
	ListTags(ctx context.Context, search string) ([]model.Tag, error)

	// Stats
	GetStats(ctx context.Context) (*model.Stats, error)

	// Import / export (catalogue only: meta-types, items, relations, tags)
	ExportCatalog(ctx context.Context) (*model.CatalogExport, error)
	// ImportCatalog upserts the catalogue by id in one transaction, rolled back on
	// any validation error or when dryRun is set. A returned error is an
	// infrastructure failure; validation problems are in the report.
	ImportCatalog(ctx context.Context, cat *model.CatalogExport, dryRun bool) (*model.ImportReport, error)
	// ImportCollection restores a collection export into the account of userID
	// (same contract as ImportCatalog: validation errors are in the report).
	ImportCollection(ctx context.Context, userID uuid.UUID, exp *model.CollectionExport, dryRun bool) (*model.ImportReport, error)

	// External sources (matching of the items proposed by a connector)
	FindItemsByAttribute(ctx context.Context, itemTypeID uuid.UUID, path, values []string, fold bool) ([]model.Item, error)
	ListRelationsBetween(ctx context.Context, ids []uuid.UUID) ([]model.ItemRelation, error)

	// Uploads: file names (without path) referenced by any attribute (items,
	// relations, collection items), for the purge of unused images.
	ReferencedUploads(ctx context.Context) ([]string, error)
}
