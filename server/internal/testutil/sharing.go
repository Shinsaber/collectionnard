package testutil

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

func (m *MockStore) ListVisibleCollections(ctx context.Context, ownerID uuid.UUID, levels []string) ([]model.Collection, error) {
	if m.ListVisibleCollectionsFn != nil {
		return m.ListVisibleCollectionsFn(ctx, ownerID, levels)
	}
	return []model.Collection{}, nil
}

func (m *MockStore) CreateShareLink(ctx context.Context, collectionID uuid.UUID, tokenHash string) (*model.ShareLink, error) {
	if m.CreateShareLinkFn != nil {
		return m.CreateShareLinkFn(ctx, collectionID, tokenHash)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ListShareLinks(ctx context.Context, collectionID uuid.UUID) ([]model.ShareLink, error) {
	if m.ListShareLinksFn != nil {
		return m.ListShareLinksFn(ctx, collectionID)
	}
	return []model.ShareLink{}, nil
}

func (m *MockStore) RevokeShareLink(ctx context.Context, collectionID, linkID uuid.UUID) error {
	if m.RevokeShareLinkFn != nil {
		return m.RevokeShareLinkFn(ctx, collectionID, linkID)
	}
	return store.ErrNotFound
}

func (m *MockStore) GetSharedCollectionID(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	if m.GetSharedCollectionIDFn != nil {
		return m.GetSharedCollectionIDFn(ctx, tokenHash)
	}
	return uuid.Nil, store.ErrNotFound
}

func (m *MockStore) ListMembers(ctx context.Context, pg model.Pagination) ([]model.User, int, error) {
	if m.ListMembersFn != nil {
		return m.ListMembersFn(ctx, pg)
	}
	return []model.User{}, 0, nil
}

func (m *MockStore) GetItemHistory(ctx context.Context, itemID uuid.UUID) (*model.ItemHistory, error) {
	if m.GetItemHistoryFn != nil {
		return m.GetItemHistoryFn(ctx, itemID)
	}
	return &model.ItemHistory{Entries: []model.HistoryEntry{}}, nil
}

func (m *MockStore) GetItemVersion(ctx context.Context, itemID uuid.UUID, version int) (json.RawMessage, error) {
	if m.GetItemVersionFn != nil {
		return m.GetItemVersionFn(ctx, itemID, version)
	}
	return nil, store.ErrNotFound
}
