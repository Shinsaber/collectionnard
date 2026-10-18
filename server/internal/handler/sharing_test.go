package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

// secret is the value of a hidden field: it must never appear in a public response.
const secret = "4242.42"

// privateSchema marks purchase_price `x-private` (default of the type).
const privateSchema = `{"type":"object","x-order":["condition","purchase_price"],
 "x-labels":{"fr":{"purchase_price":"Prix d'achat"}},
 "properties":{"condition":{"type":"string"},"purchase_price":{"type":"number","x-private":true}}}`

type sharingFixture struct {
	col         *model.Collection
	owner       *model.User
	statsHidden []string
	shareHash   string
}

// newSharing: a collection of UserID with one entry carrying the secret price.
func newSharing(visibility string) (*sharingFixture, *testutil.MockStore) {
	f := &sharingFixture{
		col:   sampleCollection(testutil.UserID),
		owner: testutil.RegularUser(),
	}
	f.col.Visibility = visibility
	f.col.CollectionType.AttributesSchema = json.RawMessage(privateSchema)
	f.owner.ProfileVisibility = visibility
	mock := &testutil.MockStore{
		GetCollectionFn: func(_ context.Context, id uuid.UUID) (*model.Collection, error) {
			if id == f.col.ID {
				c := *f.col
				return &c, nil
			}
			return nil, store.ErrNotFound
		},
		GetUserByIDFn: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			if id == f.owner.ID {
				return f.owner, nil
			}
			return nil, store.ErrNotFound
		},
		GetUserByUsernameFn: func(_ context.Context, username string) (*model.User, string, error) {
			if username == f.owner.Username {
				return f.owner, "hash", nil
			}
			return nil, "", store.ErrNotFound
		},
		ListCollectionItemsFn: func(_ context.Context, colID uuid.UUID, _ model.Pagination, _ *uuid.UUID, _ string) ([]model.CollectionItem, int, error) {
			return []model.CollectionItem{{
				ID: uuid.New(), CollectionID: colID, ItemID: uuid.New(),
				Attributes: json.RawMessage(`{"condition":"new","purchase_price":` + secret + `}`),
				CreatedAt:  time.Now(), UpdatedAt: time.Now(),
			}}, 1, nil
		},
		GetCollectionStatsFn: func(_ context.Context, id uuid.UUID, hidden []string) (*model.CollectionStats, error) {
			f.statsHidden = hidden
			return &model.CollectionStats{CollectionID: id}, nil
		},
		GetSharedCollectionIDFn: func(_ context.Context, hash string) (uuid.UUID, error) {
			if hash != "" && hash == f.shareHash {
				return f.col.ID, nil
			}
			return uuid.Nil, store.ErrNotFound
		},
		ListVisibleCollectionsFn: func(_ context.Context, _ uuid.UUID, levels []string) ([]model.Collection, error) {
			if slices.Contains(levels, f.col.Visibility) {
				return []model.Collection{*f.col}, nil
			}
			return []model.Collection{}, nil
		},
	}
	return f, mock
}

// Strong filtering: whatever the reader (anonymous, member, owner, admin), no
// public route leaks the hidden value or its label; stats are told to skip it.
func TestPublicRoutes_StripHiddenFields(t *testing.T) {
	f, mock := newSharing("public")
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	base := "/api/v1/public/collections/" + f.col.ID.String()

	for name, token := range map[string]string{
		"anonymous": "", "owner": testutil.UserToken(), "admin": testutil.AdminToken(),
	} {
		for _, path := range []string{base, base + "/items", base + "/stats", "/api/v1/public/users/user/collections"} {
			f.statsHidden = nil
			status, _, raw := callJSON(t, srv, http.MethodGet, path, token, nil)
			if status != http.StatusOK {
				t.Fatalf("%s %s: %d %s", name, path, status, raw)
			}
			if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "purchase_price") || strings.Contains(string(raw), "Prix d") {
				t.Errorf("%s %s leaks the hidden field: %s", name, path, raw)
			}
			if strings.HasSuffix(path, "/stats") && !slices.Equal(f.statsHidden, []string{"purchase_price"}) {
				t.Errorf("%s stats computed with hidden=%v", name, f.statsHidden)
			}
		}
	}

	// The owner's override replaces the default: price shown, condition hidden.
	f.col.HiddenFields = []string{"condition"}
	_, _, raw := callJSON(t, srv, http.MethodGet, base+"/items", "", nil)
	if !strings.Contains(string(raw), secret) || strings.Contains(string(raw), `"condition"`) {
		t.Errorf("override not applied: %s", raw)
	}
}

func TestPublicCollection_Visibility(t *testing.T) {
	cases := []struct {
		visibility string
		token      string
		want       int
	}{
		{"public", "", http.StatusOK},
		{"members", "", http.StatusNotFound},
		{"members", testutil.AdminToken(), http.StatusOK},
		{"private", testutil.AdminToken(), http.StatusNotFound}, // no admin exception
		{"private", testutil.UserToken(), http.StatusOK},        // owner preview
		{"public", "not-a-jwt", http.StatusUnauthorized},        // invalid token: refresh, not anonymous
	}
	for _, c := range cases {
		f, mock := newSharing(c.visibility)
		srv := testutil.NewTestServer(mock)
		for _, path := range []string{"/api/v1/public/collections/" + f.col.ID.String(), "/api/v1/public/users/user"} {
			if status, _, raw := callJSON(t, srv, http.MethodGet, path, c.token, nil); status != c.want {
				t.Errorf("%s %s with %q: %d, want %d (%s)", c.visibility, path, c.token, status, c.want, raw)
			}
		}
		srv.Close()
	}
}

func TestPublicCollection_DisabledOwner(t *testing.T) {
	f, mock := newSharing("public")
	now := time.Now()
	f.owner.DisabledAt = &now
	f.shareHash = "x"
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	for _, path := range []string{"/api/v1/public/collections/" + f.col.ID.String(), "/api/v1/public/users/user"} {
		if status, _, _ := callJSON(t, srv, http.MethodGet, path, "", nil); status != http.StatusNotFound {
			t.Errorf("%s: %d", path, status)
		}
	}
}

func TestShareLinks(t *testing.T) {
	f, mock := newSharing("private")
	var created, revoked uuid.UUID
	linkID := uuid.New()
	mock.CreateShareLinkFn = func(_ context.Context, colID uuid.UUID, hash string) (*model.ShareLink, error) {
		created = colID
		f.shareHash = hash
		return &model.ShareLink{ID: linkID, CollectionID: colID, CreatedAt: time.Now()}, nil
	}
	mock.RevokeShareLinkFn = func(_ context.Context, colID, id uuid.UUID) error {
		if id != linkID {
			return store.ErrNotFound
		}
		revoked = id
		f.shareHash = ""
		return nil
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	links := "/api/v1/collections/" + f.col.ID.String() + "/share-links"

	if status, _, _ := callJSON(t, srv, http.MethodPost, links, testutil.AdminToken(), nil); status != http.StatusNotFound {
		t.Errorf("someone else's collection: %d", status)
	}
	status, _, raw := callJSON(t, srv, http.MethodPost, links, testutil.UserToken(), nil)
	var out model.ShareLinkCreated
	if status != http.StatusCreated || json.Unmarshal(raw, &out) != nil || out.Token == "" || created != f.col.ID {
		t.Fatalf("create: %d %s", status, raw)
	}
	if f.shareHash == out.Token {
		t.Error("the raw token must not be stored")
	}

	// Private collection, anonymous reader: the link opens it, hidden field stripped.
	for _, path := range []string{"/api/v1/share", "/api/v1/share/items", "/api/v1/share/stats"} {
		status, _, raw := callJSON(t, srv, http.MethodGet, path+"?token="+out.Token, "", nil)
		if status != http.StatusOK || strings.Contains(string(raw), secret) {
			t.Errorf("%s: %d %s", path, status, raw)
		}
	}
	if status, _, _ := callJSON(t, srv, http.MethodGet, "/api/v1/share", "", nil); status != http.StatusBadRequest {
		t.Errorf("missing token: %d", status)
	}

	if status, _, _ := callJSON(t, srv, http.MethodDelete, links+"/"+linkID.String(), testutil.UserToken(), nil); status != http.StatusNoContent || revoked != linkID {
		t.Fatalf("revoke: %d", status)
	}
	if status, _, _ := callJSON(t, srv, http.MethodGet, "/api/v1/share?token="+out.Token, "", nil); status != http.StatusNotFound {
		t.Errorf("revoked link: %d", status)
	}
}

func TestUpdateCollection_VisibilityAndHiddenFields(t *testing.T) {
	f, mock := newSharing("private")
	var got model.CollectionUpdate
	mock.UpdateCollectionFn = func(_ context.Context, _ uuid.UUID, in model.CollectionUpdate) (*model.Collection, error) {
		got = in
		return f.col, nil
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	path := "/api/v1/collections/" + f.col.ID.String()

	if status, _, _ := callJSON(t, srv, http.MethodPut, path, testutil.UserToken(), map[string]any{"visibility": "everyone"}); status != http.StatusBadRequest {
		t.Errorf("bad visibility: %d", status)
	}
	if status, code, _ := callJSON(t, srv, http.MethodPut, path, testutil.UserToken(), map[string]any{"hidden_fields": []string{"nope"}}); status != http.StatusUnprocessableEntity || code != "validation_error" {
		t.Errorf("unknown field: %d %s", status, code)
	}
	status, _, _ := callJSON(t, srv, http.MethodPut, path, testutil.UserToken(),
		map[string]any{"visibility": "members", "hidden_fields": []string{"purchase_price", "condition", "condition"}})
	if status != http.StatusOK || *got.Visibility != "members" || !got.HiddenFields.Set || !slices.Equal(got.HiddenFields.Value, []string{"condition", "purchase_price"}) {
		t.Errorf("set: %d %+v", status, got)
	}
	_, _, _ = callJSON(t, srv, http.MethodPut, path, testutil.UserToken(), map[string]any{"hidden_fields": nil})
	if !got.HiddenFields.Set || got.HiddenFields.Value != nil {
		t.Errorf("null resets to the default: %+v", got.HiddenFields)
	}
	_, _, _ = callJSON(t, srv, http.MethodPut, path, testutil.UserToken(), map[string]any{"name": "Renommée"})
	if got.HiddenFields.Set {
		t.Error("absent hidden_fields must stay unchanged")
	}
}

func TestListMembers(t *testing.T) {
	u := testutil.RegularUser()
	u.ProfileVisibility = "members"
	mock := &testutil.MockStore{
		ListMembersFn: func(_ context.Context, _ model.Pagination) ([]model.User, int, error) {
			return []model.User{*u}, 1, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	status, _, raw := callJSON(t, srv, http.MethodGet, "/api/v1/members", testutil.UserToken(), nil)
	if status != http.StatusOK || !strings.Contains(string(raw), `"username":"user"`) || strings.Contains(string(raw), "email") || strings.Contains(string(raw), "role") {
		t.Errorf("members: %d %s", status, raw)
	}
	if status, _, _ := callJSON(t, srv, http.MethodGet, "/api/v1/members", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", status)
	}
}
