package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

var (
	reviewItemID = uuid.MustParse("e0000000-0000-0000-0000-000000000001")
	authorID     = uuid.MustParse("a0000000-0000-0000-0000-0000000000aa")
)

func comicReviewType() *model.ReviewType {
	return &model.ReviewType{
		ID:   uuid.MustParse("f1000000-0000-0000-0000-000000000001"),
		Name: "ComicReview",
		AttributesSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"x-order":["story","art"],"properties":{
			"story":{"type":"integer","minimum":0,"maximum":10},"art":{"type":"integer","minimum":0,"maximum":10}}}`),
	}
}

func callReview(t *testing.T, mock *testutil.MockStore, method, path, token, body string) (*http.Response, string) {
	t.Helper()
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	var rd io.Reader
	if body != "" {
		rd = bytes.NewBufferString(body)
	}
	req, _ := http.NewRequest(method, srv.URL+"/api/v1"+path, rd)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw)
}

func openReviews(mock *testutil.MockStore) *testutil.MockStore {
	mock.GetItemReviewTypeFn = func(context.Context, uuid.UUID) (*model.ReviewType, error) {
		return comicReviewType(), nil
	}
	return mock
}

func TestPutReview_Rating(t *testing.T) {
	var got model.ReviewWrite
	mock := openReviews(&testutil.MockStore{
		PutReviewFn: func(_ context.Context, itemID, userID uuid.UUID, in model.ReviewWrite) (*model.Review, error) {
			got = in
			if userID != testutil.UserID || itemID != reviewItemID {
				t.Errorf("mauvais couple : %s %s", userID, itemID)
			}
			return &model.Review{ID: uuid.New(), ItemID: itemID, Rating: in.Rating, Visibility: in.Visibility, Lang: in.Lang}, nil
		},
	})
	path := "/items/" + reviewItemID.String() + "/reviews/mine"

	resp, body := callReview(t, mock, "PUT", path, testutil.UserToken(),
		`{"rating":8,"body":"  Très bien  ","spoiler":true,"lang":"fr","visibility":"public"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("note globale : %d %s", resp.StatusCode, body)
	}
	if got.Rating != 8 || got.Criteria != nil || got.ReviewTypeID != nil || got.Body != "Très bien" || !got.Spoiler {
		t.Errorf("écriture : %+v", got)
	}

	resp, body = callReview(t, mock, "PUT", path, testutil.UserToken(),
		`{"criteria":{"story":8,"art":7},"lang":"en","visibility":"private"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("par critère : %d %s", resp.StatusCode, body)
	}
	if got.Rating != 7.5 || got.ReviewTypeID == nil || got.Lang != "en" {
		t.Errorf("la note est la moyenne calculée par le serveur : %+v", got)
	}
}

func TestPutReview_Invalid(t *testing.T) {
	mock := openReviews(&testutil.MockStore{
		PutReviewFn: func(context.Context, uuid.UUID, uuid.UUID, model.ReviewWrite) (*model.Review, error) {
			t.Error("écriture d'un avis invalide")
			return nil, store.ErrNotFound
		},
	})
	path := "/items/" + reviewItemID.String() + "/reviews/mine"
	long := strings.Repeat("a", 10001)
	for name, body := range map[string]string{
		"sans note":        `{"body":"texte","lang":"fr","visibility":"public"}`,
		"les deux modes":   `{"rating":8,"criteria":{"story":8,"art":7},"lang":"fr","visibility":"public"}`,
		"hors échelle":     `{"rating":11,"lang":"fr","visibility":"public"}`,
		"critère manquant": `{"criteria":{"story":8},"lang":"fr","visibility":"public"}`,
		"critère inconnu":  `{"criteria":{"story":8,"art":7,"son":1},"lang":"fr","visibility":"public"}`,
		"langue inconnue":  `{"rating":8,"lang":"es","visibility":"public"}`,
		"visibilité":       `{"rating":8,"lang":"fr","visibility":"rating_only"}`,
		"texte trop long":  `{"rating":8,"lang":"fr","visibility":"public","body":"` + long + `"}`,
	} {
		if resp, raw := callReview(t, mock, "PUT", path, testutil.UserToken(), body); resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("%s : %d %s", name, resp.StatusCode, raw)
		}
	}
}

func TestPutReview_Access(t *testing.T) {
	path := "/items/" + reviewItemID.String() + "/reviews/mine"
	body := `{"rating":8,"lang":"fr","visibility":"public"}`

	if resp, _ := callReview(t, openReviews(&testutil.MockStore{}), "PUT", path, "", body); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonyme : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, &testutil.MockStore{}, "PUT", path, testutil.UserToken(), body); resp.StatusCode != http.StatusNotFound {
		t.Errorf("item inconnu : %d", resp.StatusCode)
	}
	closed := &testutil.MockStore{GetItemReviewTypeFn: func(context.Context, uuid.UUID) (*model.ReviewType, error) { return nil, nil }}
	resp, raw := callReview(t, closed, "PUT", path, testutil.UserToken(), body)
	if resp.StatusCode != http.StatusConflict || !strings.Contains(raw, "reviews_closed") {
		t.Errorf("avis fermés sur ce type : %d %s", resp.StatusCode, raw)
	}
}

// Le JSON brut ne doit jamais porter l'auteur d'un avis qui n'est pas signé pour le lecteur.
func TestListItemReviews_HidesAuthor(t *testing.T) {
	var levels []string
	mock := openReviews(&testutil.MockStore{
		ListItemReviewsFn: func(_ context.Context, _ uuid.UUID, p model.ReviewListParams, _ model.Pagination) ([]model.Review, int, error) {
			levels = p.Levels
			author := func() *model.ReviewAuthor { return &model.ReviewAuthor{Username: "camille"} }
			return []model.Review{
				{ID: uuid.New(), UserID: authorID, Body: "signé pour tous", Visibility: "public", Author: author(), CreatedAt: time.Now()},
				{ID: uuid.New(), UserID: authorID, Body: "pour les membres", Visibility: "members", Author: author(), CreatedAt: time.Now()},
				{ID: uuid.New(), UserID: authorID, Body: "sans nom", Visibility: "anonymous", Author: author(), CreatedAt: time.Now()},
			}, 3, nil
		},
	})
	path := "/items/" + reviewItemID.String() + "/reviews"

	authors := func(token string) map[string]bool {
		resp, raw := callReview(t, mock, "GET", path, token, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%d %s", resp.StatusCode, raw)
		}
		if strings.Contains(raw, authorID.String()) {
			t.Errorf("identifiant de l'auteur dans la réponse : %s", raw)
		}
		var page struct {
			Data []model.Review `json:"data"`
		}
		_ = json.Unmarshal([]byte(raw), &page)
		out := map[string]bool{}
		for _, r := range page.Data {
			out[r.Visibility] = r.Author != nil
		}
		return out
	}

	got := authors("")
	if !got["public"] || got["members"] || got["anonymous"] {
		t.Errorf("visiteur : %v", got)
	}
	if len(levels) != 1 || levels[0] != "public" {
		t.Errorf("niveaux signés pour un visiteur : %v", levels)
	}
	got = authors(testutil.UserToken())
	if !got["public"] || !got["members"] || got["anonymous"] {
		t.Errorf("membre : %v", got)
	}
	got = authors(testutil.AdminToken())
	if got["anonymous"] {
		t.Error("un admin est un lecteur comme un autre : l'anonymat tient")
	}
}

func TestListItemReviews_BadParams(t *testing.T) {
	path := "/items/" + reviewItemID.String() + "/reviews"
	mock := openReviews(&testutil.MockStore{})
	for _, q := range []string{"?sort=useful", "?lang=es"} {
		if resp, _ := callReview(t, mock, "GET", path+q, "", ""); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s : %d", q, resp.StatusCode)
		}
	}
	if resp, _ := callReview(t, &testutil.MockStore{}, "GET", path, "", ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("item inconnu : %d", resp.StatusCode)
	}
}

func TestReviewSummary_Threshold(t *testing.T) {
	path := "/items/" + reviewItemID.String() + "/reviews/summary"
	stats := func(count, art int) *testutil.MockStore {
		return openReviews(&testutil.MockStore{
			GetReviewStatsFn: func(context.Context, uuid.UUID, uuid.UUID, []string) (*model.ReviewStats, error) {
				return &model.ReviewStats{Count: count, Avg: 7.26, NoteOnly: 1, Criteria: map[string]model.ReviewCriterionStats{
					"story":   {Count: count, Avg: 8},
					"art":     {Count: art, Avg: 6},
					"removed": {Count: count, Avg: 1},
				}}, nil
			},
		})
	}
	read := func(mock *testutil.MockStore) model.ReviewSummary {
		resp, raw := callReview(t, mock, "GET", path, "", "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%d %s", resp.StatusCode, raw)
		}
		var s model.ReviewSummary
		if err := json.Unmarshal([]byte(raw), &s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	s := read(stats(4, 4))
	if s.Rating != nil || s.Count != 4 || s.MinRatings != 5 || s.Criteria[0].Rating != nil {
		t.Errorf("sous le seuil, seul le nombre sort : %+v", s)
	}

	s = read(stats(5, 2))
	if s.Rating == nil || *s.Rating != 7.3 {
		t.Errorf("moyenne au seuil : %+v", s.Rating)
	}
	if len(s.Criteria) != 2 || s.Criteria[0].Key != "story" || s.Criteria[0].Rating == nil || s.Criteria[1].Rating != nil {
		t.Errorf("critères (ordre du schéma, seuil par critère, critère retiré ignoré) : %+v", s.Criteria)
	}

	closed := &testutil.MockStore{GetItemReviewTypeFn: func(context.Context, uuid.UUID) (*model.ReviewType, error) { return nil, nil }}
	if s = read(closed); s.Enabled || s.ReviewType != nil {
		t.Errorf("avis fermés : %+v", s)
	}
}

func TestListPublicReviews_SignedOnly(t *testing.T) {
	var levels []string
	mock := &testutil.MockStore{
		GetUserByUsernameFn: func(context.Context, string) (*model.User, string, error) {
			return &model.User{ID: authorID, Username: "camille", ProfileVisibility: "public"}, "", nil
		},
		ListUserReviewsFn: func(_ context.Context, _ uuid.UUID, l []string, _ model.Pagination) ([]model.Review, int, error) {
			levels = l
			return nil, 0, nil
		},
	}
	if resp, raw := callReview(t, mock, "GET", "/public/users/camille/reviews", "", ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	if len(levels) != 1 || levels[0] != "public" {
		t.Errorf("un visiteur ne voit que le signé public : %v", levels)
	}
	if resp, _ := callReview(t, mock, "GET", "/public/users/camille/reviews", testutil.UserToken(), ""); resp.StatusCode != http.StatusOK {
		t.Fatal(resp.StatusCode)
	}
	if len(levels) != 2 {
		t.Errorf("un membre voit aussi le signé membres : %v", levels)
	}
	mock.GetUserByUsernameFn = func(context.Context, string) (*model.User, string, error) {
		return &model.User{ID: authorID, Username: "camille", ProfileVisibility: "private"}, "", nil
	}
	if resp, _ := callReview(t, mock, "GET", "/public/users/camille/reviews", "", ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("profil privé : %d", resp.StatusCode)
	}
}

func TestReviewType_Admin(t *testing.T) {
	created := 0
	mock := &testutil.MockStore{
		CreateReviewTypeFn: func(_ context.Context, in model.ReviewTypeCreate) (*model.ReviewType, error) {
			created++
			return &model.ReviewType{ID: uuid.New(), Name: in.Name, AttributesSchema: in.AttributesSchema}, nil
		},
	}
	good := `{"name":"X","attributes_schema":{"type":"object","properties":{"a":{"type":"integer","minimum":0,"maximum":10}}}}`
	bad := `{"name":"X","attributes_schema":{"type":"object","properties":{"a":{"type":"string"}}}}`

	if resp, _ := callReview(t, mock, "POST", "/review-types", "", good); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonyme : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "POST", "/review-types", testutil.UserToken(), good); resp.StatusCode != http.StatusForbidden {
		t.Errorf("membre : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "POST", "/review-types", testutil.ContributorToken(), good); resp.StatusCode != http.StatusForbidden {
		t.Errorf("contributeur : %d", resp.StatusCode)
	}
	if resp, _ := callReview(t, mock, "POST", "/review-types", testutil.AdminToken(), bad); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("critère qui n'est pas un entier de 0 à 10 : %d", resp.StatusCode)
	}
	if resp, raw := callReview(t, mock, "POST", "/review-types", testutil.AdminToken(), good); resp.StatusCode != http.StatusCreated {
		t.Errorf("admin : %d %s", resp.StatusCode, raw)
	}
	if created != 1 {
		t.Errorf("créations : %d", created)
	}
}

func TestItemType_ReviewTypeTriState(t *testing.T) {
	var got model.ItemTypeUpdate
	mock := &testutil.MockStore{
		UpdateItemTypeFn: func(_ context.Context, _ uuid.UUID, in model.ItemTypeUpdate) (*model.ItemType, error) {
			got = in
			return testutil.SampleItemType(), nil
		},
	}
	path := "/item-types/b0000000-0000-0000-0000-000000000001"
	rt := "f1000000-0000-0000-0000-000000000001"

	callReview(t, mock, "PUT", path, testutil.AdminToken(), `{"name":"X"}`)
	if got.ReviewTypeID.Set {
		t.Error("absent : le type est gardé")
	}
	callReview(t, mock, "PUT", path, testutil.AdminToken(), `{"review_type_id":null}`)
	if !got.ReviewTypeID.Set || got.ReviewTypeID.Value != nil {
		t.Errorf("null ferme les avis : %+v", got.ReviewTypeID)
	}
	callReview(t, mock, "PUT", path, testutil.AdminToken(), `{"review_type_id":"`+rt+`"}`)
	if !got.ReviewTypeID.Set || got.ReviewTypeID.Value == nil || got.ReviewTypeID.Value.String() != rt {
		t.Errorf("identifiant : %+v", got.ReviewTypeID)
	}
}

func TestPutReview_Context(t *testing.T) {
	edition := uuid.MustParse("e0000000-0000-0000-0000-0000000000e1")
	var got model.ReviewWrite
	mock := openReviews(&testutil.MockStore{
		IsReviewContextFn: func(_ context.Context, _, _, contextID uuid.UUID) (bool, error) {
			return contextID == edition, nil
		},
		PutReviewFn: func(_ context.Context, _, _ uuid.UUID, in model.ReviewWrite) (*model.Review, error) {
			got = in
			return &model.Review{ID: uuid.New()}, nil
		},
	})
	path := "/items/" + reviewItemID.String() + "/reviews/mine"
	body := func(ctx string) string {
		return `{"rating":7,"lang":"fr","visibility":"public","context_item_id":"` + ctx + `"}`
	}
	if resp, raw := callReview(t, mock, "PUT", path, testutil.UserToken(), body(edition.String())); resp.StatusCode != http.StatusOK {
		t.Fatalf("édition liée : %d %s", resp.StatusCode, raw)
	}
	if got.ContextItemID == nil || *got.ContextItemID != edition {
		t.Errorf("édition enregistrée : %+v", got.ContextItemID)
	}
	resp, raw := callReview(t, mock, "PUT", path, testutil.UserToken(), body(uuid.NewString()))
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(raw, "context_item_id") {
		t.Errorf("édition sans lien avec l'œuvre : %d %s", resp.StatusCode, raw)
	}
}

// L'édition utilisée suit l'auteur : rien sur un avis non signé pour le lecteur.
func TestListItemReviews_HidesContextWithAuthor(t *testing.T) {
	edition := uuid.MustParse("e0000000-0000-0000-0000-0000000000e1")
	var filter *uuid.UUID
	mock := openReviews(&testutil.MockStore{
		ListItemReviewsFn: func(_ context.Context, _ uuid.UUID, p model.ReviewListParams, _ model.Pagination) ([]model.Review, int, error) {
			filter = p.ContextItemID
			ctx := func() (*uuid.UUID, *model.Item) {
				id := edition
				return &id, &model.Item{ID: id, Attributes: json.RawMessage(`{"title":"Collector numéroté"}`)}
			}
			a, ai := ctx()
			b, bi := ctx()
			return []model.Review{
				{ID: uuid.New(), UserID: authorID, Visibility: "public", ContextItemID: a, ContextItem: ai, Author: &model.ReviewAuthor{Username: "camille"}},
				{ID: uuid.New(), UserID: authorID, Visibility: "anonymous", Body: "x", ContextItemID: b, ContextItem: bi, Author: &model.ReviewAuthor{Username: "camille"}},
			}, 2, nil
		},
	})
	resp, raw := callReview(t, mock, "GET", "/items/"+reviewItemID.String()+"/reviews?context_item_id="+edition.String(), "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	if filter == nil || *filter != edition {
		t.Errorf("filtre : %v", filter)
	}
	if strings.Count(raw, "Collector numéroté") != 1 {
		t.Errorf("l'édition d'un avis anonyme ne sort pas : %s", raw)
	}
	if resp, _ := callReview(t, mock, "GET", "/items/"+reviewItemID.String()+"/reviews?context_item_id=x", "", ""); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("filtre invalide : %d", resp.StatusCode)
	}
}

func TestReviewType_ContextLabel(t *testing.T) {
	mock := &testutil.MockStore{
		CreateReviewTypeFn: func(_ context.Context, in model.ReviewTypeCreate) (*model.ReviewType, error) {
			return &model.ReviewType{ID: uuid.New(), Name: in.Name, ContextLabel: in.ContextLabel}, nil
		},
	}
	if resp, _ := callReview(t, mock, "POST", "/review-types", testutil.AdminToken(),
		`{"name":"X","context_label":{"es":"Jugado en"}}`); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("langue hors interface : %d", resp.StatusCode)
	}
	if resp, raw := callReview(t, mock, "POST", "/review-types", testutil.AdminToken(),
		`{"name":"X","context_label":{"fr":"Joué sur","en":"Played on"}}`); resp.StatusCode != http.StatusCreated || !strings.Contains(raw, "Joué sur") {
		t.Errorf("libellé : %d %s", resp.StatusCode, raw)
	}
}
