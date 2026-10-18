package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

// callJSON sends a JSON request and returns the status, error code and body.
func callJSON(t *testing.T, srv *httptest.Server, method, path, token string, body any) (int, string, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, srv.URL+path, r)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header = testutil.MergeHeaders(req.Header, testutil.AuthHeader(token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var e model.APIError
	_ = json.Unmarshal(raw, &e)
	return resp.StatusCode, e.Error, raw
}

func serverWith(mock *testutil.MockStore, memberInvites bool) *httptest.Server {
	cfg := testutil.TestConfig()
	cfg.MemberInvites = memberInvites
	return testutil.NewTestServerFromHandler(testutil.NewTestHandlerWithConfig(mock, cfg))
}

func noTokens(mock *testutil.MockStore) {
	mock.CreateRefreshTokenFn = func(context.Context, uuid.UUID, string, time.Time) error { return nil }
}

func TestRegister(t *testing.T) {
	var gotHash string
	mock := &testutil.MockStore{
		RegisterWithInvitationFn: func(_ context.Context, tokenHash, username, email, _ string) (*model.User, error) {
			gotHash = tokenHash
			switch username {
			case "taken":
				return nil, store.ErrConflict
			case "late":
				return nil, store.ErrNotFound
			}
			return &model.User{ID: uuid.New(), Username: username, Email: email, Role: model.RoleUser}, nil
		},
	}
	noTokens(mock)
	srv := serverWith(mock, false)
	defer srv.Close()

	status, _, raw := callJSON(t, srv, "POST", "/api/v1/auth/register", "",
		model.RegisterRequest{Token: "tok", Username: "camille", Email: "camille@example.org", Password: "longenough"})
	var res model.LoginResponse
	_ = json.Unmarshal(raw, &res)
	if status != http.StatusOK || res.Token == "" || res.User.Username != "camille" {
		t.Fatalf("register: %d %s", status, raw)
	}
	if gotHash == "" || gotHash == "tok" {
		t.Errorf("the store must get the hash of the token, got %q", gotHash)
	}
	for _, c := range []struct {
		req    model.RegisterRequest
		status int
		code   string
	}{
		{model.RegisterRequest{Token: "tok", Username: "a b", Email: "x@y.z", Password: "longenough"}, 400, "invalid_username"},
		{model.RegisterRequest{Token: "tok", Username: "camille", Email: "nope", Password: "longenough"}, 400, "invalid_email"},
		{model.RegisterRequest{Token: "tok", Username: "camille", Email: "x@y.z", Password: "short"}, 400, "weak_password"},
		{model.RegisterRequest{Username: "camille", Email: "x@y.z", Password: "longenough"}, 400, "bad_request"},
		{model.RegisterRequest{Token: "tok", Username: "late", Email: "x@y.z", Password: "longenough"}, 401, "invalid_invitation"},
		{model.RegisterRequest{Token: "tok", Username: "taken", Email: "x@y.z", Password: "longenough"}, 409, "account_exists"},
	} {
		if status, code, raw := callJSON(t, srv, "POST", "/api/v1/auth/register", "", c.req); status != c.status || code != c.code {
			t.Errorf("%+v: %d %s", c.req, status, raw)
		}
	}
}

func TestCheckInvitation(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	mock := &testutil.MockStore{CheckInvitationFn: func(_ context.Context, h string) (time.Time, error) {
		if h == "" {
			return time.Time{}, store.ErrNotFound
		}
		return exp, nil
	}}
	srv := serverWith(mock, false)
	defer srv.Close()
	_, _, raw := callJSON(t, srv, "GET", "/api/v1/auth/invitation?token=abc", "", nil)
	var ok model.InvitationCheck
	_ = json.Unmarshal(raw, &ok)
	if !ok.Valid || ok.ExpiresAt == nil {
		t.Errorf("valid invitation: %s", raw)
	}
	_, _, raw = callJSON(t, srv, "GET", "/api/v1/auth/invitation", "", nil)
	if json.Unmarshal(raw, &ok); ok.Valid {
		t.Errorf("no token: %s", raw)
	}
}

func TestDisabledAccount(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret-pass"), bcrypt.MinCost)
	off := time.Now()
	disabled := testutil.RegularUser()
	disabled.DisabledAt = &off
	mock := &testutil.MockStore{
		GetUserByUsernameFn:   func(context.Context, string) (*model.User, string, error) { return disabled, string(hash), nil },
		ConsumeRefreshTokenFn: func(context.Context, string) (uuid.UUID, error) { return disabled.ID, nil },
		GetUserByIDFn:         func(context.Context, uuid.UUID) (*model.User, error) { return disabled, nil },
	}
	srv := serverWith(mock, false)
	defer srv.Close()
	if status, code, _ := callJSON(t, srv, "POST", "/api/v1/auth/login", "", model.LoginRequest{Username: "u", Password: "secret-pass"}); status != 401 || code != "account_disabled" {
		t.Errorf("login: %d %s", status, code)
	}
	if status, code, _ := callJSON(t, srv, "POST", "/api/v1/auth/refresh", "", model.RefreshRequest{RefreshToken: "r"}); status != 401 || code != "account_disabled" {
		t.Errorf("refresh: %d %s", status, code)
	}
}

func TestCanInvite(t *testing.T) {
	mock := &testutil.MockStore{GetUserByIDFn: func(_ context.Context, id uuid.UUID) (*model.User, error) {
		return &model.User{ID: id}, nil
	}}
	for _, c := range []struct {
		token   string
		members bool
		want    bool
	}{
		{testutil.AdminToken(), false, true},
		{testutil.UserToken(), false, false},
		{testutil.UserToken(), true, true},
	} {
		srv := serverWith(mock, c.members)
		_, _, raw := callJSON(t, srv, "GET", "/api/v1/auth/me", c.token, nil)
		srv.Close()
		var u model.User
		_ = json.Unmarshal(raw, &u)
		if u.CanInvite != c.want {
			t.Errorf("members=%v: can_invite %v, want %v", c.members, u.CanInvite, c.want)
		}
	}
}

func TestInvitations(t *testing.T) {
	var stored struct {
		hash, role string
		by         uuid.UUID
		days       float64
	}
	mine := uuid.New()
	mock := &testutil.MockStore{
		CreateInvitationFn: func(_ context.Context, h string, by uuid.UUID, role, _ string, exp time.Time) (*model.Invitation, error) {
			stored.hash, stored.role, stored.by = h, role, by
			stored.days = time.Until(exp).Hours() / 24
			return &model.Invitation{ID: uuid.New(), Role: role, ExpiresAt: exp, Status: model.InvitationPending}, nil
		},
		GetInvitationFn: func(_ context.Context, id uuid.UUID) (*model.Invitation, error) {
			owner := testutil.AdminID
			if id == mine {
				owner = testutil.UserID
			}
			return &model.Invitation{ID: id, CreatedBy: &owner}, nil
		},
		DeleteInvitationFn: func(_ context.Context, id uuid.UUID) error {
			if id == mine {
				return nil
			}
			return store.ErrConflict
		},
		ListInvitationsFn: func(_ context.Context, by *uuid.UUID) ([]model.Invitation, error) {
			if by != nil && *by != testutil.UserID {
				t.Errorf("member listing filtered by %v", by)
			}
			return []model.Invitation{}, nil
		},
	}

	// Members off: reserved to admins.
	srv := serverWith(mock, false)
	if status, _, _ := callJSON(t, srv, "POST", "/api/v1/invitations", testutil.UserToken(), nil); status != 403 {
		t.Errorf("member without MEMBER_INVITES: %d", status)
	}
	status, _, raw := callJSON(t, srv, "POST", "/api/v1/invitations", testutil.AdminToken(), model.InvitationCreate{Role: "admin", ExpiresInDays: 3})
	var created model.InvitationCreated
	_ = json.Unmarshal(raw, &created)
	if status != 201 || created.Token == "" || stored.hash == created.Token || stored.role != "admin" ||
		stored.by != testutil.AdminID || stored.days < 2.9 || stored.days > 3.1 {
		t.Errorf("admin invitation: %d %s %+v", status, raw, stored)
	}
	for _, days := range []int{-1, 91} {
		if status, _, _ := callJSON(t, srv, "POST", "/api/v1/invitations", testutil.AdminToken(), model.InvitationCreate{ExpiresInDays: days}); status != 400 {
			t.Errorf("expires_in_days %d: %d", days, status)
		}
	}
	srv.Close()

	// Members on: users only, their own invitations.
	srv = serverWith(mock, true)
	defer srv.Close()
	if status, _, _ := callJSON(t, srv, "POST", "/api/v1/invitations", testutil.UserToken(), nil); status != 201 || stored.role != "user" || stored.days < 6.9 {
		t.Errorf("member invitation: %d %+v", status, stored)
	}
	if status, _, _ := callJSON(t, srv, "POST", "/api/v1/invitations", testutil.UserToken(), model.InvitationCreate{Role: "admin"}); status != 403 {
		t.Errorf("member inviting an admin: %d", status)
	}
	if status, _, _ := callJSON(t, srv, "GET", "/api/v1/invitations", testutil.UserToken(), nil); status != 200 {
		t.Errorf("member list: %d", status)
	}
	if status, _, _ := callJSON(t, srv, "DELETE", "/api/v1/invitations/"+mine.String(), testutil.UserToken(), nil); status != 204 {
		t.Errorf("own invitation: %d", status)
	}
	if status, _, _ := callJSON(t, srv, "DELETE", "/api/v1/invitations/"+uuid.NewString(), testutil.UserToken(), nil); status != 403 {
		t.Errorf("someone else's invitation: %d", status)
	}
	if status, code, _ := callJSON(t, srv, "DELETE", "/api/v1/invitations/"+uuid.NewString(), testutil.AdminToken(), nil); status != 409 || code != "invitation_used" {
		t.Errorf("used invitation: %d %s", status, code)
	}
}

func TestUsersAdmin(t *testing.T) {
	mock := &testutil.MockStore{
		ListUsersFn: func(context.Context, model.Pagination) ([]model.User, int, error) {
			return []model.User{*testutil.AdminUser(), *testutil.RegularUser()}, 2, nil
		},
		UpdateUserFn: func(_ context.Context, id uuid.UUID, in model.UserUpdate) (*model.User, error) {
			if in.Role != nil && *in.Role == "user" {
				return nil, store.ErrLastAdmin
			}
			u := testutil.RegularUser()
			return u, nil
		},
		CreatePasswordResetFn: func(context.Context, uuid.UUID, string, uuid.UUID, time.Time) error { return nil },
	}
	srv := serverWith(mock, false)
	defer srv.Close()
	if status, _, _ := callJSON(t, srv, "GET", "/api/v1/users", testutil.UserToken(), nil); status != 403 {
		t.Errorf("user listing accounts: %d", status)
	}
	if status, _, raw := callJSON(t, srv, "GET", "/api/v1/users", testutil.AdminToken(), nil); status != 200 || !bytes.Contains(raw, []byte(`"total":2`)) {
		t.Errorf("list: %d %s", status, raw)
	}
	yes, user := true, "user"
	if status, code, _ := callJSON(t, srv, "PUT", "/api/v1/users/"+testutil.AdminID.String(), testutil.AdminToken(), model.UserUpdate{Disabled: &yes}); status != 409 || code != "self_lockout" {
		t.Errorf("self disable: %d %s", status, code)
	}
	if status, code, _ := callJSON(t, srv, "PUT", "/api/v1/users/"+uuid.NewString(), testutil.AdminToken(), model.UserUpdate{Role: &user}); status != 409 || code != "last_admin" {
		t.Errorf("last admin: %d %s", status, code)
	}
	if status, _, _ := callJSON(t, srv, "PUT", "/api/v1/users/"+testutil.UserID.String(), testutil.AdminToken(), model.UserUpdate{Disabled: &yes}); status != 200 {
		t.Errorf("disable: %d", status)
	}
	status, _, raw := callJSON(t, srv, "POST", "/api/v1/users/"+testutil.UserID.String()+"/reset-link", testutil.AdminToken(), nil)
	var link model.ResetLink
	_ = json.Unmarshal(raw, &link)
	if status != 201 || link.Token == "" || time.Until(link.ExpiresAt) < 23*time.Hour {
		t.Errorf("reset link: %d %s", status, raw)
	}
}

func TestProfileAndPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	setCalled := false
	mock := &testutil.MockStore{
		UpdateProfileFn: func(_ context.Context, _ uuid.UUID, in model.ProfileUpdate) (*model.User, error) {
			u := testutil.RegularUser()
			u.DisplayName = &in.DisplayName
			return u, nil
		},
		GetUserPasswordHashFn: func(context.Context, uuid.UUID) (string, error) { return string(hash), nil },
		SetPasswordFn:         func(context.Context, uuid.UUID, string) error { setCalled = true; return nil },
		GetUserByIDFn:         func(context.Context, uuid.UUID) (*model.User, error) { return testutil.RegularUser(), nil },
		ResetPasswordFn: func(_ context.Context, h, _ string) error {
			if h == "" {
				return store.ErrNotFound
			}
			return nil
		},
	}
	noTokens(mock)
	srv := serverWith(mock, false)
	defer srv.Close()
	tok := testutil.UserToken()

	if status, _, _ := callJSON(t, srv, "PUT", "/api/v1/me/profile", tok, model.ProfileUpdate{DisplayName: "Camille", Avatar: "/api/v1/uploads/a.webp"}); status != 200 {
		t.Errorf("profile: %d", status)
	}
	for _, avatar := range []string{"https://evil.example/x.png", "/api/v1/uploads/../x", "/api/v1/uploads/a/b.png"} {
		if status, _, _ := callJSON(t, srv, "PUT", "/api/v1/me/profile", tok, model.ProfileUpdate{Avatar: avatar}); status != 400 {
			t.Errorf("avatar %q: %d", avatar, status)
		}
	}
	if status, code, _ := callJSON(t, srv, "PUT", "/api/v1/me/password", tok, model.PasswordChange{CurrentPassword: "wrong", NewPassword: "new-password"}); status != 403 || code != "wrong_password" || setCalled {
		t.Errorf("wrong current password: %d %s", status, code)
	}
	status, _, raw := callJSON(t, srv, "PUT", "/api/v1/me/password", tok, model.PasswordChange{CurrentPassword: "old-password", NewPassword: "new-password"})
	var res model.LoginResponse
	_ = json.Unmarshal(raw, &res)
	if status != 200 || !setCalled || res.RefreshToken == "" {
		t.Errorf("password change: %d %s", status, raw)
	}
	if status, _, _ := callJSON(t, srv, "POST", "/api/v1/auth/reset-password", "", model.ResetPasswordRequest{Token: "t", Password: "brand-new-pass"}); status != 204 {
		t.Errorf("reset: %d", status)
	}
	if status, code, _ := callJSON(t, srv, "POST", "/api/v1/auth/reset-password", "", model.ResetPasswordRequest{Token: "t", Password: "short"}); status != 400 || code != "weak_password" {
		t.Errorf("reset weak: %d %s", status, code)
	}
}
