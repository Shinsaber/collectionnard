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
	"github.com/shinsaber/colec/internal/testutil"
)

// Le fil suit la visibilité de l'avis : une note seule n'en a pas, un avis « membres » n'en a que pour les membres.
func TestReviewComments_Visibility(t *testing.T) {
	path := "/reviews/" + reviewID.String() + "/comments"
	for name, tc := range map[string]struct {
		visibility, body, token string
		want                    int
	}{
		"public, visiteur":   {"public", "x", "", http.StatusOK},
		"membres, visiteur":  {"members", "x", "", http.StatusNotFound},
		"membres, membre":    {"members", "x", testutil.UserToken(), http.StatusOK},
		"note seule":         {"private", "x", testutil.UserToken(), http.StatusNotFound},
		"anonyme sans texte": {"anonymous", "", testutil.UserToken(), http.StatusNotFound},
		"anonyme avec texte": {"anonymous", "x", "", http.StatusOK},
	} {
		mock := &testutil.MockStore{GetReviewFn: reviewBy(authorID, tc.visibility, tc.body)}
		if resp, raw := callReview(t, mock, "GET", path, tc.token, ""); resp.StatusCode != tc.want {
			t.Errorf("%s : %d %s", name, resp.StatusCode, raw)
		}
	}
}

// Dans le fil d'un avis anonyme, son auteur reste « l'auteur de l'avis », sans nom.
func TestReviewComments_AnonymousAuthor(t *testing.T) {
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(authorID, "anonymous", "texte"),
		ListCommentsFn: func(context.Context, string, uuid.UUID, uuid.UUID, model.Pagination) ([]model.Comment, int, error) {
			return []model.Comment{
				{ID: uuid.New(), AuthorID: authorID, Author: &model.ReviewAuthor{Username: "camille"}, Body: "merci"},
				{ID: uuid.New(), AuthorID: testutil.UserID, Author: &model.ReviewAuthor{Username: "moi"}, Body: "d'accord"},
			}, 2, nil
		},
	}
	resp, raw := callReview(t, mock, "GET", "/reviews/"+reviewID.String()+"/comments", testutil.UserToken(), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	if strings.Contains(raw, "camille") || strings.Contains(raw, authorID.String()) {
		t.Errorf("auteur de l'avis anonyme nommé : %s", raw)
	}
	var page struct {
		Data []model.Comment `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &page)
	if !page.Data[0].TargetAuthor || page.Data[0].Author != nil || !page.Data[1].Mine || page.Data[1].Author == nil {
		t.Errorf("fil : %+v", page.Data)
	}
}

func TestCreateReviewComment(t *testing.T) {
	var notified []uuid.UUID
	var payload any
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(authorID, "public", "x"),
		CommentParticipantsFn: func(context.Context, string, uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{testutil.UserID, testutil.AdminID, authorID}, nil
		},
		GetUserByIDFn: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Username: "moi"}, nil
		},
		CreateNotificationsFn: func(_ context.Context, users []uuid.UUID, kind string, p any) error {
			notified, payload = users, p
			return nil
		},
	}
	path := "/reviews/" + reviewID.String() + "/comments"
	if resp, _ := callReview(t, mock, "POST", path, "", `{"body":"x"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonyme : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "POST", path, testutil.UserToken(), `{"body":"   "}`); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("vide : %d", resp.StatusCode)
	}
	if resp, raw := callReview(t, mock, "POST", path, testutil.UserToken(), `{"body":"  Bien vu  "}`); resp.StatusCode != http.StatusCreated || !strings.Contains(raw, `"mine":true`) {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	if len(notified) != 2 || notified[0] != authorID || notified[1] != testutil.AdminID {
		t.Errorf("auteur puis participants, jamais l'émetteur ni de doublon : %v", notified)
	}
	if p, _ := payload.(map[string]any); p["by"] != "moi" {
		t.Errorf("payload : %v", payload)
	}

	mock.GetReviewFn = func(context.Context, uuid.UUID) (*model.Review, error) {
		return &model.Review{ID: reviewID, UserID: authorID, Visibility: "public", Body: "x", CommentsClosed: true}, nil
	}
	if resp, raw := callReview(t, mock, "POST", path, testutil.UserToken(), `{"body":"x"}`); resp.StatusCode != http.StatusConflict || !strings.Contains(raw, "comments_closed") {
		t.Errorf("fil fermé : %d %s", resp.StatusCode, raw)
	}
}

// L'auteur d'un avis anonyme qui répond n'est pas nommé dans la notification.
func TestCreateReviewComment_AnonymousNotification(t *testing.T) {
	var payload map[string]any
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(testutil.UserID, "anonymous", "x"),
		CommentParticipantsFn: func(context.Context, string, uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{authorID}, nil
		},
		CreateNotificationsFn: func(_ context.Context, _ []uuid.UUID, _ string, p any) error {
			payload, _ = p.(map[string]any)
			return nil
		},
	}
	if resp, _ := callReview(t, mock, "POST", "/reviews/"+reviewID.String()+"/comments", testutil.UserToken(), `{"body":"merci"}`); resp.StatusCode != http.StatusCreated {
		t.Fatal(resp.StatusCode)
	}
	if payload["by"] != nil || payload["by_review_author"] != true {
		t.Errorf("payload : %v", payload)
	}
}

func TestReportComment(t *testing.T) {
	commentID := uuid.New()
	mock := &testutil.MockStore{
		GetCommentFn: func(context.Context, uuid.UUID) (*model.Comment, error) {
			return &model.Comment{ID: commentID, TargetKind: "review", TargetID: reviewID, AuthorID: authorID}, nil
		},
		GetReviewFn: reviewBy(testutil.AdminID, "members", "x"),
	}
	body := `{"target_kind":"comment","target_id":"` + commentID.String() + `","reason":"harassment"}`
	if resp, raw := callReview(t, mock, "POST", "/reports", testutil.UserToken(), body); resp.StatusCode != http.StatusCreated {
		t.Errorf("signaler un commentaire d'un fil visible : %d %s", resp.StatusCode, raw)
	}
	mock.GetReviewFn = reviewBy(testutil.AdminID, "private", "x")
	if resp, _ := callReview(t, mock, "POST", "/reports", testutil.UserToken(), body); resp.StatusCode != http.StatusNotFound {
		t.Errorf("fil invisible : %d", resp.StatusCode)
	}
	var kind string
	mock.GetReviewFn = reviewBy(testutil.AdminID, "public", "x")
	mock.ModerateFn = func(_ context.Context, k string, _, _ uuid.UUID, _, _ string) (*model.ModerationResult, error) {
		kind = k
		return &model.ModerationResult{AuthorID: authorID, ItemID: reviewItemID}, nil
	}
	if resp, raw := callReview(t, mock, "POST", "/moderation/comments/"+commentID.String()+"/hide", testutil.ModeratorToken(), `{"reason":"Harcèlement"}`); resp.StatusCode != http.StatusNoContent || kind != "comment" {
		t.Errorf("masquer un commentaire : %d %s %s", resp.StatusCode, raw, kind)
	}
}

func TestCommentOwnership(t *testing.T) {
	mock := &testutil.MockStore{}
	id := uuid.New().String()
	if resp, _ := callReview(t, mock, "PUT", "/comments/"+id, testutil.UserToken(), `{"body":"x"}`); resp.StatusCode != http.StatusNotFound {
		t.Errorf("modifier le commentaire d'un autre : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "DELETE", "/comments/"+id, testutil.UserToken(), ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("supprimer le commentaire d'un autre : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "POST", "/reviews/"+id+"/comments/close", testutil.UserToken(), ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("fermer le fil d'un autre : %d", resp.StatusCode)
	}
}

// Un commentateur au profil privé n'est jamais nommé ; « membres » ne l'est que pour un membre.
func TestReviewComments_PrivateProfile(t *testing.T) {
	someone := uuid.New()
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(authorID, "public", "texte"),
		ListCommentsFn: func(context.Context, string, uuid.UUID, uuid.UUID, model.Pagination) ([]model.Comment, int, error) {
			return []model.Comment{
				{ID: uuid.New(), AuthorID: someone, AuthorProfile: "private", Author: &model.ReviewAuthor{Username: "discret"}, Body: "a"},
				{ID: uuid.New(), AuthorID: uuid.New(), AuthorProfile: "members", Author: &model.ReviewAuthor{Username: "membre"}, Body: "b"},
				{ID: uuid.New(), AuthorID: uuid.New(), AuthorProfile: "public", Author: &model.ReviewAuthor{Username: "public"}, Body: "c"},
			}, 3, nil
		},
	}
	path := "/reviews/" + reviewID.String() + "/comments"
	_, raw := callReview(t, mock, "GET", path, testutil.UserToken(), "")
	if strings.Contains(raw, "discret") || !strings.Contains(raw, `"membre"`) || !strings.Contains(raw, `"public"`) {
		t.Errorf("membre : %s", raw)
	}
	_, raw = callReview(t, mock, "GET", path, "", "")
	if strings.Contains(raw, "discret") || strings.Contains(raw, `"membre"`) || !strings.Contains(raw, `"public"`) {
		t.Errorf("visiteur : %s", raw)
	}
}

// Un commentaire supprimé, ou masqué pour un autre que son auteur, reste une trace sans texte ni auteur.
func TestReviewComments_Traces(t *testing.T) {
	now := time.Now()
	reason := "Hors sujet"
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(authorID, "public", "texte"),
		ListCommentsFn: func(context.Context, string, uuid.UUID, uuid.UUID, model.Pagination) ([]model.Comment, int, error) {
			return []model.Comment{
				{ID: uuid.New(), AuthorID: uuid.New(), AuthorProfile: "public", Author: &model.ReviewAuthor{Username: "parti"}, Body: "", DeletedAt: &now},
				{ID: uuid.New(), AuthorID: uuid.New(), AuthorProfile: "public", Author: &model.ReviewAuthor{Username: "masque"}, Body: "insulte", HiddenAt: &now, HiddenReason: &reason},
				{ID: uuid.New(), AuthorID: testutil.UserID, AuthorProfile: "public", Author: &model.ReviewAuthor{Username: "moi"}, Body: "mon texte masqué", HiddenAt: &now, HiddenReason: &reason},
			}, 3, nil
		},
	}
	_, raw := callReview(t, mock, "GET", "/reviews/"+reviewID.String()+"/comments", testutil.UserToken(), "")
	for _, leak := range []string{"parti", "masque", "insulte"} {
		if strings.Contains(raw, leak) {
			t.Errorf("%q dans une trace : %s", leak, raw)
		}
	}
	if !strings.Contains(raw, "mon texte masqué") || !strings.Contains(raw, reason) {
		t.Errorf("son auteur voit son commentaire masqué et le motif : %s", raw)
	}
}
