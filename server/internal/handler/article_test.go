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

var articleID = uuid.MustParse("e0000000-0000-0000-0000-0000000000a1")

func TestCreateArticle_StatusByRole(t *testing.T) {
	var status string
	mock := &testutil.MockStore{
		CreateArticleFn: func(_ context.Context, itemID, author uuid.UUID, in model.ArticleWrite, st string) (*model.Article, error) {
			status = st
			if in.Title != "Tome 12 annoncé" || in.URL != "https://example.org/news" {
				t.Errorf("article : %+v", in)
			}
			return &model.Article{ID: articleID, ItemID: itemID, AuthorID: author, Status: st}, nil
		},
	}
	path := "/items/" + reviewItemID.String() + "/articles"
	body := `{"title":"  Tome 12 annoncé ","url":" https://example.org/news ","message":"","lang":"fr"}`
	if resp, _ := callReview(t, mock, "POST", path, "", body); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonyme : %d", resp.StatusCode)
	}
	if resp, raw := callReview(t, mock, "POST", path, testutil.UserToken(), body); resp.StatusCode != http.StatusCreated || status != "pending" {
		t.Errorf("membre : %d %s %s", resp.StatusCode, status, raw)
	}
	if resp, _ := callReview(t, mock, "POST", path, testutil.ModeratorToken(), body); resp.StatusCode != http.StatusCreated || status != "published" {
		t.Errorf("modérateur : %d %s", resp.StatusCode, status)
	}
	if resp, _ := callReview(t, mock, "POST", path, testutil.ContributorToken(), body); resp.StatusCode != http.StatusCreated || status != "pending" {
		t.Errorf("contributeur : %d %s", resp.StatusCode, status)
	}
}

func TestCreateArticle_Invalid(t *testing.T) {
	mock := &testutil.MockStore{}
	path := "/items/" + reviewItemID.String() + "/articles"
	for name, body := range map[string]string{
		"sans titre":        `{"title":" ","url":"https://example.org","lang":"fr"}`,
		"javascript":        `{"title":"x","url":"javascript:alert(1)","lang":"fr"}`,
		"ftp":               `{"title":"x","url":"ftp://example.org/f","lang":"fr"}`,
		"identifiants":      `{"title":"x","url":"https://user:pw@example.org","lang":"fr"}`,
		"sans hôte":         `{"title":"x","url":"https://","lang":"fr"}`,
		"langue":            `{"title":"x","url":"https://example.org","lang":"es"}`,
		"message trop long": `{"title":"x","url":"https://example.org","lang":"fr","message":"` + strings.Repeat("a", 1001) + `"}`,
	} {
		if resp, raw := callReview(t, mock, "POST", path, testutil.UserToken(), body); resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("%s : %d %s", name, resp.StatusCode, raw)
		}
	}
}

func TestReviewArticle(t *testing.T) {
	var published *bool
	var notified string
	mock := &testutil.MockStore{
		GetArticleFn: func(context.Context, uuid.UUID) (*model.Article, error) {
			return &model.Article{ID: articleID, ItemID: reviewItemID, AuthorID: authorID, Status: "pending"}, nil
		},
		ReviewArticleFn: func(_ context.Context, _, _ uuid.UUID, publish bool, _ string) (*model.Article, error) {
			published = &publish
			return &model.Article{ID: articleID, ItemID: reviewItemID, AuthorID: authorID}, nil
		},
		CreateNotificationsFn: func(_ context.Context, _ []uuid.UUID, kind string, _ any) error {
			notified = kind
			return nil
		},
	}
	base := "/moderation/articles/" + articleID.String()
	if resp, _ := callReview(t, mock, "POST", base+"/publish", testutil.UserToken(), ""); resp.StatusCode != http.StatusForbidden {
		t.Errorf("membre : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "POST", base+"/publish", testutil.ModeratorToken(), ""); resp.StatusCode != http.StatusNoContent || published == nil || !*published || notified != "article_published" {
		t.Errorf("publier : %d %v %s", resp.StatusCode, published, notified)
	}
	if resp, _ := callReview(t, mock, "POST", base+"/reject", testutil.ModeratorToken(), `{"reason":" "}`); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("refus sans motif : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "POST", base+"/reject", testutil.ModeratorToken(), `{"reason":"Lien publicitaire"}`); resp.StatusCode != http.StatusNoContent || *published || notified != "article_rejected" {
		t.Errorf("refuser : %d %s", resp.StatusCode, notified)
	}
	mock.ReviewArticleFn = func(context.Context, uuid.UUID, uuid.UUID, bool, string) (*model.Article, error) {
		return nil, store.ErrConflict
	}
	if resp, raw := callReview(t, mock, "POST", base+"/publish", testutil.AdminToken(), ""); resp.StatusCode != http.StatusConflict || !strings.Contains(raw, "moderation_state") {
		t.Errorf("plus en attente : %d %s", resp.StatusCode, raw)
	}
	mock.GetArticleFn = func(context.Context, uuid.UUID) (*model.Article, error) {
		return &model.Article{ID: articleID, AuthorID: testutil.ModeratorID, Status: "pending"}, nil
	}
	if resp, _ := callReview(t, mock, "POST", base+"/publish", testutil.ModeratorToken(), ""); resp.StatusCode != http.StatusForbidden {
		t.Errorf("son propre article : %d", resp.StatusCode)
	}
}

// Le fil d'un article n'existe qu'une fois publié ; le motif d'un refus ne va qu'à son auteur.
func TestArticleThreadAndReasons(t *testing.T) {
	reason := "Lien publicitaire"
	mock := &testutil.MockStore{
		GetArticleFn: func(context.Context, uuid.UUID) (*model.Article, error) {
			return &model.Article{ID: articleID, AuthorID: authorID, Status: "pending"}, nil
		},
		GetItemFn: func(context.Context, uuid.UUID) (*model.Item, error) { return testutil.SampleItem(), nil },
		ListItemArticlesFn: func(context.Context, uuid.UUID, uuid.UUID, model.Pagination) ([]model.Article, int, error) {
			return []model.Article{{ID: articleID, AuthorID: authorID, Status: "rejected", RejectReason: &reason}}, 1, nil
		},
	}
	if resp, _ := callReview(t, mock, "GET", "/articles/"+articleID.String()+"/comments", "", ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("fil d'un article en attente : %d", resp.StatusCode)
	}
	if resp, raw := callReview(t, mock, "GET", "/items/"+reviewItemID.String()+"/articles", testutil.UserToken(), ""); resp.StatusCode != http.StatusOK || strings.Contains(raw, reason) {
		t.Errorf("motif visible d'un autre : %d %s", resp.StatusCode, raw)
	}
}

func TestArticles_PrivateProfile(t *testing.T) {
	mock := &testutil.MockStore{
		GetItemFn: func(context.Context, uuid.UUID) (*model.Item, error) { return testutil.SampleItem(), nil },
		ListItemArticlesFn: func(context.Context, uuid.UUID, uuid.UUID, model.Pagination) ([]model.Article, int, error) {
			return []model.Article{{ID: articleID, AuthorID: authorID, AuthorProfile: "private",
				Author: &model.ReviewAuthor{Username: "discret"}, Status: "published"}}, 1, nil
		},
	}
	if _, raw := callReview(t, mock, "GET", "/items/"+reviewItemID.String()+"/articles", testutil.UserToken(), ""); strings.Contains(raw, "discret") {
		t.Errorf("auteur au profil privé nommé : %s", raw)
	}
}
