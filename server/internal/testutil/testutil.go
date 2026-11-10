// Package testutil provides the handler test kit: MockStore (one XxxFn field per
// Store method, ErrNotFound by default), test servers, tokens and fixtures. No
// PostgreSQL needed.
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
	ListItemRelationsPageFn      func(ctx context.Context, itemID uuid.UUID, rtID *uuid.UUID, pg model.Pagination) ([]model.ItemRelation, int, error)
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
	ListCollectionItemsFn        func(ctx context.Context, colID uuid.UUID, pg model.Pagination, f store.CollectionItemFilters) ([]model.CollectionItem, int, error)
	AddCollectionItemFn          func(ctx context.Context, colID uuid.UUID, in model.CollectionItemAdd) (*model.CollectionItem, error)
	UpdateCollectionItemFn       func(ctx context.Context, colID, itemID uuid.UUID, in model.CollectionItemUpdate) (*model.CollectionItem, error)
	RemoveCollectionItemFn       func(ctx context.Context, colID, itemID uuid.UUID) error
	SplitCollectionLotFn         func(ctx context.Context, colID, lotID uuid.UUID, quantity int, attrs json.RawMessage) (*model.CollectionItem, error)
	MergeCollectionLotsFn        func(ctx context.Context, colID, lotID, into uuid.UUID, galleries []string) (*model.CollectionItem, error)
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
	CreateNotificationsFn        func(ctx context.Context, userIDs []uuid.UUID, kind string, payload any) error
	ListNotificationsFn          func(ctx context.Context, userID uuid.UUID, pg model.Pagination) ([]model.Notification, int, int, error)
	MarkNotificationsReadFn      func(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error
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
	ListItemArticlesFn           func(ctx context.Context, itemID, viewer uuid.UUID, pg model.Pagination) ([]model.Article, int, error)
	GetArticleFn                 func(ctx context.Context, id uuid.UUID) (*model.Article, error)
	CreateArticleFn              func(ctx context.Context, itemID, authorID uuid.UUID, in model.ArticleWrite, status string) (*model.Article, error)
	UpdateArticleFn              func(ctx context.Context, id, authorID uuid.UUID, in model.ArticleWrite, status string) (*model.Article, error)
	DeleteArticleFn              func(ctx context.Context, id, authorID uuid.UUID) error
	CountPendingArticlesFn       func(ctx context.Context) (int, error)
	ListPendingArticlesFn        func(ctx context.Context, pg model.Pagination) ([]model.Article, int, error)
	ReviewArticleFn              func(ctx context.Context, id, moderatorID uuid.UUID, publish bool, reason string) (*model.Article, error)
	SetArticleCommentsClosedFn   func(ctx context.Context, id, userID uuid.UUID, closed bool) error
	CreateRecommendationFn       func(ctx context.Context, senderID, recipientID, itemID uuid.UUID, message string) (*model.Recommendation, error)
	GetRecommendationFn          func(ctx context.Context, id, recipientID uuid.UUID) (*model.Recommendation, error)
	ListRecommendationsFn        func(ctx context.Context, recipientID uuid.UUID, pg model.Pagination) ([]model.Recommendation, int, int, error)
	MarkRecommendationsReadFn    func(ctx context.Context, recipientID uuid.UUID, ids []uuid.UUID) error
	DeleteRecommendationFn       func(ctx context.Context, id, recipientID uuid.UUID) error
	BlockUserFn                  func(ctx context.Context, blockerID, blockedID uuid.UUID, anonymous bool) (*model.Block, error)
	ListBlocksFn                 func(ctx context.Context, blockerID uuid.UUID, pg model.Pagination) ([]model.Block, int, error)
	DeleteBlockFn                func(ctx context.Context, blockerID, id uuid.UUID) error
	// BlockedByFn, BlockersFn, IsBlockedFn: nothing blocked when unset.
	BlockedByFn func(ctx context.Context, blockerID uuid.UUID) ([]uuid.UUID, error)
	BlockersFn  func(ctx context.Context, blockedID uuid.UUID, users []uuid.UUID) ([]uuid.UUID, error)
	IsBlockedFn func(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error)
	// FriendIDsFn: no friend when unset.
	FriendIDsFn           func(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	ListFriendshipsFn     func(ctx context.Context, userID uuid.UUID, status string, pg model.Pagination) ([]model.Friendship, int, error)
	RequestFriendFn       func(ctx context.Context, from, to uuid.UUID) (*model.Friendship, error)
	AcceptFriendRequestFn func(ctx context.Context, userID, id uuid.UUID) (*model.Friendship, error)
	DeleteFriendshipFn    func(ctx context.Context, userID, id uuid.UUID) error
	CreateFriendLinkFn    func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*model.FriendLink, error)
	ListFriendLinksFn     func(ctx context.Context, userID uuid.UUID) ([]model.FriendLink, error)
	DeleteFriendLinkFn    func(ctx context.Context, userID, id uuid.UUID) error
	GetFriendLinkFn       func(ctx context.Context, tokenHash string) (*model.FriendLink, error)
	UseFriendLinkFn       func(ctx context.Context, tokenHash string, userID uuid.UUID) (*model.Friendship, error)
	// Loans (P7-3): no loan when unset.
	GetCollectionLotFn  func(ctx context.Context, collectionID, lotID uuid.UUID) (*model.CollectionItem, error)
	CreateLoanFn        func(ctx context.Context, lotID uuid.UUID, in model.LoanInput) (*model.Loan, error)
	GetLoanFn           func(ctx context.Context, id uuid.UUID) (*model.Loan, error)
	UpdateLoanFn        func(ctx context.Context, id uuid.UUID, in model.LoanInput) (*model.Loan, error)
	DeleteLoanFn        func(ctx context.Context, id uuid.UUID) error
	ListLotLoansFn      func(ctx context.Context, lotID uuid.UUID) ([]model.Loan, error)
	ActiveLoansFn       func(ctx context.Context, lotIDs []uuid.UUID) (map[uuid.UUID][]model.Loan, error)
	ListLoansFn         func(ctx context.Context, ownerID uuid.UUID, status string, pg model.Pagination) ([]model.Loan, int, error)
	ListBorrowingsFn    func(ctx context.Context, borrowerID uuid.UUID, status string, pg model.Pagination) ([]model.Loan, int, error)
	OverdueLoansFn      func(ctx context.Context) ([]model.Loan, error)
	MarkLoansRemindedFn func(ctx context.Context, ids []uuid.UUID) error
	// Gift reservations (P7-4): none when unset.
	CreateReservationFn              func(ctx context.Context, lotID, giverID uuid.UUID, quantity int, expiresAt, note string) (*model.GiftReservation, error)
	GetReservationFn                 func(ctx context.Context, id uuid.UUID) (*model.GiftReservation, error)
	UpdateReservationFn              func(ctx context.Context, id uuid.UUID, in model.ReservationUpdate) (*model.GiftReservation, error)
	DeleteReservationFn              func(ctx context.Context, id uuid.UUID) error
	LotReservationsFn                func(ctx context.Context, lotIDs []uuid.UUID) (map[uuid.UUID][]model.GiftReservation, error)
	ListGiftsFn                      func(ctx context.Context, giverID uuid.UUID, status string, pg model.Pagination) ([]model.GiftReservation, int, error)
	LeftReservationsFn               func(ctx context.Context) ([]model.GiftReservation, error)
	MarkReservationsNotifiedFn       func(ctx context.Context, ids []uuid.UUID) error
	ExpiringReservationsFn           func(ctx context.Context) ([]model.GiftReservation, error)
	MarkReservationsExpiryRemindedFn func(ctx context.Context, ids []uuid.UUID) error
	HandOverReservationFn            func(ctx context.Context, id uuid.UUID) (*model.GiftReservation, error)
	GivenGiftsFn                     func(ctx context.Context, lotIDs []uuid.UUID) (map[uuid.UUID][]model.GiftReservation, error)
	ReceiveGiftFn                    func(ctx context.Context, id uuid.UUID, into *uuid.UUID, attrs json.RawMessage) (*model.GiftReservation, error)
	DeclineGiftFn                    func(ctx context.Context, id uuid.UUID) error
	ListCommentsFn                   func(ctx context.Context, kind string, targetID, viewer uuid.UUID, pg model.Pagination) ([]model.Comment, int, error)
	GetCommentFn                     func(ctx context.Context, id uuid.UUID) (*model.Comment, error)
	CreateCommentFn                  func(ctx context.Context, kind string, targetID, authorID uuid.UUID, body string) (*model.Comment, error)
	UpdateCommentFn                  func(ctx context.Context, id, authorID uuid.UUID, body string) (*model.Comment, error)
	DeleteCommentFn                  func(ctx context.Context, id, authorID uuid.UUID) error
	CommentParticipantsFn            func(ctx context.Context, kind string, targetID uuid.UUID) ([]uuid.UUID, error)
	SetReviewCommentsClosedFn        func(ctx context.Context, reviewID, userID uuid.UUID, closed bool) error
	GetReviewFn                      func(ctx context.Context, id uuid.UUID) (*model.Review, error)
	CreateReportFn                   func(ctx context.Context, reporterID uuid.UUID, in model.ReportCreate) (*model.Report, error)
	CountModerationQueueFn           func(ctx context.Context) (int, error)
	ListModerationQueueFn            func(ctx context.Context, pg model.Pagination) ([]model.ModerationItem, int, error)
	ModerationRecapFn                func(ctx context.Context, userID uuid.UUID) (*model.ModerationRecap, error)
	ModerateFn                       func(ctx context.Context, kind string, targetID, moderatorID uuid.UUID, action, reason string) (*model.ModerationResult, error)
	ListModerationLogFn              func(ctx context.Context, pg model.Pagination) ([]model.ModerationLogEntry, int, error)
	ModerationHistoryFn              func(ctx context.Context, userID uuid.UUID) (*model.ModerationHistory, error)
	SetReactionFn                    func(ctx context.Context, reviewID, userID uuid.UUID, kind string, on bool) error
	ReactionsOfFn                    func(ctx context.Context, reviewIDs []uuid.UUID, viewer uuid.UUID) (map[uuid.UUID]map[string]int, map[uuid.UUID][]string, error)
	ListReviewTypesFn                func(ctx context.Context, pg model.Pagination) ([]model.ReviewType, int, error)
	GetReviewTypeFn                  func(ctx context.Context, id uuid.UUID) (*model.ReviewType, error)
	CreateReviewTypeFn               func(ctx context.Context, in model.ReviewTypeCreate) (*model.ReviewType, error)
	UpdateReviewTypeFn               func(ctx context.Context, id uuid.UUID, in model.ReviewTypeUpdate) (*model.ReviewType, error)
	DeleteReviewTypeFn               func(ctx context.Context, id uuid.UUID) error
	ListReviewContextItemsFn         func(ctx context.Context, reviewTypeID, itemID uuid.UUID) ([]model.Item, error)
	IsReviewContextFn                func(ctx context.Context, reviewTypeID, itemID, contextID uuid.UUID) (bool, error)
	GetItemReviewTypeFn              func(ctx context.Context, itemID uuid.UUID) (*model.ReviewType, error)
	ListItemReviewsFn                func(ctx context.Context, itemID uuid.UUID, p model.ReviewListParams, pg model.Pagination) ([]model.Review, int, error)
	GetReviewStatsFn                 func(ctx context.Context, itemID, viewer uuid.UUID, levels []string) (*model.ReviewStats, error)
	GetMyReviewFn                    func(ctx context.Context, itemID, userID uuid.UUID) (*model.Review, error)
	PutReviewFn                      func(ctx context.Context, itemID, userID uuid.UUID, in model.ReviewWrite) (*model.Review, error)
	DeleteReviewFn                   func(ctx context.Context, itemID, userID uuid.UUID) error
	ListUserReviewsFn                func(ctx context.Context, userID uuid.UUID, levels []string, pg model.Pagination) ([]model.Review, int, error)
}

func (m *MockStore) ListReviewTypes(ctx context.Context, pg model.Pagination) ([]model.ReviewType, int, error) {
	if m.ListReviewTypesFn != nil {
		return m.ListReviewTypesFn(ctx, pg)
	}
	return nil, 0, nil
}

func (m *MockStore) GetReviewType(ctx context.Context, id uuid.UUID) (*model.ReviewType, error) {
	if m.GetReviewTypeFn != nil {
		return m.GetReviewTypeFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) CreateReviewType(ctx context.Context, in model.ReviewTypeCreate) (*model.ReviewType, error) {
	if m.CreateReviewTypeFn != nil {
		return m.CreateReviewTypeFn(ctx, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) UpdateReviewType(ctx context.Context, id uuid.UUID, in model.ReviewTypeUpdate) (*model.ReviewType, error) {
	if m.UpdateReviewTypeFn != nil {
		return m.UpdateReviewTypeFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) DeleteReviewType(ctx context.Context, id uuid.UUID) error {
	if m.DeleteReviewTypeFn != nil {
		return m.DeleteReviewTypeFn(ctx, id)
	}
	return store.ErrNotFound
}

func (m *MockStore) ListReviewContextItems(ctx context.Context, reviewTypeID, itemID uuid.UUID) ([]model.Item, error) {
	if m.ListReviewContextItemsFn != nil {
		return m.ListReviewContextItemsFn(ctx, reviewTypeID, itemID)
	}
	return []model.Item{}, nil
}

func (m *MockStore) IsReviewContext(ctx context.Context, reviewTypeID, itemID, contextID uuid.UUID) (bool, error) {
	if m.IsReviewContextFn != nil {
		return m.IsReviewContextFn(ctx, reviewTypeID, itemID, contextID)
	}
	return false, nil
}

func (m *MockStore) GetItemReviewType(ctx context.Context, itemID uuid.UUID) (*model.ReviewType, error) {
	if m.GetItemReviewTypeFn != nil {
		return m.GetItemReviewTypeFn(ctx, itemID)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ListItemReviews(ctx context.Context, itemID uuid.UUID, p model.ReviewListParams, pg model.Pagination) ([]model.Review, int, error) {
	if m.ListItemReviewsFn != nil {
		return m.ListItemReviewsFn(ctx, itemID, p, pg)
	}
	return nil, 0, nil
}

func (m *MockStore) GetReviewStats(ctx context.Context, itemID, viewer uuid.UUID, levels []string) (*model.ReviewStats, error) {
	if m.GetReviewStatsFn != nil {
		return m.GetReviewStatsFn(ctx, itemID, viewer, levels)
	}
	return &model.ReviewStats{}, nil
}

func (m *MockStore) GetMyReview(ctx context.Context, itemID, userID uuid.UUID) (*model.Review, error) {
	if m.GetMyReviewFn != nil {
		return m.GetMyReviewFn(ctx, itemID, userID)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) PutReview(ctx context.Context, itemID, userID uuid.UUID, in model.ReviewWrite) (*model.Review, error) {
	if m.PutReviewFn != nil {
		return m.PutReviewFn(ctx, itemID, userID, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) DeleteReview(ctx context.Context, itemID, userID uuid.UUID) error {
	if m.DeleteReviewFn != nil {
		return m.DeleteReviewFn(ctx, itemID, userID)
	}
	return store.ErrNotFound
}

func (m *MockStore) ListUserReviews(ctx context.Context, userID uuid.UUID, levels []string, pg model.Pagination) ([]model.Review, int, error) {
	if m.ListUserReviewsFn != nil {
		return m.ListUserReviewsFn(ctx, userID, levels, pg)
	}
	return nil, 0, nil
}

func (m *MockStore) GetReview(ctx context.Context, id uuid.UUID) (*model.Review, error) {
	if m.GetReviewFn != nil {
		return m.GetReviewFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) CreateReport(ctx context.Context, reporterID uuid.UUID, in model.ReportCreate) (*model.Report, error) {
	if m.CreateReportFn != nil {
		return m.CreateReportFn(ctx, reporterID, in)
	}
	return &model.Report{ID: uuid.New(), Status: "open"}, nil
}

func (m *MockStore) CountModerationQueue(ctx context.Context) (int, error) {
	if m.CountModerationQueueFn != nil {
		return m.CountModerationQueueFn(ctx)
	}
	return 0, nil
}

func (m *MockStore) ListModerationQueue(ctx context.Context, pg model.Pagination) ([]model.ModerationItem, int, error) {
	if m.ListModerationQueueFn != nil {
		return m.ListModerationQueueFn(ctx, pg)
	}
	return nil, 0, nil
}

func (m *MockStore) ModerationRecap(ctx context.Context, userID uuid.UUID) (*model.ModerationRecap, error) {
	if m.ModerationRecapFn != nil {
		return m.ModerationRecapFn(ctx, userID)
	}
	return &model.ModerationRecap{}, nil
}

func (m *MockStore) Moderate(ctx context.Context, kind string, targetID, moderatorID uuid.UUID, action, reason string) (*model.ModerationResult, error) {
	if m.ModerateFn != nil {
		return m.ModerateFn(ctx, kind, targetID, moderatorID, action, reason)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ListModerationLog(ctx context.Context, pg model.Pagination) ([]model.ModerationLogEntry, int, error) {
	if m.ListModerationLogFn != nil {
		return m.ListModerationLogFn(ctx, pg)
	}
	return nil, 0, nil
}

func (m *MockStore) ModerationHistory(ctx context.Context, userID uuid.UUID) (*model.ModerationHistory, error) {
	if m.ModerationHistoryFn != nil {
		return m.ModerationHistoryFn(ctx, userID)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) SetReaction(ctx context.Context, reviewID, userID uuid.UUID, kind string, on bool) error {
	if m.SetReactionFn != nil {
		return m.SetReactionFn(ctx, reviewID, userID, kind, on)
	}
	return nil
}

func (m *MockStore) ReactionsOf(ctx context.Context, reviewIDs []uuid.UUID, viewer uuid.UUID) (map[uuid.UUID]map[string]int, map[uuid.UUID][]string, error) {
	if m.ReactionsOfFn != nil {
		return m.ReactionsOfFn(ctx, reviewIDs, viewer)
	}
	return map[uuid.UUID]map[string]int{}, map[uuid.UUID][]string{}, nil
}

func (m *MockStore) ListComments(ctx context.Context, kind string, targetID, viewer uuid.UUID, pg model.Pagination) ([]model.Comment, int, error) {
	if m.ListCommentsFn != nil {
		return m.ListCommentsFn(ctx, kind, targetID, viewer, pg)
	}
	return []model.Comment{}, 0, nil
}

func (m *MockStore) GetComment(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	if m.GetCommentFn != nil {
		return m.GetCommentFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) CreateComment(ctx context.Context, kind string, targetID, authorID uuid.UUID, body string) (*model.Comment, error) {
	if m.CreateCommentFn != nil {
		return m.CreateCommentFn(ctx, kind, targetID, authorID, body)
	}
	return &model.Comment{ID: uuid.New(), AuthorID: authorID, Body: body}, nil
}

func (m *MockStore) UpdateComment(ctx context.Context, id, authorID uuid.UUID, body string) (*model.Comment, error) {
	if m.UpdateCommentFn != nil {
		return m.UpdateCommentFn(ctx, id, authorID, body)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) DeleteComment(ctx context.Context, id, authorID uuid.UUID) error {
	if m.DeleteCommentFn != nil {
		return m.DeleteCommentFn(ctx, id, authorID)
	}
	return store.ErrNotFound
}

func (m *MockStore) CommentParticipants(ctx context.Context, kind string, targetID uuid.UUID) ([]uuid.UUID, error) {
	if m.CommentParticipantsFn != nil {
		return m.CommentParticipantsFn(ctx, kind, targetID)
	}
	return nil, nil
}

func (m *MockStore) SetReviewCommentsClosed(ctx context.Context, reviewID, userID uuid.UUID, closed bool) error {
	if m.SetReviewCommentsClosedFn != nil {
		return m.SetReviewCommentsClosedFn(ctx, reviewID, userID, closed)
	}
	return store.ErrNotFound
}

func (m *MockStore) ListItemArticles(ctx context.Context, itemID, viewer uuid.UUID, pg model.Pagination) ([]model.Article, int, error) {
	if m.ListItemArticlesFn != nil {
		return m.ListItemArticlesFn(ctx, itemID, viewer, pg)
	}
	return []model.Article{}, 0, nil
}

func (m *MockStore) GetArticle(ctx context.Context, id uuid.UUID) (*model.Article, error) {
	if m.GetArticleFn != nil {
		return m.GetArticleFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) CreateArticle(ctx context.Context, itemID, authorID uuid.UUID, in model.ArticleWrite, status string) (*model.Article, error) {
	if m.CreateArticleFn != nil {
		return m.CreateArticleFn(ctx, itemID, authorID, in, status)
	}
	return &model.Article{ID: uuid.New(), ItemID: itemID, AuthorID: authorID, Title: in.Title, Status: status}, nil
}

func (m *MockStore) UpdateArticle(ctx context.Context, id, authorID uuid.UUID, in model.ArticleWrite, status string) (*model.Article, error) {
	if m.UpdateArticleFn != nil {
		return m.UpdateArticleFn(ctx, id, authorID, in, status)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) DeleteArticle(ctx context.Context, id, authorID uuid.UUID) error {
	if m.DeleteArticleFn != nil {
		return m.DeleteArticleFn(ctx, id, authorID)
	}
	return store.ErrNotFound
}

func (m *MockStore) CountPendingArticles(ctx context.Context) (int, error) {
	if m.CountPendingArticlesFn != nil {
		return m.CountPendingArticlesFn(ctx)
	}
	return 0, nil
}

func (m *MockStore) ListPendingArticles(ctx context.Context, pg model.Pagination) ([]model.Article, int, error) {
	if m.ListPendingArticlesFn != nil {
		return m.ListPendingArticlesFn(ctx, pg)
	}
	return []model.Article{}, 0, nil
}

func (m *MockStore) ReviewArticle(ctx context.Context, id, moderatorID uuid.UUID, publish bool, reason string) (*model.Article, error) {
	if m.ReviewArticleFn != nil {
		return m.ReviewArticleFn(ctx, id, moderatorID, publish, reason)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) SetArticleCommentsClosed(ctx context.Context, id, userID uuid.UUID, closed bool) error {
	if m.SetArticleCommentsClosedFn != nil {
		return m.SetArticleCommentsClosedFn(ctx, id, userID, closed)
	}
	return store.ErrNotFound
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
func (m *MockStore) ListCollectionItems(ctx context.Context, colID uuid.UUID, pg model.Pagination, f store.CollectionItemFilters) ([]model.CollectionItem, int, error) {
	if m.ListCollectionItemsFn != nil {
		return m.ListCollectionItemsFn(ctx, colID, pg, f)
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
func (m *MockStore) SplitCollectionLot(ctx context.Context, colID, lotID uuid.UUID, quantity int, attrs json.RawMessage) (*model.CollectionItem, error) {
	if m.SplitCollectionLotFn != nil {
		return m.SplitCollectionLotFn(ctx, colID, lotID, quantity, attrs)
	}
	return nil, store.ErrNotFound
}
func (m *MockStore) MergeCollectionLots(ctx context.Context, colID, lotID, into uuid.UUID, galleries []string) (*model.CollectionItem, error) {
	if m.MergeCollectionLotsFn != nil {
		return m.MergeCollectionLotsFn(ctx, colID, lotID, into, galleries)
	}
	return nil, store.ErrNotFound
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
	// ModeratorID : rôle moderator (I-1).
	ModeratorID = uuid.MustParse("a0000000-0000-0000-0000-000000000004")
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
		UploadRateLimit: 1000,
		UploadRateBytes: 1 << 30,
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

// ModeratorToken : jeton d'un modérateur (signalements, I-1).
func ModeratorToken() string {
	auth := middleware.NewAuth(TestSecret, TestAccessTTL)
	t, _ := auth.GenerateToken(ModeratorID, "moderator")
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

// ListItemRelationsPage returns the page given by ListItemRelationsPageFn, or nothing (empty, 0).
func (m *MockStore) ListItemRelationsPage(ctx context.Context, itemID uuid.UUID, rtID *uuid.UUID, pg model.Pagination) ([]model.ItemRelation, int, error) {
	if m.ListItemRelationsPageFn != nil {
		return m.ListItemRelationsPageFn(ctx, itemID, rtID, pg)
	}
	return []model.ItemRelation{}, 0, nil
}

// --- Recommendations and blocks (I-4) ---

func (m *MockStore) CreateRecommendation(ctx context.Context, senderID, recipientID, itemID uuid.UUID, message string) (*model.Recommendation, error) {
	if m.CreateRecommendationFn != nil {
		return m.CreateRecommendationFn(ctx, senderID, recipientID, itemID, message)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) GetRecommendation(ctx context.Context, id, recipientID uuid.UUID) (*model.Recommendation, error) {
	if m.GetRecommendationFn != nil {
		return m.GetRecommendationFn(ctx, id, recipientID)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ListRecommendations(ctx context.Context, recipientID uuid.UUID, pg model.Pagination) ([]model.Recommendation, int, int, error) {
	if m.ListRecommendationsFn != nil {
		return m.ListRecommendationsFn(ctx, recipientID, pg)
	}
	return nil, 0, 0, store.ErrNotFound
}

func (m *MockStore) MarkRecommendationsRead(ctx context.Context, recipientID uuid.UUID, ids []uuid.UUID) error {
	if m.MarkRecommendationsReadFn != nil {
		return m.MarkRecommendationsReadFn(ctx, recipientID, ids)
	}
	return store.ErrNotFound
}

func (m *MockStore) DeleteRecommendation(ctx context.Context, id, recipientID uuid.UUID) error {
	if m.DeleteRecommendationFn != nil {
		return m.DeleteRecommendationFn(ctx, id, recipientID)
	}
	return store.ErrNotFound
}

func (m *MockStore) BlockUser(ctx context.Context, blockerID, blockedID uuid.UUID, anonymous bool) (*model.Block, error) {
	if m.BlockUserFn != nil {
		return m.BlockUserFn(ctx, blockerID, blockedID, anonymous)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ListBlocks(ctx context.Context, blockerID uuid.UUID, pg model.Pagination) ([]model.Block, int, error) {
	if m.ListBlocksFn != nil {
		return m.ListBlocksFn(ctx, blockerID, pg)
	}
	return nil, 0, store.ErrNotFound
}

func (m *MockStore) DeleteBlock(ctx context.Context, blockerID, id uuid.UUID) error {
	if m.DeleteBlockFn != nil {
		return m.DeleteBlockFn(ctx, blockerID, id)
	}
	return store.ErrNotFound
}

func (m *MockStore) BlockedBy(ctx context.Context, blockerID uuid.UUID) ([]uuid.UUID, error) {
	if m.BlockedByFn != nil {
		return m.BlockedByFn(ctx, blockerID)
	}
	return nil, nil
}

func (m *MockStore) Blockers(ctx context.Context, blockedID uuid.UUID, users []uuid.UUID) ([]uuid.UUID, error) {
	if m.BlockersFn != nil {
		return m.BlockersFn(ctx, blockedID, users)
	}
	return nil, nil
}

func (m *MockStore) IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	if m.IsBlockedFn != nil {
		return m.IsBlockedFn(ctx, blockerID, blockedID)
	}
	return false, nil
}

func (m *MockStore) FriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if m.FriendIDsFn != nil {
		return m.FriendIDsFn(ctx, userID)
	}
	return nil, nil
}

func (m *MockStore) ListFriendships(ctx context.Context, userID uuid.UUID, status string, pg model.Pagination) ([]model.Friendship, int, error) {
	if m.ListFriendshipsFn != nil {
		return m.ListFriendshipsFn(ctx, userID, status, pg)
	}
	return nil, 0, store.ErrNotFound
}

func (m *MockStore) RequestFriend(ctx context.Context, from, to uuid.UUID) (*model.Friendship, error) {
	if m.RequestFriendFn != nil {
		return m.RequestFriendFn(ctx, from, to)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) AcceptFriendRequest(ctx context.Context, userID, id uuid.UUID) (*model.Friendship, error) {
	if m.AcceptFriendRequestFn != nil {
		return m.AcceptFriendRequestFn(ctx, userID, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) DeleteFriendship(ctx context.Context, userID, id uuid.UUID) error {
	if m.DeleteFriendshipFn != nil {
		return m.DeleteFriendshipFn(ctx, userID, id)
	}
	return store.ErrNotFound
}

func (m *MockStore) CreateFriendLink(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*model.FriendLink, error) {
	if m.CreateFriendLinkFn != nil {
		return m.CreateFriendLinkFn(ctx, userID, tokenHash, expiresAt)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ListFriendLinks(ctx context.Context, userID uuid.UUID) ([]model.FriendLink, error) {
	if m.ListFriendLinksFn != nil {
		return m.ListFriendLinksFn(ctx, userID)
	}
	return []model.FriendLink{}, nil
}

func (m *MockStore) DeleteFriendLink(ctx context.Context, userID, id uuid.UUID) error {
	if m.DeleteFriendLinkFn != nil {
		return m.DeleteFriendLinkFn(ctx, userID, id)
	}
	return store.ErrNotFound
}

func (m *MockStore) GetFriendLink(ctx context.Context, tokenHash string) (*model.FriendLink, error) {
	if m.GetFriendLinkFn != nil {
		return m.GetFriendLinkFn(ctx, tokenHash)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) UseFriendLink(ctx context.Context, tokenHash string, userID uuid.UUID) (*model.Friendship, error) {
	if m.UseFriendLinkFn != nil {
		return m.UseFriendLinkFn(ctx, tokenHash, userID)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) CreateLoan(ctx context.Context, lotID uuid.UUID, in model.LoanInput) (*model.Loan, error) {
	if m.CreateLoanFn != nil {
		return m.CreateLoanFn(ctx, lotID, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) GetLoan(ctx context.Context, id uuid.UUID) (*model.Loan, error) {
	if m.GetLoanFn != nil {
		return m.GetLoanFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) UpdateLoan(ctx context.Context, id uuid.UUID, in model.LoanInput) (*model.Loan, error) {
	if m.UpdateLoanFn != nil {
		return m.UpdateLoanFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) DeleteLoan(ctx context.Context, id uuid.UUID) error {
	if m.DeleteLoanFn != nil {
		return m.DeleteLoanFn(ctx, id)
	}
	return store.ErrNotFound
}

func (m *MockStore) ListLotLoans(ctx context.Context, lotID uuid.UUID) ([]model.Loan, error) {
	if m.ListLotLoansFn != nil {
		return m.ListLotLoansFn(ctx, lotID)
	}
	return []model.Loan{}, nil
}

func (m *MockStore) ActiveLoans(ctx context.Context, lotIDs []uuid.UUID) (map[uuid.UUID][]model.Loan, error) {
	if m.ActiveLoansFn != nil {
		return m.ActiveLoansFn(ctx, lotIDs)
	}
	return map[uuid.UUID][]model.Loan{}, nil
}

func (m *MockStore) ListLoans(ctx context.Context, ownerID uuid.UUID, status string, pg model.Pagination) ([]model.Loan, int, error) {
	if m.ListLoansFn != nil {
		return m.ListLoansFn(ctx, ownerID, status, pg)
	}
	return []model.Loan{}, 0, nil
}

func (m *MockStore) ListBorrowings(ctx context.Context, borrowerID uuid.UUID, status string, pg model.Pagination) ([]model.Loan, int, error) {
	if m.ListBorrowingsFn != nil {
		return m.ListBorrowingsFn(ctx, borrowerID, status, pg)
	}
	return []model.Loan{}, 0, nil
}

func (m *MockStore) OverdueLoans(ctx context.Context) ([]model.Loan, error) {
	if m.OverdueLoansFn != nil {
		return m.OverdueLoansFn(ctx)
	}
	return nil, nil
}

func (m *MockStore) MarkLoansReminded(ctx context.Context, ids []uuid.UUID) error {
	if m.MarkLoansRemindedFn != nil {
		return m.MarkLoansRemindedFn(ctx, ids)
	}
	return nil
}

func (m *MockStore) GetCollectionLot(ctx context.Context, collectionID, lotID uuid.UUID) (*model.CollectionItem, error) {
	if m.GetCollectionLotFn != nil {
		return m.GetCollectionLotFn(ctx, collectionID, lotID)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) CreateReservation(ctx context.Context, lotID, giverID uuid.UUID, quantity int, expiresAt, note string) (*model.GiftReservation, error) {
	if m.CreateReservationFn != nil {
		return m.CreateReservationFn(ctx, lotID, giverID, quantity, expiresAt, note)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) GetReservation(ctx context.Context, id uuid.UUID) (*model.GiftReservation, error) {
	if m.GetReservationFn != nil {
		return m.GetReservationFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) UpdateReservation(ctx context.Context, id uuid.UUID, in model.ReservationUpdate) (*model.GiftReservation, error) {
	if m.UpdateReservationFn != nil {
		return m.UpdateReservationFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) DeleteReservation(ctx context.Context, id uuid.UUID) error {
	if m.DeleteReservationFn != nil {
		return m.DeleteReservationFn(ctx, id)
	}
	return store.ErrNotFound
}

func (m *MockStore) LotReservations(ctx context.Context, lotIDs []uuid.UUID) (map[uuid.UUID][]model.GiftReservation, error) {
	if m.LotReservationsFn != nil {
		return m.LotReservationsFn(ctx, lotIDs)
	}
	return map[uuid.UUID][]model.GiftReservation{}, nil
}

func (m *MockStore) ListGifts(ctx context.Context, giverID uuid.UUID, status string, pg model.Pagination) ([]model.GiftReservation, int, error) {
	if m.ListGiftsFn != nil {
		return m.ListGiftsFn(ctx, giverID, status, pg)
	}
	return []model.GiftReservation{}, 0, nil
}

func (m *MockStore) LeftReservations(ctx context.Context) ([]model.GiftReservation, error) {
	if m.LeftReservationsFn != nil {
		return m.LeftReservationsFn(ctx)
	}
	return nil, nil
}

func (m *MockStore) MarkReservationsNotified(ctx context.Context, ids []uuid.UUID) error {
	if m.MarkReservationsNotifiedFn != nil {
		return m.MarkReservationsNotifiedFn(ctx, ids)
	}
	return nil
}

func (m *MockStore) ExpiringReservations(ctx context.Context) ([]model.GiftReservation, error) {
	if m.ExpiringReservationsFn != nil {
		return m.ExpiringReservationsFn(ctx)
	}
	return nil, nil
}

func (m *MockStore) MarkReservationsExpiryReminded(ctx context.Context, ids []uuid.UUID) error {
	if m.MarkReservationsExpiryRemindedFn != nil {
		return m.MarkReservationsExpiryRemindedFn(ctx, ids)
	}
	return nil
}

func (m *MockStore) HandOverReservation(ctx context.Context, id uuid.UUID) (*model.GiftReservation, error) {
	if m.HandOverReservationFn != nil {
		return m.HandOverReservationFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) GivenGifts(ctx context.Context, lotIDs []uuid.UUID) (map[uuid.UUID][]model.GiftReservation, error) {
	if m.GivenGiftsFn != nil {
		return m.GivenGiftsFn(ctx, lotIDs)
	}
	return map[uuid.UUID][]model.GiftReservation{}, nil
}

func (m *MockStore) ReceiveGift(ctx context.Context, id uuid.UUID, into *uuid.UUID, attrs json.RawMessage) (*model.GiftReservation, error) {
	if m.ReceiveGiftFn != nil {
		return m.ReceiveGiftFn(ctx, id, into, attrs)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) DeclineGift(ctx context.Context, id uuid.UUID) error {
	if m.DeclineGiftFn != nil {
		return m.DeclineGiftFn(ctx, id)
	}
	return store.ErrNotFound
}
