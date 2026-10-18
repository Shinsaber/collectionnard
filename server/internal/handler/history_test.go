package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

// Contributor: writes in the catalogue, never deletes an item nor touches
// meta-types or administration; every write carries its author.
func TestContributorRights(t *testing.T) {
	var actor store.Actor
	mock := &testutil.MockStore{
		GetItemTypeFn: func(_ context.Context, _ uuid.UUID) (*model.ItemType, error) { return testutil.SampleItemType(), nil },
		CreateItemFn: func(ctx context.Context, _ model.ItemCreate) (*model.Item, error) {
			actor = store.ActorFrom(ctx)
			return testutil.SampleItem(), nil
		},
		ImportCatalogFn: func(ctx context.Context, _ *model.CatalogExport, _ bool) (*model.ImportReport, error) {
			actor = store.ActorFrom(ctx)
			return &model.ImportReport{}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	body := map[string]any{"item_type_id": testutil.SampleItemType().ID, "attributes": map[string]any{"titre": "X"}}

	if status, _, raw := callJSON(t, srv, http.MethodPost, "/api/v1/items", testutil.ContributorToken(), body); status != http.StatusCreated {
		t.Fatalf("contributor create: %d %s", status, raw)
	}
	if actor.UserID != testutil.ContributorID || actor.Origin != store.OriginDirect {
		t.Errorf("actor: %+v", actor)
	}
	if status, _, _ := callJSON(t, srv, http.MethodPost, "/api/v1/items", testutil.UserToken(), body); status != http.StatusForbidden {
		t.Errorf("member create: %d", status)
	}
	for _, c := range []struct{ method, path string }{
		{http.MethodDelete, "/api/v1/items/" + uuid.NewString()},
		{http.MethodPost, "/api/v1/item-types"},
		{http.MethodGet, "/api/v1/users"},
		{http.MethodPost, "/api/v1/import"},
	} {
		if status, _, _ := callJSON(t, srv, c.method, c.path, testutil.ContributorToken(), map[string]any{}); status != http.StatusForbidden {
			t.Errorf("contributor %s %s: %d", c.method, c.path, status)
		}
	}

	// Import by an admin: origin `import`.
	cat := map[string]any{"format": model.CatalogFormat, "version": model.ExportVersion}
	if status, _, raw := callJSON(t, srv, http.MethodPost, "/api/v1/import", testutil.AdminToken(), cat); status != http.StatusOK {
		t.Fatalf("import: %d %s", status, raw)
	}
	if actor.UserID != testutil.AdminID || actor.Origin != store.OriginImport {
		t.Errorf("import actor: %+v", actor)
	}
}

func TestItemHistory_Authors(t *testing.T) {
	public, private, disabled := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()
	users := map[uuid.UUID]*model.User{
		public:   {ID: public, Username: "camille", ProfileVisibility: "members"},
		private:  {ID: private, Username: "secret", ProfileVisibility: "private"},
		disabled: {ID: disabled, Username: "parti", ProfileVisibility: "public", DisabledAt: &now},
	}
	entry := func(v int, u *uuid.UUID) model.HistoryEntry {
		return model.HistoryEntry{Kind: "item", Version: v, Origin: "direct", UserID: u,
			Before: json.RawMessage("null"), After: json.RawMessage(`{"titre":"X"}`), CreatedAt: now}
	}
	mock := &testutil.MockStore{
		GetUserByIDFn: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			if u, ok := users[id]; ok {
				return u, nil
			}
			return nil, store.ErrNotFound
		},
		GetItemHistoryFn: func(_ context.Context, _ uuid.UUID) (*model.ItemHistory, error) {
			created := entry(1, nil)
			return &model.ItemHistory{
				Entries: []model.HistoryEntry{entry(4, &public), entry(3, &private), entry(2, &disabled)},
				Created: &created,
			}, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	path := "/api/v1/items/" + uuid.NewString() + "/history"

	read := func(token string) model.ItemHistory {
		status, _, raw := callJSON(t, srv, http.MethodGet, path, token, nil)
		var h model.ItemHistory
		if status != http.StatusOK || json.Unmarshal(raw, &h) != nil {
			t.Fatalf("history: %d %s", status, raw)
		}
		if strings.Contains(string(raw), "user_id") {
			t.Errorf("raw user ids leaked: %s", raw)
		}
		return h
	}
	h := read(testutil.UserToken())
	kinds := []string{h.Entries[0].Author.Kind, h.Entries[1].Author.Kind, h.Entries[2].Author.Kind, h.Created.Author.Kind}
	if strings.Join(kinds, ",") != "user,member,member,system" || h.Entries[0].Author.User.Username != "camille" {
		t.Errorf("member view: %v %+v", kinds, h.Entries[0].Author)
	}
	if h.Entries[1].Author.User != nil {
		t.Error("a private profile must not be named")
	}
	// Admins see every author (moderation).
	h = read(testutil.AdminToken())
	if h.Entries[1].Author.Kind != "user" || h.Entries[2].Author.User.Username != "parti" {
		t.Errorf("admin view: %+v", h.Entries)
	}
}

func TestRevertItem(t *testing.T) {
	var actor store.Actor
	var saved json.RawMessage
	mock := &testutil.MockStore{
		GetItemFn: func(_ context.Context, _ uuid.UUID) (*model.Item, error) { return testutil.SampleItem(), nil },
		GetItemVersionFn: func(_ context.Context, _ uuid.UUID, v int) (json.RawMessage, error) {
			switch v {
			case 1:
				return json.RawMessage(`{"titre":"Ancien"}`), nil
			case 2:
				return json.RawMessage(`{"autre":1}`), nil // ne passe plus le schéma actuel
			case 3:
				return nil, store.ErrDeletedVersion
			}
			return nil, store.ErrNotFound
		},
		UpdateItemFn: func(ctx context.Context, _ uuid.UUID, in model.ItemUpdate) (*model.Item, error) {
			actor, saved = store.ActorFrom(ctx), in.Attributes
			return testutil.SampleItem(), nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	path := "/api/v1/items/" + testutil.SampleItem().ID.String() + "/revert"

	if status, _, raw := callJSON(t, srv, http.MethodPost, path, testutil.ContributorToken(), map[string]int{"version": 1}); status != http.StatusOK {
		t.Fatalf("revert: %d %s", status, raw)
	}
	if string(saved) != `{"titre":"Ancien"}` || actor.Origin != store.OriginRevert || actor.Ref != "1" || actor.UserID != testutil.ContributorID {
		t.Errorf("saved %s with %+v", saved, actor)
	}
	for v, want := range map[int]int{2: http.StatusUnprocessableEntity, 3: http.StatusBadRequest, 9: http.StatusNotFound, 0: http.StatusBadRequest} {
		if status, _, _ := callJSON(t, srv, http.MethodPost, path, testutil.AdminToken(), map[string]int{"version": v}); status != want {
			t.Errorf("version %d: %d, want %d", v, status, want)
		}
	}
	if status, _, _ := callJSON(t, srv, http.MethodPost, path, testutil.UserToken(), map[string]int{"version": 1}); status != http.StatusForbidden {
		t.Errorf("member revert: %d", status)
	}
}

func TestInviteContributor(t *testing.T) {
	mock := &testutil.MockStore{
		CreateInvitationFn: func(_ context.Context, _ string, _ uuid.UUID, role, _ string, _ time.Time) (*model.Invitation, error) {
			return &model.Invitation{ID: uuid.New(), Role: role}, nil
		},
	}
	srv := serverWith(mock, true)
	defer srv.Close()
	if status, _, raw := callJSON(t, srv, http.MethodPost, "/api/v1/invitations", testutil.AdminToken(), map[string]any{"role": "contributor"}); status != http.StatusCreated {
		t.Errorf("admin invites a contributor: %d %s", status, raw)
	}
	if status, _, _ := callJSON(t, srv, http.MethodPost, "/api/v1/invitations", testutil.UserToken(), map[string]any{"role": "contributor"}); status != http.StatusForbidden {
		t.Errorf("member invites a contributor: %d", status)
	}
}
