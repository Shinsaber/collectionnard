package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/testutil"
)

// world is an in-memory proposal store: statuses guarded like the SQL one.
type world struct {
	proposals map[uuid.UUID]*model.Proposal
	current   map[uuid.UUID]json.RawMessage // current attributes of catalog items
	dryRuns   int
	importErr bool
	applied   *model.CatalogExport
	actor     store.Actor
	waiting   int
}

func newWorld() (*world, *testutil.MockStore) {
	wd := &world{proposals: map[uuid.UUID]*model.Proposal{}, current: map[uuid.UUID]json.RawMessage{}}
	move := func(id uuid.UUID, from []string, to string) (*model.Proposal, error) {
		p, ok := wd.proposals[id]
		if !ok {
			return nil, store.ErrNotFound
		}
		if !slices.Contains(from, p.Status) {
			return nil, store.ErrConflict
		}
		p.Status = to
		return p, nil
	}
	mock := &testutil.MockStore{
		CreateProposalFn: func(_ context.Context, author uuid.UUID, in model.ProposalInput) (*model.Proposal, error) {
			p := &model.Proposal{ID: uuid.New(), Kind: in.Kind, Status: model.ProposalDraft, Title: in.Title,
				Payload: in.Payload, ItemID: in.ItemID, AuthorID: author, CreatedAt: time.Now()}
			wd.proposals[p.ID] = p
			return p, nil
		},
		GetProposalFn: func(_ context.Context, id uuid.UUID) (*model.Proposal, error) {
			if p, ok := wd.proposals[id]; ok {
				c := *p
				return &c, nil
			}
			return nil, store.ErrNotFound
		},
		SubmitProposalFn: func(_ context.Context, id uuid.UUID, base *model.ProposalBase) (*model.Proposal, error) {
			p, err := move(id, []string{model.ProposalDraft, model.ProposalChangesRequested}, model.ProposalPending)
			if err == nil {
				p.Base = base
			}
			return p, err
		},
		ReviewProposalFn: func(_ context.Context, id uuid.UUID, status string, _ uuid.UUID, _ string) (*model.Proposal, error) {
			return move(id, []string{model.ProposalPending}, status)
		},
		WithdrawProposalFn: func(_ context.Context, id uuid.UUID) (*model.Proposal, error) {
			return move(id, []string{model.ProposalDraft, model.ProposalPending, model.ProposalChangesRequested}, model.ProposalWithdrawn)
		},
		ImportCatalogFn: func(_ context.Context, _ *model.CatalogExport, dryRun bool) (*model.ImportReport, error) {
			wd.dryRuns++
			rep := model.NewImportReport(dryRun)
			if wd.importErr {
				rep.Errors = append(rep.Errors, model.ImportError{Entity: model.EntityItems, Message: "invalid"})
			}
			return rep, nil
		},
		EntitySnapshotFn: func(_ context.Context, items, _ []uuid.UUID) (*model.ProposalBase, error) {
			b := &model.ProposalBase{Items: map[uuid.UUID]json.RawMessage{}, Relations: map[uuid.UUID]json.RawMessage{}}
			for _, id := range items {
				if a, ok := wd.current[id]; ok {
					b.Items[id] = a
				}
			}
			return b, nil
		},
		AcceptCatalogProposalFn: func(ctx context.Context, id, _ uuid.UUID, payload *model.CatalogExport, _ string) (*model.ImportReport, error) {
			if _, err := move(id, []string{model.ProposalPending}, model.ProposalAccepted); err != nil {
				return nil, err
			}
			wd.applied, wd.actor = payload, store.ActorFrom(ctx)
			rep := model.NewImportReport(false)
			rep.Applied = true
			return rep, nil
		},
		ProposalQuotaUsageFn: func(_ context.Context, _ uuid.UUID) (int, int, int, error) {
			return wd.waiting, 0, 0, nil
		},
	}
	return wd, mock
}

func payloadFor(itemID uuid.UUID, attrs string) map[string]any {
	return map[string]any{"items": []map[string]any{{
		"id": itemID, "item_type_id": testutil.SampleItemType().ID, "attributes": json.RawMessage(attrs),
	}}}
}

func create(t *testing.T, srv *httptest.Server, token string, body map[string]any) (int, string, model.Proposal) {
	t.Helper()
	status, code, raw := callJSON(t, srv, http.MethodPost, "/api/v1/proposals", token, body)
	var p model.Proposal
	_ = json.Unmarshal(raw, &p)
	return status, code, p
}

func TestProposal_SubmitAndModerate(t *testing.T) {
	wd, mock := newWorld()
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	existing := uuid.New()
	wd.current[existing] = json.RawMessage(`{"titre":"Avant"}`)

	// Méta-types refusés ; brouillon puis soumission (simulation d'import, instantané de départ).
	bad := map[string]any{"kind": "catalog", "title": "x", "payload": map[string]any{"item_types": []map[string]any{{"id": uuid.New()}}}}
	if status, code, _ := create(t, srv, testutil.UserToken(), bad); status != http.StatusUnprocessableEntity || code != "proposal_meta_types" {
		t.Errorf("meta types: %d %s", status, code)
	}
	status, _, p := create(t, srv, testutil.UserToken(), map[string]any{
		"kind": "catalog", "title": "Corriger le titre", "payload": payloadFor(existing, `{"titre":"Après"}`),
	})
	if status != http.StatusCreated || p.Status != model.ProposalDraft {
		t.Fatalf("draft: %d %+v", status, p)
	}
	base := "/api/v1/proposals/" + p.ID.String()
	if status, _, _ := callJSON(t, srv, http.MethodGet, base, testutil.ContributorToken(), nil); status != http.StatusNotFound {
		t.Errorf("a moderator never sees a draft: %d", status)
	}
	if status, _, raw := callJSON(t, srv, http.MethodPost, base+"/submit", testutil.UserToken(), nil); status != http.StatusOK || wd.dryRuns != 1 {
		t.Fatalf("submit: %d %s", status, raw)
	}
	if b := wd.proposals[p.ID].Base; b == nil || string(b.Items[existing]) != `{"titre":"Avant"}` {
		t.Errorf("base: %+v", b)
	}

	// Droits de modération.
	if status, _, _ := callJSON(t, srv, http.MethodPost, base+"/accept", testutil.UserToken(), nil); status != http.StatusForbidden {
		t.Errorf("member accepts: %d", status)
	}
	if status, _, _ := callJSON(t, srv, http.MethodPost, base+"/reject", testutil.AdminToken(), map[string]string{"comment": " "}); status != http.StatusBadRequest {
		t.Errorf("reject without comment: %d", status)
	}

	// L'item a changé depuis : conflit, puis acceptation une fois tranché.
	wd.current[existing] = json.RawMessage(`{"titre":"Entre-temps"}`)
	status, code, raw := callJSON(t, srv, http.MethodPost, base+"/accept", testutil.ContributorToken(), map[string]any{})
	if status != http.StatusConflict || code != "proposal_conflict" {
		t.Fatalf("conflict: %d %s %s", status, code, raw)
	}
	status, _, raw = callJSON(t, srv, http.MethodPost, base+"/accept", testutil.ContributorToken(),
		map[string]any{"conflicts_resolved": true, "payload": payloadFor(existing, `{"titre":"Tranché"}`)})
	if status != http.StatusOK || wd.proposals[p.ID].Status != model.ProposalAccepted {
		t.Fatalf("accept: %d %s", status, raw)
	}
	if string(wd.applied.Items[0].Attributes) != `{"titre":"Tranché"}` {
		t.Errorf("moderator's correction not applied: %s", wd.applied.Items[0].Attributes)
	}
	if wd.actor.UserID != testutil.UserID || wd.actor.Origin != store.OriginProposal || wd.actor.Ref != p.ID.String() {
		t.Errorf("writes must be credited to the author: %+v", wd.actor)
	}
	if status, code, _ := callJSON(t, srv, http.MethodPost, base+"/accept", testutil.AdminToken(), nil); status != http.StatusConflict || code != "proposal_state" {
		t.Errorf("accept twice: %d %s", status, code)
	}
	if status, _, _ := callJSON(t, srv, http.MethodPost, base+"/comments", testutil.UserToken(), map[string]string{"body": "merci"}); status != http.StatusConflict {
		t.Errorf("comment on a decided proposal: %d", status)
	}
}

func TestProposal_OwnAndInvalidAndQuota(t *testing.T) {
	wd, mock := newWorld()
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	body := map[string]any{"kind": "catalog", "title": "Nouveau", "submit": true, "payload": payloadFor(uuid.New(), `{"titre":"X"}`)}

	// Un contributeur ne modère pas sa propre proposition.
	_, _, own := create(t, srv, testutil.ContributorToken(), body)
	if status, code, _ := callJSON(t, srv, http.MethodPost, "/api/v1/proposals/"+own.ID.String()+"/accept", testutil.ContributorToken(), nil); status != http.StatusForbidden || code != "own_proposal" {
		t.Errorf("own proposal: %d %s", status, code)
	}

	// Simulation d'import en échec : 422 avec le rapport, reste en brouillon.
	wd.importErr = true
	status, code, _ := create(t, srv, testutil.UserToken(), body)
	if status != http.StatusUnprocessableEntity || code != "proposal_invalid" {
		t.Errorf("invalid: %d %s", status, code)
	}
	wd.importErr = false

	// Quota atteint (5 en attente).
	wd.waiting = 5
	if status, code, _ := create(t, srv, testutil.UserToken(), body); status != http.StatusConflict || code != "proposal_quota" {
		t.Errorf("quota: %d %s", status, code)
	}
	// Contributeurs sans quota.
	if status, _, _ := create(t, srv, testutil.ContributorToken(), body); status != http.StatusCreated {
		t.Errorf("contributor quota: %d", status)
	}
}

func TestProposal_ChangesRequestedAndReports(t *testing.T) {
	wd, mock := newWorld()
	srv := testutil.NewTestServer(mock)
	defer srv.Close()

	if status, _, _ := create(t, srv, testutil.UserToken(), map[string]any{"kind": "report", "title": "Erreur", "description": "faux"}); status != http.StatusBadRequest {
		t.Errorf("report without item: %d", status)
	}
	_, _, rep := create(t, srv, testutil.UserToken(), map[string]any{
		"kind": "report", "title": "Erreur", "description": "Mauvais tome", "item_id": uuid.New(), "submit": true,
	})
	base := "/api/v1/proposals/" + rep.ID.String()
	if status, _, _ := callJSON(t, srv, http.MethodPost, base+"/request-changes", testutil.AdminToken(), map[string]string{"comment": "Lequel ?"}); status != http.StatusOK {
		t.Fatalf("request changes: %d", status)
	}
	if wd.proposals[rep.ID].Status != model.ProposalChangesRequested {
		t.Errorf("status: %s", wd.proposals[rep.ID].Status)
	}
	if status, _, _ := callJSON(t, srv, http.MethodPost, base+"/submit", testutil.UserToken(), nil); status != http.StatusOK {
		t.Errorf("resubmit: %d", status)
	}
	if status, _, _ := callJSON(t, srv, http.MethodPost, base+"/accept", testutil.AdminToken(), nil); status != http.StatusOK || wd.proposals[rep.ID].Status != model.ProposalAccepted {
		t.Errorf("report handled: %d", status)
	}
	if wd.dryRuns != 0 {
		t.Error("a report never imports")
	}
	if status, _, _ := callJSON(t, srv, http.MethodPost, base+"/withdraw", testutil.UserToken(), nil); status != http.StatusConflict {
		t.Errorf("withdraw after decision: %d", status)
	}
}

func TestFindDuplicates(t *testing.T) {
	typ := &model.ItemType{ID: uuid.New(), Name: "Volume", AttributesSchema: json.RawMessage(
		`{"type":"object","properties":{"isbn":{"type":"string","x-identifier":true},"title":{"type":"string"}}}`)}
	same := model.Item{ID: uuid.New(), ItemTypeID: typ.ID, Attributes: json.RawMessage(`{"isbn":"978"}`)}
	var asked []string
	mock := &testutil.MockStore{
		GetItemTypeFn: func(_ context.Context, _ uuid.UUID) (*model.ItemType, error) { return typ, nil },
		FindItemsByAttributeFn: func(_ context.Context, _ uuid.UUID, path, values []string, _ bool) ([]model.Item, error) {
			asked = append(asked, path...)
			if values[0] == "978" {
				return []model.Item{same}, nil
			}
			return nil, nil
		},
	}
	srv := testutil.NewTestServer(mock)
	defer srv.Close()
	status, _, raw := callJSON(t, srv, http.MethodPost, "/api/v1/items/duplicates", testutil.UserToken(),
		map[string]any{"item_type_id": typ.ID, "attributes": map[string]any{"isbn": "978", "title": "X"}})
	var got []model.Duplicate
	if status != http.StatusOK || json.Unmarshal(raw, &got) != nil || len(got) != 1 || got[0].Field != "isbn" || got[0].Item.ID != same.ID {
		t.Fatalf("duplicates: %d %s", status, raw)
	}
	if !slices.Equal(asked, []string{"isbn"}) {
		t.Errorf("only identifier fields are searched: %v", asked)
	}
	_, _, raw = callJSON(t, srv, http.MethodPost, "/api/v1/items/duplicates", testutil.UserToken(),
		map[string]any{"item_type_id": typ.ID, "attributes": map[string]any{"isbn": "978"}, "exclude_id": same.ID})
	if string(raw) != "[]\n" && string(raw) != "[]" {
		t.Errorf("the edited item is not its own duplicate: %s", raw)
	}
}
