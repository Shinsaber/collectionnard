//go:build integration

package store_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

func TestProposalLifecycle(t *testing.T) {
	st := freshStore(t, "colec_test_proposals")
	if _, err := st.InstallDomains(ctx, "books"); err != nil {
		t.Fatal(err)
	}
	author, _ := st.CreateUser(ctx, "camille", "c@example.org", "x", model.RoleUser)
	mod, _ := st.CreateUser(ctx, "dominique", "d@example.org", "x", model.RoleContributor)
	volume := uuid.MustParse("b0000000-0000-0000-0000-000000000015")
	existing, err := st.CreateItem(ctx, model.ItemCreate{ItemTypeID: volume, Attributes: json.RawMessage(`{"number":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	newID := uuid.New()
	payload := &model.CatalogExport{Format: model.CatalogFormat, Version: model.ExportVersion, Items: []model.ExportItem{
		{ID: existing.ID, ItemTypeID: volume, Attributes: json.RawMessage(`{"number":2,"title":{"fr":"T2"}}`)},
		{ID: newID, ItemTypeID: volume, Attributes: json.RawMessage(`{"number":3,"title":{"fr":"T3"},"images":{"front":["/api/v1/uploads/cover.jpg"]}}`)},
	}}

	p, err := st.CreateProposal(ctx, author.ID, model.ProposalInput{Kind: model.ProposalCatalog, Title: "Tomes", Payload: payload})
	if err != nil || p.Status != model.ProposalDraft || p.Author.Username != "camille" || p.Payload == nil {
		t.Fatalf("create: %v %+v", err, p)
	}
	if refs, _ := st.ReferencedUploads(ctx); !slices.Contains(refs, "cover.jpg") {
		t.Errorf("image of a pending proposal must be kept by the purge: %v", refs)
	}

	base, err := st.EntitySnapshot(ctx, []uuid.UUID{existing.ID, newID}, nil)
	if err != nil || len(base.Items) != 1 || string(base.Items[existing.ID]) != `{"number": 1}` {
		t.Fatalf("snapshot: %v %+v", err, base)
	}
	if p, err = st.SubmitProposal(ctx, p.ID, base); err != nil || p.Status != model.ProposalPending || p.Base == nil {
		t.Fatalf("submit: %v %+v", err, p)
	}
	if _, err := st.SubmitProposal(ctx, p.ID, base); !errors.Is(err, store.ErrConflict) {
		t.Errorf("submit twice: %v", err)
	}
	if n, _ := st.CountProposalsToReview(ctx, mod.ID); n != 1 {
		t.Errorf("to review: %d", n)
	}
	if n, _ := st.CountProposalsToReview(ctx, author.ID); n != 0 {
		t.Errorf("one's own proposal is not to review: %d", n)
	}
	list, total, _ := st.ListProposals(ctx, model.ProposalFilter{ExcludeAuthor: &mod.ID, Review: true}, model.Pagination{Page: 1, PerPage: 20})
	if total != 1 || list[0].ID != p.ID {
		t.Errorf("review queue: %d %+v", total, list)
	}

	if _, err := st.CreateProposalComment(ctx, p.ID, mod.ID, "Merci !"); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.GetProposal(ctx, p.ID); got.CommentCount != 1 {
		t.Errorf("comment count: %d", got.CommentCount)
	}

	// Acceptation : import et statut ensemble, écritures créditées à l'auteur.
	actx := store.WithActor(ctx, store.Actor{UserID: author.ID, Origin: store.OriginProposal, Ref: p.ID.String()})
	rep, err := st.AcceptCatalogProposal(actx, p.ID, mod.ID, payload, "Bien vu")
	if err != nil || !rep.Applied {
		t.Fatalf("accept: %v %+v", err, rep)
	}
	got, _ := st.GetProposal(ctx, p.ID)
	if got.Status != model.ProposalAccepted || got.Reviewer == nil || got.Reviewer.Username != "dominique" || got.DecidedAt == nil {
		t.Errorf("accepted: %+v", got)
	}
	h, _ := st.GetItemHistory(ctx, existing.ID)
	if e := h.Entries[0]; e.Origin != store.OriginProposal || *e.UserID != author.ID || *e.OriginRef != p.ID.String() {
		t.Errorf("history of the accepted write: %+v", e)
	}
	if _, err := st.AcceptCatalogProposal(actx, p.ID, mod.ID, payload, ""); !errors.Is(err, store.ErrConflict) {
		t.Errorf("accept twice must roll back: %v", err)
	}
	if refs, _ := st.ReferencedUploads(ctx); !slices.Contains(refs, "cover.jpg") {
		t.Error("the image is now referenced by the created item")
	}

	// Quota : en attente, acceptées.
	waiting, accepted, today, err := st.ProposalQuotaUsage(ctx, author.ID)
	if err != nil || waiting != 0 || accepted != 1 || today != 1 {
		t.Errorf("quota usage: %d %d %d %v", waiting, accepted, today, err)
	}

	// Refus puis retrait impossible ; un brouillon se retire.
	r, _ := st.CreateProposal(ctx, author.ID, model.ProposalInput{Kind: model.ProposalSchemaSuggestion, Title: "Champ", Description: "illustrateur"})
	if _, err := st.ReviewProposal(ctx, r.ID, model.ProposalRejected, mod.ID, "non"); !errors.Is(err, store.ErrConflict) {
		t.Errorf("reject a draft: %v", err)
	}
	if r, err = st.WithdrawProposal(ctx, r.ID); err != nil || r.Status != model.ProposalWithdrawn {
		t.Errorf("withdraw: %v %+v", err, r)
	}
}

func TestNotificationsStore(t *testing.T) {
	st := freshStore(t, "colec_test_notifications")
	a, _ := st.CreateUser(ctx, "camille", "c@example.org", "x", model.RoleUser)
	b, _ := st.CreateUser(ctx, "dominique", "d@example.org", "x", model.RoleUser)
	if err := st.CreateNotifications(ctx, []uuid.UUID{a.ID, a.ID, b.ID}, model.NotifyProposalAccepted, map[string]any{"title": "T"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateNotifications(ctx, []uuid.UUID{a.ID}, model.NotifyProposalComment, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	list, total, unread, err := st.ListNotifications(ctx, a.ID, model.Pagination{Page: 1, PerPage: 20})
	if err != nil || total != 2 || unread != 2 || list[0].Kind != model.NotifyProposalComment {
		t.Fatalf("list (duplicates ignored, newest first): %v %d %d %+v", err, total, unread, list)
	}
	if err := st.MarkNotificationsRead(ctx, a.ID, []uuid.UUID{list[1].ID}); err != nil {
		t.Fatal(err)
	}
	if _, _, unread, _ = st.ListNotifications(ctx, a.ID, model.Pagination{Page: 1, PerPage: 20}); unread != 1 {
		t.Errorf("one read: %d unread", unread)
	}
	// Marquer « tout » ne touche pas aux notifications d'un autre compte.
	if err := st.MarkNotificationsRead(ctx, a.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, _, unread, _ = st.ListNotifications(ctx, b.ID, model.Pagination{Page: 1, PerPage: 20}); unread != 1 {
		t.Errorf("other account: %d unread", unread)
	}
}
