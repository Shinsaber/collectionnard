package store

import (
	"context"
	"encoding/json"
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
	GetUserPasswordHash(ctx context.Context, id uuid.UUID) (string, error)

	// Accounts, invitations, password resets (P4-0, social.md)
	ListUsers(ctx context.Context, pg model.Pagination) ([]model.User, int, error)
	UpdateUser(ctx context.Context, id uuid.UUID, in model.UserUpdate) (*model.User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, in model.ProfileUpdate) (*model.User, error)
	SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	CreateInvitation(ctx context.Context, tokenHash string, createdBy uuid.UUID, role, note string, expiresAt time.Time) (*model.Invitation, error)
	GetInvitation(ctx context.Context, id uuid.UUID) (*model.Invitation, error)
	ListInvitations(ctx context.Context, createdBy *uuid.UUID) ([]model.Invitation, error)
	DeleteInvitation(ctx context.Context, id uuid.UUID) error
	CheckInvitation(ctx context.Context, tokenHash string) (time.Time, error)
	RegisterWithInvitation(ctx context.Context, tokenHash, username, email, passwordHash string) (*model.User, error)
	CreatePasswordReset(ctx context.Context, userID uuid.UUID, tokenHash string, createdBy uuid.UUID, expiresAt time.Time) error
	ResetPassword(ctx context.Context, tokenHash, passwordHash string) error

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
	// GetOfflineSnapshot: the user's collections, items and entries plus every meta-type
	// (offline copy of the mobile app).
	GetOfflineSnapshot(ctx context.Context, userID uuid.UUID) (*model.OfflineSnapshot, error)

	// Domain packages (P3-0): state on the instance, install / update by merge.
	ListDomains(ctx context.Context) ([]model.DomainInfo, error)
	InstallDomain(ctx context.Context, pkg *model.DomainPackage, dryRun bool) (*model.DomainInstallReport, error)
	RefreshInstalledDomains(ctx context.Context, except string) error

	// Tag
	ListTags(ctx context.Context, search string) ([]model.Tag, error)

	// Stats
	GetStats(ctx context.Context) (*model.Stats, error)
	GetCollectionStats(ctx context.Context, collectionID uuid.UUID, hidden []string) (*model.CollectionStats, error)

	// Visibilité et partage (P4-2)
	ListVisibleCollections(ctx context.Context, ownerID uuid.UUID, levels []string) ([]model.Collection, error)
	CreateShareLink(ctx context.Context, collectionID uuid.UUID, tokenHash string) (*model.ShareLink, error)
	ListShareLinks(ctx context.Context, collectionID uuid.UUID) ([]model.ShareLink, error)
	RevokeShareLink(ctx context.Context, collectionID, linkID uuid.UUID) error
	GetSharedCollectionID(ctx context.Context, tokenHash string) (uuid.UUID, error)
	ListMembers(ctx context.Context, pg model.Pagination) ([]model.User, int, error)

	// Historique du catalogue (P4-3a)
	GetItemHistory(ctx context.Context, itemID uuid.UUID) (*model.ItemHistory, error)
	GetItemVersion(ctx context.Context, itemID uuid.UUID, version int) (json.RawMessage, error)

	// Propositions et modération (P4-3b)
	CreateProposal(ctx context.Context, authorID uuid.UUID, in model.ProposalInput) (*model.Proposal, error)
	GetProposal(ctx context.Context, id uuid.UUID) (*model.Proposal, error)
	ListProposals(ctx context.Context, f model.ProposalFilter, pg model.Pagination) ([]model.Proposal, int, error)
	UpdateProposal(ctx context.Context, id uuid.UUID, in model.ProposalInput) (*model.Proposal, error)
	SubmitProposal(ctx context.Context, id uuid.UUID, base *model.ProposalBase) (*model.Proposal, error)
	WithdrawProposal(ctx context.Context, id uuid.UUID) (*model.Proposal, error)
	ReviewProposal(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID, comment string) (*model.Proposal, error)
	AcceptCatalogProposal(ctx context.Context, id, reviewerID uuid.UUID, payload *model.CatalogExport, comment string) (*model.ImportReport, error)
	ProposalQuotaUsage(ctx context.Context, authorID uuid.UUID) (waiting, accepted, today int, err error)
	CountProposalsToReview(ctx context.Context, reviewerID uuid.UUID) (int, error)
	EntitySnapshot(ctx context.Context, itemIDs, relationIDs []uuid.UUID) (*model.ProposalBase, error)
	ListProposalComments(ctx context.Context, proposalID uuid.UUID) ([]model.ProposalComment, error)
	CreateProposalComment(ctx context.Context, proposalID, authorID uuid.UUID, body string) (*model.ProposalComment, error)
	GetUserStats(ctx context.Context, userID uuid.UUID) (*model.MeStats, error)

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
