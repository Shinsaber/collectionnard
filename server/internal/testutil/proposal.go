package testutil

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

// Propositions et modération (P4-3b).

func (m *MockStore) CreateProposal(ctx context.Context, authorID uuid.UUID, in model.ProposalInput) (*model.Proposal, error) {
	if m.CreateProposalFn != nil {
		return m.CreateProposalFn(ctx, authorID, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) GetProposal(ctx context.Context, id uuid.UUID) (*model.Proposal, error) {
	if m.GetProposalFn != nil {
		return m.GetProposalFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ListProposals(ctx context.Context, f model.ProposalFilter, pg model.Pagination) ([]model.Proposal, int, error) {
	if m.ListProposalsFn != nil {
		return m.ListProposalsFn(ctx, f, pg)
	}
	return []model.Proposal{}, 0, nil
}

func (m *MockStore) UpdateProposal(ctx context.Context, id uuid.UUID, in model.ProposalInput) (*model.Proposal, error) {
	if m.UpdateProposalFn != nil {
		return m.UpdateProposalFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) SubmitProposal(ctx context.Context, id uuid.UUID, base *model.ProposalBase) (*model.Proposal, error) {
	if m.SubmitProposalFn != nil {
		return m.SubmitProposalFn(ctx, id, base)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) WithdrawProposal(ctx context.Context, id uuid.UUID) (*model.Proposal, error) {
	if m.WithdrawProposalFn != nil {
		return m.WithdrawProposalFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ReviewProposal(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID, comment string) (*model.Proposal, error) {
	if m.ReviewProposalFn != nil {
		return m.ReviewProposalFn(ctx, id, status, reviewerID, comment)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) AcceptCatalogProposal(ctx context.Context, id, reviewerID uuid.UUID, payload *model.CatalogExport, comment string) (*model.ImportReport, error) {
	if m.AcceptCatalogProposalFn != nil {
		return m.AcceptCatalogProposalFn(ctx, id, reviewerID, payload, comment)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ProposalQuotaUsage(ctx context.Context, authorID uuid.UUID) (int, int, int, error) {
	if m.ProposalQuotaUsageFn != nil {
		return m.ProposalQuotaUsageFn(ctx, authorID)
	}
	return 0, 0, 0, nil
}

func (m *MockStore) CountProposalsToReview(ctx context.Context, reviewerID uuid.UUID) (int, error) {
	if m.CountProposalsToReviewFn != nil {
		return m.CountProposalsToReviewFn(ctx, reviewerID)
	}
	return 0, nil
}

func (m *MockStore) EntitySnapshot(ctx context.Context, itemIDs, relationIDs []uuid.UUID) (*model.ProposalBase, error) {
	if m.EntitySnapshotFn != nil {
		return m.EntitySnapshotFn(ctx, itemIDs, relationIDs)
	}
	return &model.ProposalBase{Items: map[uuid.UUID]json.RawMessage{}, Relations: map[uuid.UUID]json.RawMessage{}}, nil
}

func (m *MockStore) ListProposalComments(ctx context.Context, proposalID uuid.UUID) ([]model.ProposalComment, error) {
	if m.ListProposalCommentsFn != nil {
		return m.ListProposalCommentsFn(ctx, proposalID)
	}
	return []model.ProposalComment{}, nil
}

func (m *MockStore) CreateProposalComment(ctx context.Context, proposalID, authorID uuid.UUID, body string) (*model.ProposalComment, error) {
	if m.CreateProposalCommentFn != nil {
		return m.CreateProposalCommentFn(ctx, proposalID, authorID, body)
	}
	return nil, store.ErrNotFound
}
