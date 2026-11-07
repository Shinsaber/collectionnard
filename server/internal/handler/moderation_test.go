package handler_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

var reviewID = uuid.MustParse("e0000000-0000-0000-0000-0000000000f1")

func reviewBy(author uuid.UUID, visibility, body string) func(context.Context, uuid.UUID) (*model.Review, error) {
	return func(context.Context, uuid.UUID) (*model.Review, error) {
		return &model.Review{ID: reviewID, ItemID: reviewItemID, UserID: author, Visibility: visibility, Body: body, Rating: 3}, nil
	}
}

func TestCreateReport(t *testing.T) {
	var got model.ReportCreate
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(authorID, "public", "texte"),
		CreateReportFn: func(_ context.Context, reporter uuid.UUID, in model.ReportCreate) (*model.Report, error) {
			got = in
			if reporter != testutil.UserID {
				t.Errorf("signaleur : %s", reporter)
			}
			return &model.Report{ID: uuid.New(), Status: "open"}, nil
		},
	}
	body := `{"target_kind":"review","target_id":"` + reviewID.String() + `","reason":"abusive","details":"  insulte  "}`
	if resp, _ := callReview(t, mock, "POST", "/reports", "", body); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonyme : %d", resp.StatusCode)
	}
	if resp, raw := callReview(t, mock, "POST", "/reports", testutil.UserToken(), body); resp.StatusCode != http.StatusCreated {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	if got.Details != "insulte" || got.Reason != "abusive" {
		t.Errorf("signalement : %+v", got)
	}

	for name, b := range map[string]string{
		"cible inconnue": `{"target_kind":"profile","target_id":"` + reviewID.String() + `","reason":"abusive"}`,
		"motif inconnu":  `{"target_kind":"review","target_id":"` + reviewID.String() + `","reason":"boring"}`,
	} {
		if resp, _ := callReview(t, mock, "POST", "/reports", testutil.UserToken(), b); resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("%s : %d", name, resp.StatusCode)
		}
	}

	mock.CreateReportFn = func(context.Context, uuid.UUID, model.ReportCreate) (*model.Report, error) {
		return nil, store.ErrConflict
	}
	if resp, raw := callReview(t, mock, "POST", "/reports", testutil.UserToken(), body); resp.StatusCode != http.StatusConflict || !strings.Contains(raw, "report_exists") {
		t.Errorf("déjà signalé : %d %s", resp.StatusCode, raw)
	}
}

// On ne signale que ce qu'on voit : ni son propre avis, ni une note seule, ni un avis masqué, ni un avis
// réservé aux membres pour… un membre qui le voit (permis), etc.
func TestCreateReport_OnlyWhatIsShown(t *testing.T) {
	body := `{"target_kind":"review","target_id":"` + reviewID.String() + `","reason":"spam"}`
	hidden := time.Now()
	for name, tc := range map[string]struct {
		get  func(context.Context, uuid.UUID) (*model.Review, error)
		want int
	}{
		"son propre avis":           {reviewBy(testutil.UserID, "public", "x"), http.StatusForbidden},
		"note seule":                {reviewBy(authorID, "private", "x"), http.StatusNotFound},
		"anonyme sans texte":        {reviewBy(authorID, "anonymous", ""), http.StatusNotFound},
		"anonyme avec texte":        {reviewBy(authorID, "anonymous", "x"), http.StatusCreated},
		"membres, lu par un membre": {reviewBy(authorID, "members", "x"), http.StatusCreated},
		"masqué": {func(context.Context, uuid.UUID) (*model.Review, error) {
			return &model.Review{ID: reviewID, UserID: authorID, Visibility: "public", HiddenAt: &hidden}, nil
		}, http.StatusNotFound},
	} {
		mock := &testutil.MockStore{GetReviewFn: tc.get}
		if resp, raw := callReview(t, mock, "POST", "/reports", testutil.UserToken(), body); resp.StatusCode != tc.want {
			t.Errorf("%s : %d %s", name, resp.StatusCode, raw)
		}
	}
}

func TestModeration_Roles(t *testing.T) {
	calls := 0
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(authorID, "public", "x"),
		ModerateFn: func(_ context.Context, kind string, _, moderator uuid.UUID, action, reason string) (*model.ModerationResult, error) {
			calls++
			if action != "hide" || reason != "Propos injurieux" {
				t.Errorf("action : %s %q", action, reason)
			}
			return &model.ModerationResult{AuthorID: authorID, ItemID: reviewItemID}, nil
		},
		CreateNotificationsFn: func(_ context.Context, users []uuid.UUID, kind string, _ any) error {
			if kind != model.NotifyReviewHidden || len(users) != 1 || users[0] != authorID {
				t.Errorf("notification : %s %v", kind, users)
			}
			return nil
		},
	}
	path := "/moderation/reviews/" + reviewID.String() + "/hide"
	body := `{"reason":"  Propos injurieux "}`
	for name, tok := range map[string]string{"membre": testutil.UserToken(), "contributeur": testutil.ContributorToken()} {
		if resp, _ := callReview(t, mock, "POST", path, tok, body); resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s : %d", name, resp.StatusCode)
		}
	}
	for name, tok := range map[string]string{"modérateur": testutil.ModeratorToken(), "admin": testutil.AdminToken()} {
		if resp, raw := callReview(t, mock, "POST", path, tok, body); resp.StatusCode != http.StatusNoContent {
			t.Errorf("%s : %d %s", name, resp.StatusCode, raw)
		}
	}
	if calls != 2 {
		t.Errorf("masquages : %d", calls)
	}
	if resp, _ := callReview(t, mock, "POST", path, testutil.ModeratorToken(), `{"reason":" "}`); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("masquer sans motif : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "GET", "/moderation/queue", testutil.ContributorToken(), ""); resp.StatusCode != http.StatusForbidden {
		t.Errorf("file pour un contributeur : %d", resp.StatusCode)
	}
	if resp, raw := callReview(t, mock, "GET", "/moderation/queue", testutil.ModeratorToken(), ""); resp.StatusCode != http.StatusOK || !strings.Contains(raw, `"data":[]`) {
		t.Errorf("file : %d %s", resp.StatusCode, raw)
	}
}

func TestModeration_OwnAndState(t *testing.T) {
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(testutil.ModeratorID, "public", "x"),
		ModerateFn: func(context.Context, string, uuid.UUID, uuid.UUID, string, string) (*model.ModerationResult, error) {
			return nil, store.ErrConflict
		},
	}
	base := "/moderation/reviews/" + reviewID.String()
	if resp, raw := callReview(t, mock, "POST", base+"/dismiss", testutil.ModeratorToken(), ""); resp.StatusCode != http.StatusForbidden || !strings.Contains(raw, "own_content") {
		t.Errorf("son propre avis : %d %s", resp.StatusCode, raw)
	}
	mock.GetReviewFn = reviewBy(authorID, "public", "x")
	if resp, raw := callReview(t, mock, "POST", base+"/restore", testutil.ModeratorToken(), ""); resp.StatusCode != http.StatusConflict || !strings.Contains(raw, "moderation_state") {
		t.Errorf("rien à rétablir : %d %s", resp.StatusCode, raw)
	}
}

func TestReactions(t *testing.T) {
	var on []bool
	mock := &testutil.MockStore{
		GetReviewFn: reviewBy(authorID, "public", "x"),
		SetReactionFn: func(_ context.Context, _, user uuid.UUID, kind string, add bool) error {
			if kind != "tempting" || user != testutil.UserID {
				t.Errorf("réaction : %s %s", kind, user)
			}
			on = append(on, add)
			return nil
		},
	}
	path := "/reviews/" + reviewID.String() + "/reactions/tempting"
	if resp, _ := callReview(t, mock, "PUT", path, testutil.UserToken(), ""); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("ajout : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "DELETE", path, testutil.UserToken(), ""); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("retrait : %d", resp.StatusCode)
	}
	if len(on) != 2 || !on[0] || on[1] {
		t.Errorf("ajout puis retrait : %v", on)
	}
	if resp, _ := callReview(t, mock, "PUT", "/reviews/"+reviewID.String()+"/reactions/mean", testutil.UserToken(), ""); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("sorte inconnue : %d", resp.StatusCode)
	}
	mock.GetReviewFn = reviewBy(testutil.UserID, "public", "x")
	if resp, _ := callReview(t, mock, "PUT", path, testutil.UserToken(), ""); resp.StatusCode != http.StatusForbidden {
		t.Errorf("son propre avis : %d", resp.StatusCode)
	}
}

func TestRateLimit_Reports(t *testing.T) {
	mock := &testutil.MockStore{GetReviewFn: reviewBy(authorID, "public", "x")}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	body := `{"target_kind":"review","target_id":"` + reviewID.String() + `","reason":"spam"}`
	var last *http.Response
	for i := 0; i < 21; i++ {
		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/reports", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+testutil.UserToken())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		last = resp
	}
	if last.StatusCode != http.StatusTooManyRequests || last.Header.Get("Retry-After") == "" {
		t.Errorf("21e signalement de l'heure : %d", last.StatusCode)
	}
}
