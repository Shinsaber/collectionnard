package handler_test

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/testutil"
)

type sent struct {
	to   []uuid.UUID
	kind string
}

func TestNotifications_OnDecisionAndComment(t *testing.T) {
	_, mock := newWorld()
	var got []sent
	mock.CreateNotificationsFn = func(_ context.Context, to []uuid.UUID, kind string, _ any) error {
		got = append(got, sent{to, kind})
		return nil
	}
	moderator := testutil.AdminID
	mock.ListProposalCommentsFn = func(_ context.Context, _ uuid.UUID) ([]model.ProposalComment, error) {
		return []model.ProposalComment{{AuthorID: moderator}, {AuthorID: testutil.ContributorID}}, nil
	}
	mock.CreateProposalCommentFn = func(_ context.Context, id, author uuid.UUID, body string) (*model.ProposalComment, error) {
		return &model.ProposalComment{ID: uuid.New(), ProposalID: id, AuthorID: author, Body: body}, nil
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	_, _, p := create(t, srv, testutil.UserToken(), map[string]any{
		"kind": "schema_suggestion", "title": "Champ", "description": "illustrateur", "submit": true,
	})
	base := "/api/v1/proposals/" + p.ID.String()

	// Message de l'auteur : les participants sont prévenus, pas l'auteur lui-même.
	if status, _, raw := callJSON(t, srv, http.MethodPost, base+"/comments", testutil.UserToken(), map[string]string{"body": "Merci"}); status != http.StatusCreated {
		t.Fatalf("comment: %d %s", status, raw)
	}
	if len(got) != 1 || got[0].kind != model.NotifyProposalComment || slices.Contains(got[0].to, testutil.UserID) ||
		!slices.Contains(got[0].to, moderator) || !slices.Contains(got[0].to, testutil.ContributorID) {
		t.Errorf("comment notification: %+v", got)
	}

	// Décision : l'auteur est prévenu.
	got = nil
	if status, _, _ := callJSON(t, srv, http.MethodPost, base+"/request-changes", testutil.AdminToken(), map[string]string{"comment": "Précise"}); status != http.StatusOK {
		t.Fatalf("request changes: %d", status)
	}
	if len(got) != 1 || got[0].kind != model.NotifyProposalChangesRequested || !slices.Equal(got[0].to, []uuid.UUID{testutil.UserID}) {
		t.Errorf("decision notification: %+v", got)
	}
}

func TestNotifications_ListAndRead(t *testing.T) {
	var marked []uuid.UUID
	var markedFor uuid.UUID
	mock := &testutil.MockStore{
		ListNotificationsFn: func(_ context.Context, user uuid.UUID, _ model.Pagination) ([]model.Notification, int, int, error) {
			return []model.Notification{{ID: uuid.New(), Kind: model.NotifyProposalAccepted}}, 3, 2, nil
		},
		MarkNotificationsReadFn: func(_ context.Context, user uuid.UUID, ids []uuid.UUID) error {
			markedFor, marked = user, ids
			return nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	status, _, raw := callJSON(t, srv, http.MethodGet, "/api/v1/me/notifications", testutil.UserToken(), nil)
	if status != http.StatusOK || !strings.Contains(string(raw), `"unread":2`) || !strings.Contains(string(raw), `"total":3`) {
		t.Errorf("list: %d %s", status, raw)
	}
	id := uuid.New()
	if status, _, _ := callJSON(t, srv, http.MethodPost, "/api/v1/me/notifications/read", testutil.UserToken(), map[string]any{"ids": []uuid.UUID{id}}); status != http.StatusNoContent {
		t.Errorf("read: %d", status)
	}
	if markedFor != testutil.UserID || !slices.Equal(marked, []uuid.UUID{id}) {
		t.Errorf("marked %v for %v", marked, markedFor)
	}
	if status, _, _ := callJSON(t, srv, http.MethodPost, "/api/v1/me/notifications/read", testutil.UserToken(), nil); status != http.StatusNoContent || marked != nil {
		t.Errorf("read all: %d %v", status, marked)
	}
}
