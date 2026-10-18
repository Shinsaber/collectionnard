package testutil

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/config"
	"github.com/shinsaber/colec/internal/handler"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

const TestSecret = "test-secret-test-secret-test-secret-00"

// TestAccessTTL is the access token lifetime used by the test servers.
const TestAccessTTL = 15 * time.Minute

// MockStore implements store.Store with configurable function fields.
type MockStore struct {
	PingFn              func(ctx context.Context) error
	GetUserByUsernameFn func(ctx context.Context, username string) (*model.User, string, error)
	GetUserByIDFn       func(ctx context.Context, id uuid.UUID) (*model.User, error)
	// Accounts (P4-0)
	GetUserPasswordHashFn        func(ctx context.Context, id uuid.UUID) (string, error)
	ListUsersFn                  func(ctx context.Context, pg model.Pagination) ([]model.User, int, error)
	UpdateUserFn                 func(ctx context.Context, id uuid.UUID, in model.UserUpdate) (*model.User, error)
	UpdateProfileFn              func(ctx context.Context, id uuid.UUID, in model.ProfileUpdate) (*model.User, error)
	SetPasswordFn                func(ctx context.Context, id uuid.UUID, passwordHash string) error
	CreateInvitationFn           func(ctx context.Context, tokenHash string, createdBy uuid.UUID, role, note string, expiresAt time.Time) (*model.Invitation, error)
	GetInvitationFn              func(ctx context.Context, id uuid.UUID) (*model.Invitation, error)
	ListInvitationsFn            func(ctx context.Context, createdBy *uuid.UUID) ([]model.Invitation, error)
	DeleteInvitationFn           func(ctx context.Context, id uuid.UUID) error
	CheckInvitationFn            func(ctx context.Context, tokenHash string) (time.Time, error)
	RegisterWithInvitationFn     func(ctx context.Context, tokenHash, username, email, passwordHash string) (*model.User, error)
	CreatePasswordResetFn        func(ctx context.Context, userID uuid.UUID, tokenHash string, createdBy uuid.UUID, expiresAt time.Time) error
	ResetPasswordFn              func(ctx context.Context, tokenHash, passwordHash string) error
	CreateRefreshTokenFn         func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	ConsumeRefreshTokenFn        func(ctx context.Context, tokenHash string) (uuid.UUID, error)
	DeleteRefreshTokenFn         func(ctx context.Context, tokenHash string) error
	DeleteExpiredRefreshTokensFn func(ctx context.Context) (int64, error)
	ListItemTypesFn              func(ctx context.Context, pg model.Pagination) ([]model.ItemType, int, error)
	GetItemTypeFn                func(ctx context.Context, id uuid.UUID) (*model.ItemType, error)
	CreateItemTypeFn             func(ctx context.Context, in model.ItemTypeCreate) (*model.ItemType, error)
	UpdateItemTypeFn             func(ctx context.Context, id uuid.UUID, in model.ItemTypeUpdate) (*model.ItemType, error)
	DeleteItemTypeFn             func(ctx context.Context, id uuid.UUID) error
	ListRelationTypesFn          func(ctx context.Context, pg model.Pagination, s, t *uuid.UUID) ([]model.RelationType, int, error)
	GetRelationTypeFn            func(ctx context.Context, id uuid.UUID) (*model.RelationType, error)
	CreateRelationTypeFn         func(ctx context.Context, in model.RelationTypeCreate) (*model.RelationType, error)
	UpdateRelationTypeFn         func(ctx context.Context, id uuid.UUID, in model.RelationTypeUpdate) (*model.RelationType, error)
	DeleteRelationTypeFn         func(ctx context.Context, id uuid.UUID) error
	ListCollectionTypesFn        func(ctx context.Context, pg model.Pagination) ([]model.CollectionType, int, error)
	GetCollectionTypeFn          func(ctx context.Context, id uuid.UUID) (*model.CollectionType, error)
	CreateCollectionTypeFn       func(ctx context.Context, in model.CollectionTypeCreate) (*model.CollectionType, error)
	UpdateCollectionTypeFn       func(ctx context.Context, id uuid.UUID, in model.CollectionTypeUpdate) (*model.CollectionType, error)
	DeleteCollectionTypeFn       func(ctx context.Context, id uuid.UUID) error
	ListItemsFn                  func(ctx context.Context, pg model.Pagination, f store.ItemFilters) ([]model.Item, int, error)
	GetItemFn                    func(ctx context.Context, id uuid.UUID) (*model.Item, error)
	CreateItemFn                 func(ctx context.Context, in model.ItemCreate) (*model.Item, error)
	UpdateItemFn                 func(ctx context.Context, id uuid.UUID, in model.ItemUpdate) (*model.Item, error)
	DeleteItemFn                 func(ctx context.Context, id uuid.UUID) error
	ListItemRelationsFn          func(ctx context.Context, itemID uuid.UUID, rtID *uuid.UUID) ([]model.ItemRelation, error)
	GetRelationFn                func(ctx context.Context, id uuid.UUID) (*model.ItemRelation, error)
	CreateRelationFn             func(ctx context.Context, in model.ItemRelationCreate) (*model.ItemRelation, error)
	UpdateRelationFn             func(ctx context.Context, id uuid.UUID, in model.ItemRelationUpdate) (*model.ItemRelation, error)
	DeleteRelationFn             func(ctx context.Context, id uuid.UUID) error
	GetItemGraphFn               func(ctx context.Context, id uuid.UUID, depth, maxNodes int, rtIDs []uuid.UUID) (*model.GraphResponse, error)
	ListCollectionsFn            func(ctx context.Context, userID uuid.UUID, pg model.Pagination, ctID *uuid.UUID) ([]model.Collection, int, error)
	GetCollectionFn              func(ctx context.Context, id uuid.UUID) (*model.Collection, error)
	CreateCollectionFn           func(ctx context.Context, userID uuid.UUID, in model.CollectionCreate) (*model.Collection, error)
	UpdateCollectionFn           func(ctx context.Context, id uuid.UUID, in model.CollectionUpdate) (*model.Collection, error)
	DeleteCollectionFn           func(ctx context.Context, id uuid.UUID) error
	ListCollectionItemsFn        func(ctx context.Context, colID uuid.UUID, pg model.Pagination, itID *uuid.UUID, sort string) ([]model.CollectionItem, int, error)
	AddCollectionItemFn          func(ctx context.Context, colID uuid.UUID, in model.CollectionItemAdd) (*model.CollectionItem, error)
	UpdateCollectionItemFn       func(ctx context.Context, colID, itemID uuid.UUID, in model.CollectionItemUpdate) (*model.CollectionItem, error)
	RemoveCollectionItemFn       func(ctx context.Context, colID, itemID uuid.UUID) error
	ListItemCollectionsFn        func(ctx context.Context, userID, itemID uuid.UUID) ([]model.ItemCollectionMembership, error)
	ListTagsFn                   func(ctx context.Context, search string) ([]model.Tag, error)
	GetStatsFn                   func(ctx context.Context) (*model.Stats, error)
	GetCollectionStatsFn         func(ctx context.Context, collectionID uuid.UUID, hidden []string) (*model.CollectionStats, error)
	ListVisibleCollectionsFn     func(ctx context.Context, ownerID uuid.UUID, levels []string) ([]model.Collection, error)
	CreateShareLinkFn            func(ctx context.Context, collectionID uuid.UUID, tokenHash string) (*model.ShareLink, error)
	ListShareLinksFn             func(ctx context.Context, collectionID uuid.UUID) ([]model.ShareLink, error)
	RevokeShareLinkFn            func(ctx context.Context, collectionID, linkID uuid.UUID) error
	GetSharedCollectionIDFn      func(ctx context.Context, tokenHash string) (uuid.UUID, error)
	ListMembersFn                func(ctx context.Context, pg model.Pagination) ([]model.User, int, error)
	GetItemHistoryFn             func(ctx context.Context, itemID uuid.UUID) (*model.ItemHistory, error)
	GetItemVersionFn             func(ctx context.Context, itemID uuid.UUID, version int) (json.RawMessage, error)
	CreateProposalFn             func(ctx context.Context, authorID uuid.UUID, in model.ProposalInput) (*model.Proposal, error)
	GetProposalFn                func(ctx context.Context, id uuid.UUID) (*model.Proposal, error)
	ListProposalsFn              func(ctx context.Context, f model.ProposalFilter, pg model.Pagination) ([]model.Proposal, int, error)
	UpdateProposalFn             func(ctx context.Context, id uuid.UUID, in model.ProposalInput) (*model.Proposal, error)
	SubmitProposalFn             func(ctx context.Context, id uuid.UUID, base *model.ProposalBase) (*model.Proposal, error)
	WithdrawProposalFn           func(ctx context.Context, id uuid.UUID) (*model.Proposal, error)
	ReviewProposalFn             func(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID, comment string) (*model.Proposal, error)
	AcceptCatalogProposalFn      func(ctx context.Context, id, reviewerID uuid.UUID, payload *model.CatalogExport, comment string) (*model.ImportReport, error)
	ProposalQuotaUsageFn         func(ctx context.Context, authorID uuid.UUID) (int, int, int, error)
	CountProposalsToReviewFn     func(ctx context.Context, reviewerID uuid.UUID) (int, error)
	EntitySnapshotFn             func(ctx context.Context, itemIDs, relationIDs []uuid.UUID) (*model.ProposalBase, error)
	ListProposalCommentsFn       func(ctx context.Context, proposalID uuid.UUID) ([]model.ProposalComment, error)
	CreateProposalCommentFn      func(ctx context.Context, proposalID, authorID uuid.UUID, body string) (*model.ProposalComment, error)
	GetUserStatsFn               func(ctx context.Context, userID uuid.UUID) (*model.MeStats, error)
	ExportCatalogFn              func(ctx context.Context) (*model.CatalogExport, error)
	ImportCatalogFn              func(ctx context.Context, cat *model.CatalogExport, dryRun bool) (*model.ImportReport, error)
	ImportCollectionFn           func(ctx context.Context, userID uuid.UUID, exp *model.CollectionExport, dryRun bool) (*model.ImportReport, error)
	FindItemsByAttributeFn       func(ctx context.Context, itemTypeID uuid.UUID, path, values []string, fold bool) ([]model.Item, error)
	ListRelationsBetweenFn       func(ctx context.Context, ids []uuid.UUID) ([]model.ItemRelation, error)
	ReferencedUploadsFn          func(ctx context.Context) ([]string, error)
	ListRelatedItemCollectionsFn func(ctx context.Context, userID, itemID uuid.UUID) ([]model.RelatedItemCollections, error)
	GetOfflineSnapshotFn         func(ctx context.Context, userID uuid.UUID) (*model.OfflineSnapshot, error)
	ListDomainsFn                func(ctx context.Context) ([]model.DomainInfo, error)
	InstallDomainFn              func(ctx context.Context, pkg *model.DomainPackage, dryRun bool) (*model.DomainInstallReport, error)
	RefreshInstalledDomainsFn    func(ctx context.Context, except string) error
}

func (m *MockStore) Ping(ctx context.Context) error {
	if m.PingFn != nil {
		return m.PingFn(ctx)
	}
	return nil
}
func (m *MockStore) GetUserByUsername(ctx context.Context, username string) (*model.User, string, error) {
	if m.GetUserByUsernameFn != nil {
		return m.GetUserByUsernameFn(ctx, username)
	}
	return nil, "", store.ErrNotFound
}
func (m *MockStore) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	if m.GetUserByIDFn != nil {
		return m.GetUserByIDFn(ctx, id)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	if m.CreateRefreshTokenFn != nil {
		return m.CreateRefreshTokenFn(ctx, userID, tokenHash, expiresAt)
	}
	return nil
}
func (m *MockStore) ConsumeRefreshToken(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	if m.ConsumeRefreshTokenFn != nil {
		return m.ConsumeRefreshTokenFn(ctx, tokenHash)
	}
	return uuid.Nil, store.ErrNotFound
}
func (m *MockStore) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	if m.DeleteRefreshTokenFn != nil {
		return m.DeleteRefreshTokenFn(ctx, tokenHash)
	}
	return nil
}
func (m *MockStore) DeleteExpiredRefreshTokens(ctx context.Context) (int64, error) {
	if m.DeleteExpiredRefreshTokensFn != nil {
		return m.DeleteExpiredRefreshTokensFn(ctx)
	}
	return 0, nil
}
func (m *MockStore) ListItemTypes(ctx context.Context, pg model.Pagination) ([]model.ItemType, int, error) {
	if m.ListItemTypesFn != nil {
		return m.ListItemTypesFn(ctx, pg)
	}
	return nil, 0, nil
}
func (m *MockStore) GetItemType(ctx context.Context, id uuid.UUID) (*model.ItemType, error) {
	if m.GetItemTypeFn != nil {
		return m.GetItemTypeFn(ctx, id)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) CreateItemType(ctx context.Context, in model.ItemTypeCreate) (*model.ItemType, error) {
	if m.CreateItemTypeFn != nil {
		return m.CreateItemTypeFn(ctx, in)
	}
	return nil, nil
}
func (m *MockStore) UpdateItemType(ctx context.Context, id uuid.UUID, in model.ItemTypeUpdate) (*model.ItemType, error) {
	if m.UpdateItemTypeFn != nil {
		return m.UpdateItemTypeFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) DeleteItemType(ctx context.Context, id uuid.UUID) error {
	if m.DeleteItemTypeFn != nil {
		return m.DeleteItemTypeFn(ctx, id)
	}
	return store.ErrNotFound
}
func (m *MockStore) ListRelationTypes(ctx context.Context, pg model.Pagination, s, t *uuid.UUID) ([]model.RelationType, int, error) {
	if m.ListRelationTypesFn != nil {
		return m.ListRelationTypesFn(ctx, pg, s, t)
	}
	return nil, 0, nil
}
func (m *MockStore) GetRelationType(ctx context.Context, id uuid.UUID) (*model.RelationType, error) {
	if m.GetRelationTypeFn != nil {
		return m.GetRelationTypeFn(ctx, id)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) CreateRelationType(ctx context.Context, in model.RelationTypeCreate) (*model.RelationType, error) {
	if m.CreateRelationTypeFn != nil {
		return m.CreateRelationTypeFn(ctx, in)
	}
	return nil, nil
}
func (m *MockStore) UpdateRelationType(ctx context.Context, id uuid.UUID, in model.RelationTypeUpdate) (*model.RelationType, error) {
	if m.UpdateRelationTypeFn != nil {
		return m.UpdateRelationTypeFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) DeleteRelationType(ctx context.Context, id uuid.UUID) error {
	if m.DeleteRelationTypeFn != nil {
		return m.DeleteRelationTypeFn(ctx, id)
	}
	return store.ErrNotFound
}
func (m *MockStore) ListCollectionTypes(ctx context.Context, pg model.Pagination) ([]model.CollectionType, int, error) {
	if m.ListCollectionTypesFn != nil {
		return m.ListCollectionTypesFn(ctx, pg)
	}
	return nil, 0, nil
}
func (m *MockStore) GetCollectionType(ctx context.Context, id uuid.UUID) (*model.CollectionType, error) {
	if m.GetCollectionTypeFn != nil {
		return m.GetCollectionTypeFn(ctx, id)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) CreateCollectionType(ctx context.Context, in model.CollectionTypeCreate) (*model.CollectionType, error) {
	if m.CreateCollectionTypeFn != nil {
		return m.CreateCollectionTypeFn(ctx, in)
	}
	return nil, nil
}
func (m *MockStore) UpdateCollectionType(ctx context.Context, id uuid.UUID, in model.CollectionTypeUpdate) (*model.CollectionType, error) {
	if m.UpdateCollectionTypeFn != nil {
		return m.UpdateCollectionTypeFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) DeleteCollectionType(ctx context.Context, id uuid.UUID) error {
	if m.DeleteCollectionTypeFn != nil {
		return m.DeleteCollectionTypeFn(ctx, id)
	}
	return store.ErrNotFound
}
func (m *MockStore) ListItems(ctx context.Context, pg model.Pagination, f store.ItemFilters) ([]model.Item, int, error) {
	if m.ListItemsFn != nil {
		return m.ListItemsFn(ctx, pg, f)
	}
	return nil, 0, nil
}
func (m *MockStore) GetItem(ctx context.Context, id uuid.UUID) (*model.Item, error) {
	if m.GetItemFn != nil {
		return m.GetItemFn(ctx, id)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) CreateItem(ctx context.Context, in model.ItemCreate) (*model.Item, error) {
	if m.CreateItemFn != nil {
		return m.CreateItemFn(ctx, in)
	}
	return nil, nil
}
func (m *MockStore) UpdateItem(ctx context.Context, id uuid.UUID, in model.ItemUpdate) (*model.Item, error) {
	if m.UpdateItemFn != nil {
		return m.UpdateItemFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) DeleteItem(ctx context.Context, id uuid.UUID) error {
	if m.DeleteItemFn != nil {
		return m.DeleteItemFn(ctx, id)
	}
	return store.ErrNotFound
}
func (m *MockStore) ListItemRelations(ctx context.Context, itemID uuid.UUID, rtID *uuid.UUID) ([]model.ItemRelation, error) {
	if m.ListItemRelationsFn != nil {
		return m.ListItemRelationsFn(ctx, itemID, rtID)
	}
	return nil, nil
}
func (m *MockStore) GetRelation(ctx context.Context, id uuid.UUID) (*model.ItemRelation, error) {
	if m.GetRelationFn != nil {
		return m.GetRelationFn(ctx, id)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) CreateRelation(ctx context.Context, in model.ItemRelationCreate) (*model.ItemRelation, error) {
	if m.CreateRelationFn != nil {
		return m.CreateRelationFn(ctx, in)
	}
	return nil, nil
}
func (m *MockStore) UpdateRelation(ctx context.Context, id uuid.UUID, in model.ItemRelationUpdate) (*model.ItemRelation, error) {
	if m.UpdateRelationFn != nil {
		return m.UpdateRelationFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) DeleteRelation(ctx context.Context, id uuid.UUID) error {
	if m.DeleteRelationFn != nil {
		return m.DeleteRelationFn(ctx, id)
	}
	return store.ErrNotFound
}
func (m *MockStore) GetItemGraph(ctx context.Context, id uuid.UUID, depth, maxNodes int, rtIDs []uuid.UUID) (*model.GraphResponse, error) {
	if m.GetItemGraphFn != nil {
		return m.GetItemGraphFn(ctx, id, depth, maxNodes, rtIDs)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) ListCollections(ctx context.Context, userID uuid.UUID, pg model.Pagination, ctID *uuid.UUID) ([]model.Collection, int, error) {
	if m.ListCollectionsFn != nil {
		return m.ListCollectionsFn(ctx, userID, pg, ctID)
	}
	return nil, 0, nil
}
func (m *MockStore) GetCollection(ctx context.Context, id uuid.UUID) (*model.Collection, error) {
	if m.GetCollectionFn != nil {
		return m.GetCollectionFn(ctx, id)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) CreateCollection(ctx context.Context, userID uuid.UUID, in model.CollectionCreate) (*model.Collection, error) {
	if m.CreateCollectionFn != nil {
		return m.CreateCollectionFn(ctx, userID, in)
	}
	return nil, nil
}
func (m *MockStore) UpdateCollection(ctx context.Context, id uuid.UUID, in model.CollectionUpdate) (*model.Collection, error) {
	if m.UpdateCollectionFn != nil {
		return m.UpdateCollectionFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	if m.DeleteCollectionFn != nil {
		return m.DeleteCollectionFn(ctx, id)
	}
	return store.ErrNotFound
}
func (m *MockStore) ListCollectionItems(ctx context.Context, colID uuid.UUID, pg model.Pagination, itID *uuid.UUID, sort string) ([]model.CollectionItem, int, error) {
	if m.ListCollectionItemsFn != nil {
		return m.ListCollectionItemsFn(ctx, colID, pg, itID, sort)
	}
	return nil, 0, nil
}
func (m *MockStore) AddCollectionItem(ctx context.Context, colID uuid.UUID, in model.CollectionItemAdd) (*model.CollectionItem, error) {
	if m.AddCollectionItemFn != nil {
		return m.AddCollectionItemFn(ctx, colID, in)
	}
	return nil, nil
}
func (m *MockStore) UpdateCollectionItem(ctx context.Context, colID, itemID uuid.UUID, in model.CollectionItemUpdate) (*model.CollectionItem, error) {
	if m.UpdateCollectionItemFn != nil {
		return m.UpdateCollectionItemFn(ctx, colID, itemID, in)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) RemoveCollectionItem(ctx context.Context, colID, itemID uuid.UUID) error {
	if m.RemoveCollectionItemFn != nil {
		return m.RemoveCollectionItemFn(ctx, colID, itemID)
	}
	return store.ErrNotFound
}
func (m *MockStore) ListItemCollections(ctx context.Context, userID, itemID uuid.UUID) ([]model.ItemCollectionMembership, error) {
	if m.ListItemCollectionsFn != nil {
		return m.ListItemCollectionsFn(ctx, userID, itemID)
	}
	return nil, nil
}
func (m *MockStore) ListTags(ctx context.Context, search string) ([]model.Tag, error) {
	if m.ListTagsFn != nil {
		return m.ListTagsFn(ctx, search)
	}
	return nil, nil
}
func (m *MockStore) GetStats(ctx context.Context) (*model.Stats, error) {
	if m.GetStatsFn != nil {
		return m.GetStatsFn(ctx)
	}
	return &model.Stats{}, nil
}

func (m *MockStore) GetCollectionStats(ctx context.Context, collectionID uuid.UUID, hidden []string) (*model.CollectionStats, error) {
	if m.GetCollectionStatsFn != nil {
		return m.GetCollectionStatsFn(ctx, collectionID, hidden)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) GetUserStats(ctx context.Context, userID uuid.UUID) (*model.MeStats, error) {
	if m.GetUserStatsFn != nil {
		return m.GetUserStatsFn(ctx, userID)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ExportCatalog(ctx context.Context) (*model.CatalogExport, error) {
	if m.ExportCatalogFn != nil {
		return m.ExportCatalogFn(ctx)
	}
	return &model.CatalogExport{Format: model.CatalogFormat, Version: model.ExportVersion}, nil
}

func (m *MockStore) ImportCatalog(ctx context.Context, cat *model.CatalogExport, dryRun bool) (*model.ImportReport, error) {
	if m.ImportCatalogFn != nil {
		return m.ImportCatalogFn(ctx, cat, dryRun)
	}
	return model.NewImportReport(dryRun), nil
}

func (m *MockStore) ImportCollection(ctx context.Context, userID uuid.UUID, exp *model.CollectionExport, dryRun bool) (*model.ImportReport, error) {
	if m.ImportCollectionFn != nil {
		return m.ImportCollectionFn(ctx, userID, exp, dryRun)
	}
	return model.NewCollectionImportReport(dryRun), nil
}

func (m *MockStore) ReferencedUploads(ctx context.Context) ([]string, error) {
	if m.ReferencedUploadsFn != nil {
		return m.ReferencedUploadsFn(ctx)
	}
	return nil, nil
}

func (m *MockStore) ListRelatedItemCollections(ctx context.Context, userID, itemID uuid.UUID) ([]model.RelatedItemCollections, error) {
	if m.ListRelatedItemCollectionsFn != nil {
		return m.ListRelatedItemCollectionsFn(ctx, userID, itemID)
	}
	return []model.RelatedItemCollections{}, nil
}

func (m *MockStore) GetOfflineSnapshot(ctx context.Context, userID uuid.UUID) (*model.OfflineSnapshot, error) {
	if m.GetOfflineSnapshotFn != nil {
		return m.GetOfflineSnapshotFn(ctx, userID)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ListDomains(ctx context.Context) ([]model.DomainInfo, error) {
	if m.ListDomainsFn != nil {
		return m.ListDomainsFn(ctx)
	}
	return []model.DomainInfo{}, nil
}

func (m *MockStore) InstallDomain(ctx context.Context, pkg *model.DomainPackage, dryRun bool) (*model.DomainInstallReport, error) {
	if m.InstallDomainFn != nil {
		return m.InstallDomainFn(ctx, pkg, dryRun)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) RefreshInstalledDomains(ctx context.Context, except string) error {
	if m.RefreshInstalledDomainsFn != nil {
		return m.RefreshInstalledDomainsFn(ctx, except)
	}
	return nil
}

// --- Test helpers ---

var (
	AdminID = uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	UserID  = uuid.MustParse("a0000000-0000-0000-0000-000000000002")
	// ContributorID : rôle contributor (P4-3).
	ContributorID = uuid.MustParse("a0000000-0000-0000-0000-000000000003")
)

func AdminUser() *model.User {
	return &model.User{
		ID: AdminID, Username: "admin", Email: "admin@test.local",
		Role: "admin", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func RegularUser() *model.User {
	return &model.User{
		ID: UserID, Username: "user", Email: "user@test.local",
		Role: "user", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// TestConfig returns the configuration used by NewTestHandler (copy it to tweak a field).
func TestConfig() *config.Config {
	return &config.Config{
		Port:            8080,
		JWTSecret:       TestSecret,
		AccessTokenTTL:  TestAccessTTL,
		RefreshTokenTTL: 24 * time.Hour,
		LoginRateLimit:  1000,
		GraphMaxNodes:   500,
		ProposalQuota:   5,
		UploadDir:       os.TempDir(),
		MaxUploadSize:   10 << 20,
		MaxImportSize:   1 << 20,
	}
}

func NewTestHandler(mockStore *MockStore) *handler.Handler {
	return NewTestHandlerWithConfig(mockStore, TestConfig())
}

func NewTestHandlerWithConfig(mockStore *MockStore, cfg *config.Config) *handler.Handler {
	auth := middleware.NewAuth(cfg.JWTSecret, cfg.AccessTokenTTL)
	return handler.New(mockStore, auth, cfg)
}

func NewTestServer(mockStore *MockStore) *httptest.Server {
	h := NewTestHandler(mockStore)
	return httptest.NewServer(h.Router())
}

func AdminToken() string {
	auth := middleware.NewAuth(TestSecret, TestAccessTTL)
	t, _ := auth.GenerateToken(AdminID, "admin")
	return t
}

// ContributorToken : jeton d'un contributeur (écrit dans le catalogue, modère).
func ContributorToken() string {
	auth := middleware.NewAuth(TestSecret, TestAccessTTL)
	t, _ := auth.GenerateToken(ContributorID, "contributor")
	return t
}

func UserToken() string {
	auth := middleware.NewAuth(TestSecret, TestAccessTTL)
	t, _ := auth.GenerateToken(UserID, "user")
	return t
}

func AuthHeader(token string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+token)
	h.Set("Content-Type", "application/json")
	return h
}

func MergeHeaders(dst, src http.Header) http.Header {
	for k, vs := range src {
		for _, v := range vs {
			dst.Set(k, v)
		}
	}
	return dst
}

func NewTestServerFromHandler(h *handler.Handler) *httptest.Server {
	return httptest.NewServer(h.Router())
}

func SampleItemType() *model.ItemType {
	return &model.ItemType{
		ID:               uuid.MustParse("b0000000-0000-0000-0000-000000000001"),
		Name:             "Serie",
		AttributesSchema: json.RawMessage(`{"type":"object","required":["titre"],"properties":{"titre":{"type":"string"}}}`),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func SampleItem() *model.Item {
	it := SampleItemType()
	return &model.Item{
		ID:         uuid.MustParse("e0000000-0000-0000-0000-000000000001"),
		ItemTypeID: it.ID,
		ItemType:   it,
		Attributes: json.RawMessage(`{"titre":"Breaking Bad"}`),
		Tags:       []model.Tag{{ID: uuid.New(), Name: "drame"}},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// Tests are noisy with the structured request logger: silence slog unless
// TEST_LOG is set.
func init() {
	if os.Getenv("TEST_LOG") == "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
}

// FindItemsByAttribute finds nothing by default (every proposed item is new).
func (m *MockStore) FindItemsByAttribute(ctx context.Context, itemTypeID uuid.UUID, path, values []string, fold bool) ([]model.Item, error) {
	if m.FindItemsByAttributeFn != nil {
		return m.FindItemsByAttributeFn(ctx, itemTypeID, path, values, fold)
	}
	return nil, nil
}

func (m *MockStore) ListRelationsBetween(ctx context.Context, ids []uuid.UUID) ([]model.ItemRelation, error) {
	if m.ListRelationsBetweenFn != nil {
		return m.ListRelationsBetweenFn(ctx, ids)
	}
	return nil, nil
}
