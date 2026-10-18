package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

// MockStore methods of the accounts (P4-0); an unset XxxFn answers store.ErrNotFound.

func (m *MockStore) GetUserPasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	if m.GetUserPasswordHashFn != nil {
		return m.GetUserPasswordHashFn(ctx, id)
	}
	return "", store.ErrNotFound
}

func (m *MockStore) ListUsers(ctx context.Context, pg model.Pagination) ([]model.User, int, error) {
	if m.ListUsersFn != nil {
		return m.ListUsersFn(ctx, pg)
	}
	return nil, 0, store.ErrNotFound
}

func (m *MockStore) UpdateUser(ctx context.Context, id uuid.UUID, in model.UserUpdate) (*model.User, error) {
	if m.UpdateUserFn != nil {
		return m.UpdateUserFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) UpdateProfile(ctx context.Context, id uuid.UUID, in model.ProfileUpdate) (*model.User, error) {
	if m.UpdateProfileFn != nil {
		return m.UpdateProfileFn(ctx, id, in)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	if m.SetPasswordFn != nil {
		return m.SetPasswordFn(ctx, id, passwordHash)
	}
	return store.ErrNotFound
}

func (m *MockStore) CreateInvitation(ctx context.Context, tokenHash string, createdBy uuid.UUID, role, note string, expiresAt time.Time) (*model.Invitation, error) {
	if m.CreateInvitationFn != nil {
		return m.CreateInvitationFn(ctx, tokenHash, createdBy, role, note, expiresAt)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) GetInvitation(ctx context.Context, id uuid.UUID) (*model.Invitation, error) {
	if m.GetInvitationFn != nil {
		return m.GetInvitationFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) ListInvitations(ctx context.Context, createdBy *uuid.UUID) ([]model.Invitation, error) {
	if m.ListInvitationsFn != nil {
		return m.ListInvitationsFn(ctx, createdBy)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	if m.DeleteInvitationFn != nil {
		return m.DeleteInvitationFn(ctx, id)
	}
	return store.ErrNotFound
}

func (m *MockStore) CheckInvitation(ctx context.Context, tokenHash string) (time.Time, error) {
	if m.CheckInvitationFn != nil {
		return m.CheckInvitationFn(ctx, tokenHash)
	}
	return time.Time{}, store.ErrNotFound
}

func (m *MockStore) RegisterWithInvitation(ctx context.Context, tokenHash, username, email, passwordHash string) (*model.User, error) {
	if m.RegisterWithInvitationFn != nil {
		return m.RegisterWithInvitationFn(ctx, tokenHash, username, email, passwordHash)
	}
	return nil, store.ErrNotFound
}

func (m *MockStore) CreatePasswordReset(ctx context.Context, userID uuid.UUID, tokenHash string, createdBy uuid.UUID, expiresAt time.Time) error {
	if m.CreatePasswordResetFn != nil {
		return m.CreatePasswordResetFn(ctx, userID, tokenHash, createdBy, expiresAt)
	}
	return store.ErrNotFound
}

func (m *MockStore) ResetPassword(ctx context.Context, tokenHash, passwordHash string) error {
	if m.ResetPasswordFn != nil {
		return m.ResetPasswordFn(ctx, tokenHash, passwordHash)
	}
	return store.ErrNotFound
}
