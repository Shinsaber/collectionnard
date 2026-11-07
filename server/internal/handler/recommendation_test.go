package handler_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

// recipient is a member of the directory who accepts recommendations.
func recipient(visibility string, accepts bool) func(context.Context, string) (*model.User, string, error) {
	return func(_ context.Context, name string) (*model.User, string, error) {
		if name != "camille" {
			return nil, "", store.ErrNotFound
		}
		return &model.User{ID: authorID, Username: "camille", ProfileVisibility: visibility, AcceptsRecommendations: accepts}, "", nil
	}
}

func recommendationMock() (*testutil.MockStore, *[]uuid.UUID) {
	var notified []uuid.UUID
	return &testutil.MockStore{
		GetItemFn:           func(context.Context, uuid.UUID) (*model.Item, error) { return &model.Item{ID: reviewItemID}, nil },
		GetUserByUsernameFn: recipient("members", true),
		CreateRecommendationFn: func(_ context.Context, sender, to, item uuid.UUID, msg string) (*model.Recommendation, error) {
			return &model.Recommendation{ID: uuid.New(), ItemID: item, SenderID: sender, Sender: &model.ReviewAuthor{Username: "user"}, Message: msg}, nil
		},
		CreateNotificationsFn: func(_ context.Context, users []uuid.UUID, _ string, _ any) error {
			notified = users
			return nil
		},
	}, &notified
}

func TestCreateRecommendation(t *testing.T) {
	path := "/items/" + reviewItemID.String() + "/recommendations"
	mock, notified := recommendationMock()
	if resp, _ := callReview(t, mock, "POST", path, "", `{"to":"camille"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonyme : %d", resp.StatusCode)
	}
	for name, body := range map[string]string{
		"sans destinataire": `{"to":"  "}`,
		"message trop long": `{"to":"camille","message":"` + strings.Repeat("a", 501) + `"}`,
		"à soi-même":        `{"to":"user"}`,
	} {
		m, _ := recommendationMock()
		m.GetUserByUsernameFn = func(_ context.Context, name string) (*model.User, string, error) {
			return &model.User{ID: testutil.UserID, Username: name, ProfileVisibility: "public", AcceptsRecommendations: true}, "", nil
		}
		if resp, raw := callReview(t, m, "POST", path, testutil.UserToken(), body); resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("%s : %d %s", name, resp.StatusCode, raw)
		}
	}
	resp, raw := callReview(t, mock, "POST", path, testutil.UserToken(), `{"to":"camille","message":" À voir ! "}`)
	if resp.StatusCode != http.StatusCreated || !strings.Contains(raw, `"message":"À voir !"`) {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	if len(*notified) != 1 || (*notified)[0] != authorID {
		t.Errorf("destinataire notifié : %v", *notified)
	}
	if strings.Contains(raw, testutil.UserID.String()) {
		t.Errorf("identifiant de l'expéditeur sérialisé : %s", raw)
	}
}

// Profil privé, refus et blocage : l'expéditeur ne distingue pas les deux derniers.
func TestCreateRecommendation_Refused(t *testing.T) {
	path := "/items/" + reviewItemID.String() + "/recommendations"
	for name, tc := range map[string]struct {
		visibility string
		accepts    bool
		blocked    bool
		user       string
		want       int
		code       string
	}{
		"inconnu":       {"public", true, false, "dominique", http.StatusNotFound, "not_found"},
		"profil privé":  {"private", true, false, "camille", http.StatusNotFound, "not_found"},
		"refuse":        {"public", false, false, "camille", http.StatusForbidden, "recommendations_refused"},
		"m'a bloqué":    {"public", true, true, "camille", http.StatusForbidden, "recommendations_refused"},
		"déjà envoyée":  {"public", true, false, "camille", http.StatusConflict, "already_recommended"},
		"profil membre": {"members", true, false, "camille", http.StatusCreated, ""},
	} {
		mock, _ := recommendationMock()
		mock.GetUserByUsernameFn = recipient(tc.visibility, tc.accepts)
		mock.IsBlockedFn = func(_ context.Context, blocker, blocked uuid.UUID) (bool, error) {
			return tc.blocked && blocker == authorID && blocked == testutil.UserID, nil
		}
		if tc.code == "already_recommended" {
			mock.CreateRecommendationFn = func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (*model.Recommendation, error) {
				return nil, store.ErrConflict
			}
		}
		resp, raw := callReview(t, mock, "POST", path, testutil.UserToken(), `{"to":"`+tc.user+`"}`)
		if resp.StatusCode != tc.want || !strings.Contains(raw, tc.code) {
			t.Errorf("%s : %d %s", name, resp.StatusCode, raw)
		}
	}
}

func TestBlocks(t *testing.T) {
	var blocked uuid.UUID
	var anonymous bool
	mock := &testutil.MockStore{
		GetUserByUsernameFn: recipient("public", true),
		GetReviewFn:         reviewBy(authorID, "anonymous", "texte"),
		BlockUserFn: func(_ context.Context, _, id uuid.UUID, anon bool) (*model.Block, error) {
			blocked, anonymous = id, anon
			return &model.Block{ID: uuid.New(), BlockedID: id, BlockedProfile: "public", Anonymous: anon,
				User: &model.ReviewAuthor{Username: "camille"}}, nil
		},
	}
	if resp, _ := callReview(t, mock, "POST", "/me/blocks", "", `{"username":"camille"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonyme : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "POST", "/me/blocks", testutil.UserToken(), `{}`); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("rien à bloquer : %d", resp.StatusCode)
	}
	resp, raw := callReview(t, mock, "POST", "/me/blocks", testutil.UserToken(), `{"username":"camille"}`)
	if resp.StatusCode != http.StatusCreated || blocked != authorID || anonymous || !strings.Contains(raw, "camille") {
		t.Errorf("par nom : %d %s", resp.StatusCode, raw)
	}
	// Depuis un avis anonyme : bloqué sans jamais être nommé.
	resp, raw = callReview(t, mock, "POST", "/me/blocks", testutil.UserToken(),
		`{"content_kind":"review","content_id":"`+reviewID.String()+`"}`)
	if resp.StatusCode != http.StatusCreated || blocked != authorID || !anonymous || strings.Contains(raw, "camille") {
		t.Errorf("avis anonyme : %d %s", resp.StatusCode, raw)
	}
	mock.GetReviewFn = reviewBy(testutil.UserID, "public", "texte")
	if resp, _ := callReview(t, mock, "POST", "/me/blocks", testutil.UserToken(),
		`{"content_kind":"review","content_id":"`+reviewID.String()+`"}`); resp.StatusCode != http.StatusNotFound {
		t.Errorf("son propre avis : %d", resp.StatusCode)
	}
}

// Un membre bloqué ne commente ni ne réagit à un avis signé ; sur un avis anonyme, le blocage ne fait que masquer.
func TestBlocked_Interactions(t *testing.T) {
	isBlocked := func(_ context.Context, blocker, blocked uuid.UUID) (bool, error) {
		return blocker == authorID && blocked == testutil.UserID, nil
	}
	for name, tc := range map[string]struct {
		visibility string
		want       int
	}{
		"avis signé":   {"public", http.StatusForbidden},
		"avis anonyme": {"anonymous", http.StatusCreated},
	} {
		var notified []uuid.UUID
		mock := &testutil.MockStore{
			GetReviewFn: reviewBy(authorID, tc.visibility, "texte"),
			IsBlockedFn: isBlocked,
			CreateCommentFn: func(_ context.Context, kind string, target, author uuid.UUID, body string) (*model.Comment, error) {
				return &model.Comment{ID: uuid.New(), TargetKind: kind, TargetID: target, AuthorID: author, Body: body}, nil
			},
			BlockersFn: func(_ context.Context, emitter uuid.UUID, users []uuid.UUID) ([]uuid.UUID, error) {
				return []uuid.UUID{authorID}, nil
			},
			CreateNotificationsFn: func(_ context.Context, users []uuid.UUID, _ string, _ any) error {
				notified = users
				return nil
			},
		}
		resp, raw := callReview(t, mock, "POST", "/reviews/"+reviewID.String()+"/comments", testutil.UserToken(), `{"body":"x"}`)
		if resp.StatusCode != tc.want {
			t.Errorf("commentaire, %s : %d %s", name, resp.StatusCode, raw)
		}
		if len(notified) != 0 {
			t.Errorf("le bloqueur n'est pas notifié, %s : %v", name, notified)
		}
		mock.SetReactionFn = func(context.Context, uuid.UUID, uuid.UUID, string, bool) error { return nil }
		want := http.StatusForbidden
		if tc.visibility == "anonymous" {
			want = http.StatusNoContent
		}
		if resp, raw := callReview(t, mock, "PUT", "/reviews/"+reviewID.String()+"/reactions/funny", testutil.UserToken(), ""); resp.StatusCode != want {
			t.Errorf("réaction, %s : %d %s", name, resp.StatusCode, raw)
		}
		if resp, _ := callReview(t, mock, "DELETE", "/reviews/"+reviewID.String()+"/reactions/funny", testutil.UserToken(), ""); resp.StatusCode != http.StatusNoContent {
			t.Errorf("retirer sa réaction reste permis, %s : %d", name, resp.StatusCode)
		}
	}
}

// Les commentaires d'un membre bloqué deviennent des traces pour le bloqueur, sauf l'auteur d'un avis anonyme.
func TestBlocked_CommentTraces(t *testing.T) {
	other := uuid.New()
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(authorID, "anonymous", "texte"),
		BlockedByFn: func(context.Context, uuid.UUID) ([]uuid.UUID, error) { return []uuid.UUID{other, authorID}, nil },
		ListCommentsFn: func(context.Context, string, uuid.UUID, uuid.UUID, model.Pagination) ([]model.Comment, int, error) {
			return []model.Comment{
				{ID: uuid.New(), AuthorID: other, AuthorProfile: "public", Author: &model.ReviewAuthor{Username: "dominique"}, Body: "pique"},
				{ID: uuid.New(), AuthorID: authorID, AuthorProfile: "public", Author: &model.ReviewAuthor{Username: "camille"}, Body: "réponse"},
			}, 2, nil
		},
	}
	resp, raw := callReview(t, mock, "GET", "/reviews/"+reviewID.String()+"/comments", testutil.UserToken(), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	if strings.Contains(raw, "dominique") || strings.Contains(raw, "pique") || !strings.Contains(raw, `"blocked":true`) {
		t.Errorf("trace du membre bloqué : %s", raw)
	}
	if !strings.Contains(raw, "réponse") || strings.Contains(raw, "camille") || strings.Count(raw, `"blocked":true`) != 1 {
		t.Errorf("l'auteur anonyme n'est ni nommé ni marqué : %s", raw)
	}
}
