//go:build integration

package store_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

// Invitation: usable once, its role given to the account; withdrawn only before use.
func TestInvitationLifecycle(t *testing.T) {
	st := freshStore(t, "colec_test_invitations")
	admin, err := st.CreateUser(ctx, "root", "root@example.org", "x", model.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	inv, err := st.CreateInvitation(ctx, "hash-1", admin.ID, model.RoleUser, "pour Camille", time.Now().Add(time.Hour))
	if err != nil || inv.Status != model.InvitationPending || inv.CreatedByUsername != "root" || inv.Note == nil {
		t.Fatalf("create: %v %+v", err, inv)
	}
	if _, err := st.CheckInvitation(ctx, "hash-1"); err != nil {
		t.Errorf("check: %v", err)
	}
	u, err := st.RegisterWithInvitation(ctx, "hash-1", "camille", "camille@example.org", "pw")
	if err != nil || u.Role != model.RoleUser {
		t.Fatalf("register: %v %+v", err, u)
	}
	if _, err := st.RegisterWithInvitation(ctx, "hash-1", "other", "other@example.org", "pw"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("second use: %v", err)
	}
	if _, err := st.CheckInvitation(ctx, "hash-1"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("check after use: %v", err)
	}
	got, _ := st.GetInvitation(ctx, inv.ID)
	if got.Status != model.InvitationUsed || got.UsedByUsername != "camille" {
		t.Errorf("used: %+v", got)
	}
	if err := st.DeleteInvitation(ctx, inv.ID); !errors.Is(err, store.ErrConflict) {
		t.Errorf("delete used: %v", err)
	}

	// Username taken (any case): the invitation stays usable.
	if _, err := st.CreateInvitation(ctx, "hash-2", admin.ID, model.RoleAdmin, "", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RegisterWithInvitation(ctx, "hash-2", "CAMILLE", "c2@example.org", "pw"); !errors.Is(err, store.ErrConflict) {
		t.Errorf("taken username: %v", err)
	}
	if u, err := st.RegisterWithInvitation(ctx, "hash-2", "dom", "dom@example.org", "pw"); err != nil || u.Role != model.RoleAdmin {
		t.Errorf("admin invitation after a conflict: %v %+v", err, u)
	}

	// Expired; withdrawn.
	exp, _ := st.CreateInvitation(ctx, "hash-3", admin.ID, model.RoleUser, "", time.Now().Add(-time.Minute))
	if exp.Status != model.InvitationExpired {
		t.Errorf("expired status: %s", exp.Status)
	}
	if _, err := st.RegisterWithInvitation(ctx, "hash-3", "late", "late@example.org", "pw"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expired: %v", err)
	}
	if err := st.DeleteInvitation(ctx, exp.ID); err != nil {
		t.Errorf("withdraw: %v", err)
	}
	mine, _ := st.ListInvitations(ctx, &admin.ID)
	all, _ := st.ListInvitations(ctx, nil)
	if len(mine) != 2 || len(all) != 2 {
		t.Errorf("lists: %d mine, %d all", len(mine), len(all))
	}
}

// An instance keeps an active admin; disabling revokes the sessions.
func TestUpdateUser_LastAdminAndDisable(t *testing.T) {
	st := freshStore(t, "colec_test_users")
	a1, _ := st.CreateUser(ctx, "a1", "a1@example.org", "x", model.RoleAdmin)
	a2, _ := st.CreateUser(ctx, "a2", "a2@example.org", "x", model.RoleAdmin)
	user, _ := st.CreateUser(ctx, "u", "u@example.org", "x", model.RoleUser)
	yes, no, roleUser := true, false, model.RoleUser

	if _, err := st.UpdateUser(ctx, a2.ID, model.UserUpdate{Role: &roleUser}); err != nil {
		t.Fatalf("demote a2 (a1 remains): %v", err)
	}
	if _, err := st.UpdateUser(ctx, a1.ID, model.UserUpdate{Disabled: &yes}); !errors.Is(err, store.ErrLastAdmin) {
		t.Errorf("disable the last admin: %v", err)
	}
	if _, err := st.UpdateUser(ctx, a1.ID, model.UserUpdate{Role: &roleUser}); !errors.Is(err, store.ErrLastAdmin) {
		t.Errorf("demote the last admin: %v", err)
	}

	if err := st.CreateRefreshToken(ctx, user.ID, "rt", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := st.UpdateUser(ctx, user.ID, model.UserUpdate{Disabled: &yes})
	if err != nil || got.DisabledAt == nil {
		t.Fatalf("disable: %v %+v", err, got)
	}
	if _, err := st.ConsumeRefreshToken(ctx, "rt"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("refresh token kept after disabling: %v", err)
	}
	if got, _ := st.UpdateUser(ctx, user.ID, model.UserUpdate{Disabled: &no}); got.DisabledAt != nil {
		t.Errorf("enable: %+v", got)
	}
	list, total, _ := st.ListUsers(ctx, model.Pagination{Page: 1, PerPage: 50})
	if total != 3 || list[0].Username != "a1" {
		t.Errorf("list: %d %+v", total, list)
	}
}

// Reset link: once, revokes the sessions, cancelled by a newer link, refused for a disabled account.
func TestPasswordReset(t *testing.T) {
	st := freshStore(t, "colec_test_resets")
	admin, _ := st.CreateUser(ctx, "root", "root@example.org", "x", model.RoleAdmin)
	user, _ := st.CreateUser(ctx, "u", "u@example.org", "old", model.RoleUser)
	_ = st.CreateRefreshToken(ctx, user.ID, "rt", time.Now().Add(time.Hour))

	if err := st.CreatePasswordReset(ctx, user.ID, "r1", admin.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := st.CreatePasswordReset(ctx, user.ID, "r2", admin.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := st.ResetPassword(ctx, "r1", "new"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("link cancelled by a newer one: %v", err)
	}
	if err := st.ResetPassword(ctx, "r2", "new"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if h, _ := st.GetUserPasswordHash(ctx, user.ID); h != "new" {
		t.Errorf("password: %q", h)
	}
	if _, err := st.ConsumeRefreshToken(ctx, "rt"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("sessions kept after reset: %v", err)
	}
	if err := st.ResetPassword(ctx, "r2", "again"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("second use: %v", err)
	}
	yes := true
	_ = st.CreatePasswordReset(ctx, user.ID, "r3", admin.ID, time.Now().Add(time.Hour))
	_, _ = st.UpdateUser(ctx, user.ID, model.UserUpdate{Disabled: &yes})
	if err := st.ResetPassword(ctx, "r3", "x"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("disabled account: %v", err)
	}
	if err := st.CreatePasswordReset(ctx, admin.ID, "r4", admin.ID, time.Now().Add(time.Hour)); err != nil {
		t.Errorf("admin link: %v", err)
	}
}

// The profile is replaced; the avatar is kept by the purge of unused images.
func TestProfileAndAvatarKeptByPurge(t *testing.T) {
	st := freshStore(t, "colec_test_profiles")
	u, _ := st.CreateUser(ctx, "u", "u@example.org", "x", model.RoleUser)
	got, err := st.UpdateProfile(ctx, u.ID, model.ProfileUpdate{DisplayName: "Camille", Bio: "Mangas", Avatar: "/api/v1/uploads/abc.webp"})
	if err != nil || got.DisplayName == nil || *got.DisplayName != "Camille" || got.Avatar == nil {
		t.Fatalf("profile: %v %+v", err, got)
	}
	refs, err := st.ReferencedUploads(ctx)
	if err != nil || !slices.Contains(refs, "abc.webp") {
		t.Errorf("avatar not referenced: %v %v", err, refs)
	}
	got, _ = st.UpdateProfile(ctx, u.ID, model.ProfileUpdate{})
	if got.DisplayName != nil || got.Bio != nil || got.Avatar != nil {
		t.Errorf("cleared profile: %+v", got)
	}
}
