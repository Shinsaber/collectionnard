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

// friendsOf: AdminID and UserID are friends.
func friendsOf(_ context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	switch id {
	case testutil.AdminID:
		return []uuid.UUID{testutil.UserID}, nil
	case testutil.UserID:
		return []uuid.UUID{testutil.AdminID}, nil
	}
	return nil, nil
}

func friendMock() (*testutil.MockStore, *[]string) {
	camille := &model.User{ID: uuid.New(), Username: "camille", ProfileVisibility: "members"}
	secret := &model.User{ID: uuid.New(), Username: "secret", ProfileVisibility: "private"}
	var kinds []string
	return &testutil.MockStore{
		GetUserByUsernameFn: func(_ context.Context, name string) (*model.User, string, error) {
			switch name {
			case "camille":
				return camille, "", nil
			case "secret":
				return secret, "", nil
			case "user":
				return testutil.RegularUser(), "", nil
			}
			return nil, "", store.ErrNotFound
		},
		GetUserByIDFn: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Username: "user"}, nil
		},
		RequestFriendFn: func(_ context.Context, from, to uuid.UUID) (*model.Friendship, error) {
			return &model.Friendship{ID: uuid.New(), Status: model.FriendPending, OtherID: to, OtherProfile: "private",
				User: &model.ReviewAuthor{Username: "camille", DisplayName: new("Camille D.")}}, nil
		},
		CreateNotificationsFn: func(_ context.Context, _ []uuid.UUID, kind string, _ any) error {
			kinds = append(kinds, kind)
			return nil
		},
	}, &kinds
}

func TestRequestFriend(t *testing.T) {
	mock, kinds := friendMock()
	user := testutil.UserToken()
	const path = "/api/v1/me/friends"

	resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"username":" camille "}`))
	if resp.StatusCode != http.StatusCreated || len(*kinds) != 1 || (*kinds)[0] != model.NotifyFriendRequest {
		t.Fatalf("request: %d %s %v", resp.StatusCode, body, *kinds)
	}
	if strings.Contains(body, "Camille D.") {
		t.Errorf("a profile no longer visible that did not accept shows only its username: %s", body)
	}
	cases := []struct {
		body string
		want int
		code string
	}{
		{`{"username":""}`, http.StatusUnprocessableEntity, "validation_error"},
		{`{"username":"user"}`, http.StatusUnprocessableEntity, "validation_error"},
		{`{"username":"nobody"}`, http.StatusNotFound, "not_found"},
		{`{"username":"secret"}`, http.StatusNotFound, "not_found"}, // profil privé : comme inconnu
	}
	for _, c := range cases {
		if resp, body := call(t, mock, "POST", path, user, strings.NewReader(c.body)); resp.StatusCode != c.want || !strings.Contains(body, c.code) {
			t.Errorf("%s: %d %s", c.body, resp.StatusCode, body)
		}
	}
	if resp, _ := call(t, mock, "POST", path, "", strings.NewReader(`{"username":"camille"}`)); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", resp.StatusCode)
	}

	mock.RequestFriendFn = func(context.Context, uuid.UUID, uuid.UUID) (*model.Friendship, error) { return nil, store.ErrConflict }
	if resp, body := call(t, mock, "POST", path, user, strings.NewReader(`{"username":"camille"}`)); resp.StatusCode != http.StatusConflict || !strings.Contains(body, "friendship_exists") {
		t.Errorf("twice: %d %s", resp.StatusCode, body)
	}
}

func TestRequestFriend_AcceptsTheirRequest(t *testing.T) {
	mock, kinds := friendMock()
	mock.RequestFriendFn = func(_ context.Context, _, to uuid.UUID) (*model.Friendship, error) {
		return &model.Friendship{ID: uuid.New(), Status: model.FriendAccepted, OtherID: to,
			User: &model.ReviewAuthor{Username: "camille", DisplayName: new("Camille D.")}}, nil
	}
	resp, body := call(t, mock, "POST", "/api/v1/me/friends", testutil.UserToken(), strings.NewReader(`{"username":"camille"}`))
	if resp.StatusCode != http.StatusCreated || !strings.Contains(body, "Camille D.") || len(*kinds) != 1 || (*kinds)[0] != model.NotifyFriendAccepted {
		t.Errorf("%d %s %v", resp.StatusCode, body, *kinds)
	}
}

// A member who blocked the requester gets no request and no notification; the
// requester cannot tell. One I blocked must be unblocked first.
func TestRequestFriend_Blocks(t *testing.T) {
	mock, kinds := friendMock()
	var camille uuid.UUID
	mock.IsBlockedFn = func(_ context.Context, blocker, _ uuid.UUID) (bool, error) { return blocker == camille, nil }
	u, _, _ := mock.GetUserByUsernameFn(context.Background(), "camille")
	camille = u.ID
	if resp, body := call(t, mock, "POST", "/api/v1/me/friends", testutil.UserToken(), strings.NewReader(`{"username":"camille"}`)); resp.StatusCode != http.StatusCreated || len(*kinds) != 0 {
		t.Errorf("silent refusal: %d %s %v", resp.StatusCode, body, *kinds)
	}
	mock.IsBlockedFn = func(_ context.Context, blocker, _ uuid.UUID) (bool, error) { return blocker == testutil.UserID, nil }
	if resp, body := call(t, mock, "POST", "/api/v1/me/friends", testutil.UserToken(), strings.NewReader(`{"username":"camille"}`)); resp.StatusCode != http.StatusConflict || !strings.Contains(body, "member_blocked") {
		t.Errorf("I blocked them: %d %s", resp.StatusCode, body)
	}
}

func TestFriendRequests_AcceptListDelete(t *testing.T) {
	mock, kinds := friendMock()
	id, other := uuid.New(), uuid.New()
	var notified []uuid.UUID
	mock.CreateNotificationsFn = func(_ context.Context, to []uuid.UUID, kind string, _ any) error {
		notified, *kinds = to, append(*kinds, kind)
		return nil
	}
	mock.AcceptFriendRequestFn = func(_ context.Context, me, got uuid.UUID) (*model.Friendship, error) {
		if me != testutil.UserID || got != id {
			return nil, store.ErrNotFound
		}
		return &model.Friendship{ID: id, Status: model.FriendAccepted, OtherID: other, User: &model.ReviewAuthor{Username: "camille"}}, nil
	}
	var status string
	mock.ListFriendshipsFn = func(_ context.Context, _ uuid.UUID, s string, _ model.Pagination) ([]model.Friendship, int, error) {
		status = s
		return []model.Friendship{}, 0, nil
	}
	mock.DeleteFriendshipFn = func(_ context.Context, me, got uuid.UUID) error {
		if me != testutil.UserID || got != id {
			return store.ErrNotFound
		}
		return nil
	}
	user := testutil.UserToken()

	accept := "/api/v1/me/friend-requests/" + id.String() + "/accept"
	if resp, body := call(t, mock, "POST", accept, user, nil); resp.StatusCode != http.StatusOK || len(notified) != 1 || notified[0] != other || (*kinds)[0] != model.NotifyFriendAccepted {
		t.Errorf("accept: %d %s %v", resp.StatusCode, body, notified)
	}
	if resp, _ := call(t, mock, "POST", accept, testutil.AdminToken(), nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("accept a request sent to another: %d", resp.StatusCode)
	}
	for path, want := range map[string]string{"/api/v1/me/friends": model.FriendAccepted, "/api/v1/me/friend-requests": model.FriendPending} {
		if resp, _ := call(t, mock, "GET", path, user, nil); resp.StatusCode != http.StatusOK || status != want {
			t.Errorf("%s: %d %s", path, resp.StatusCode, status)
		}
	}
	if resp, _ := call(t, mock, "DELETE", "/api/v1/me/friends/"+id.String(), user, nil); resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete: %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "DELETE", "/api/v1/me/friends/"+uuid.New().String(), user, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("delete unknown: %d", resp.StatusCode)
	}
}

// The `friends` visibility opens a collection and a profile to the owner's
// friends only (AdminID is a friend of UserID, the moderator is not).
func TestPublicCollection_FriendsVisibility(t *testing.T) {
	f, mock := newSharing("friends")
	mock.FriendIDsFn = friendsOf
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	for _, c := range []struct {
		token string
		want  int
	}{
		{testutil.AdminToken(), http.StatusOK},
		{testutil.UserToken(), http.StatusOK},
		{testutil.ModeratorToken(), http.StatusNotFound},
		{"", http.StatusNotFound},
	} {
		for _, path := range []string{"/api/v1/public/collections/" + f.col.ID.String(), "/api/v1/public/users/user"} {
			if status, _, raw := callJSON(t, srv, http.MethodGet, path, c.token, nil); status != c.want {
				t.Errorf("%s: %d, want %d (%s)", path, status, c.want, raw)
			}
		}
	}
	status, _, raw := callJSON(t, srv, http.MethodGet, "/api/v1/public/users/user/collections", testutil.AdminToken(), nil)
	if status != http.StatusOK || !strings.Contains(string(raw), f.col.ID.String()) {
		t.Errorf("a friend lists the friends collections: %d %s", status, raw)
	}
}

func TestUpdateCollection_FriendsVisibility(t *testing.T) {
	f, mock := newSharing("private")
	var got *string
	mock.UpdateCollectionFn = func(_ context.Context, _ uuid.UUID, in model.CollectionUpdate) (*model.Collection, error) {
		got = in.Visibility
		return f.col, nil
	}
	mock.GetCollectionTypeFn = func(context.Context, uuid.UUID) (*model.CollectionType, error) { return f.col.CollectionType, nil }
	path := "/api/v1/collections/" + f.col.ID.String()
	if resp, body := call(t, mock, "PUT", path, testutil.UserToken(), strings.NewReader(`{"name":"x","visibility":"friends"}`)); resp.StatusCode != http.StatusOK || got == nil || *got != "friends" {
		t.Errorf("%d %s", resp.StatusCode, body)
	}
}

// A `friends` review is signed for the author's friends only.
func TestItemReviews_FriendsSigned(t *testing.T) {
	mock := &testutil.MockStore{
		FriendIDsFn: friendsOf,
		GetItemReviewTypeFn: func(context.Context, uuid.UUID) (*model.ReviewType, error) {
			return &model.ReviewType{ID: uuid.New()}, nil
		},
		ListItemReviewsFn: func(context.Context, uuid.UUID, model.ReviewListParams, model.Pagination) ([]model.Review, int, error) {
			return []model.Review{{ID: uuid.New(), UserID: testutil.UserID, Visibility: "friends", Body: "x",
				Author: &model.ReviewAuthor{Username: "user"}}}, 1, nil
		},
	}
	path := "/items/" + uuid.New().String() + "/reviews"
	if _, body := callReview(t, mock, "GET", path, testutil.AdminToken(), ""); !strings.Contains(body, `"username":"user"`) {
		t.Errorf("friend: %s", body)
	}
	if _, body := callReview(t, mock, "GET", path, testutil.ModeratorToken(), ""); strings.Contains(body, `"username"`) {
		t.Errorf("not a friend: %s", body)
	}
}

func TestFriendLinks(t *testing.T) {
	owner := uuid.New()
	var hashes []string
	link := &model.FriendLink{ID: uuid.New(), OwnerID: owner, Owner: &model.ReviewAuthor{Username: "camille"}}
	var kinds []string
	var notified []uuid.UUID
	var used string
	mock := &testutil.MockStore{
		CreateFriendLinkFn: func(_ context.Context, _ uuid.UUID, hash string, _ time.Time) (*model.FriendLink, error) {
			hashes = append(hashes, hash)
			return &model.FriendLink{ID: uuid.New(), OwnerID: testutil.UserID, Owner: &model.ReviewAuthor{Username: "user"}}, nil
		},
		GetFriendLinkFn: func(_ context.Context, hash string) (*model.FriendLink, error) {
			if len(hashes) == 0 || hash != hashes[0] {
				return nil, store.ErrNotFound
			}
			return link, nil
		},
		UseFriendLinkFn: func(_ context.Context, hash string, me uuid.UUID) (*model.Friendship, error) {
			used = hash
			return &model.Friendship{ID: uuid.New(), Status: model.FriendAccepted, OtherID: owner, User: link.Owner}, nil
		},
		GetUserByIDFn: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Username: "user"}, nil
		},
		CreateNotificationsFn: func(_ context.Context, to []uuid.UUID, kind string, _ any) error {
			notified, kinds = to, append(kinds, kind)
			return nil
		},
	}
	user := testutil.UserToken()

	resp, body := call(t, mock, "POST", "/api/v1/me/friend-links", user, nil)
	var created model.FriendLink
	if resp.StatusCode != http.StatusCreated || json.Unmarshal([]byte(body), &created) != nil || created.Token == "" || strings.Contains(body, `"user"`) {
		t.Fatalf("create: %d %s", resp.StatusCode, body)
	}
	if hashes[0] == created.Token {
		t.Error("only the hash of the token is stored")
	}
	open := "/api/v1/friend-links/" + created.Token
	if resp, body := call(t, mock, "GET", open, testutil.AdminToken(), nil); resp.StatusCode != http.StatusOK || !strings.Contains(body, "camille") {
		t.Errorf("open: %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, mock, "GET", "/api/v1/friend-links/nope", testutil.AdminToken(), nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown token: %d", resp.StatusCode)
	}
	if resp, _ := call(t, mock, "GET", open, "", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", resp.StatusCode)
	}
	if resp, body := call(t, mock, "POST", open+"/accept", testutil.AdminToken(), nil); resp.StatusCode != http.StatusOK || used != hashes[0] ||
		len(notified) != 1 || notified[0] != owner || kinds[0] != model.NotifyFriendAccepted {
		t.Errorf("accept: %d %s %v", resp.StatusCode, body, notified)
	}

	link.OwnerID = testutil.UserID
	if resp, _ := call(t, mock, "POST", open+"/accept", user, nil); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("own link: %d", resp.StatusCode)
	}
	link.OwnerID = owner
	mock.IsBlockedFn = func(_ context.Context, blocker, _ uuid.UUID) (bool, error) { return blocker == owner, nil }
	if resp, _ := call(t, mock, "GET", open, testutil.AdminToken(), nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("link of a member who blocked me: %d", resp.StatusCode)
	}
	mock.IsBlockedFn = nil
	mock.UseFriendLinkFn = func(context.Context, string, uuid.UUID) (*model.Friendship, error) { return nil, store.ErrConflict }
	if resp, body := call(t, mock, "POST", open+"/accept", testutil.AdminToken(), nil); resp.StatusCode != http.StatusConflict || !strings.Contains(body, "friendship_exists") {
		t.Errorf("already friends: %d %s", resp.StatusCode, body)
	}

	mock.ListFriendLinksFn = func(context.Context, uuid.UUID) ([]model.FriendLink, error) {
		return make([]model.FriendLink, 10), nil
	}
	if resp, body := call(t, mock, "POST", "/api/v1/me/friend-links", user, nil); resp.StatusCode != http.StatusConflict || !strings.Contains(body, "friend_links_limit") {
		t.Errorf("limit: %d %s", resp.StatusCode, body)
	}
}
