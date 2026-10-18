//go:build integration

package store_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
)

// Les déclencheurs de la migration 010 historisent chaque écriture avec l'auteur et l'origine
// de la transaction ; une écriture sans changement n'ajoute rien ; la suppression est gardée.
func TestHistoryTriggers(t *testing.T) {
	st := freshStore(t, "colec_test_history")
	if _, err := st.InstallDomains(ctx, "books"); err != nil {
		t.Fatal(err)
	}
	contributor, err := st.CreateUser(ctx, "camille", "camille@example.org", "x", model.RoleContributor)
	if err != nil {
		t.Fatalf("contributor role refused by the database: %v", err)
	}
	actx := store.WithActor(ctx, store.Actor{UserID: contributor.ID, Origin: store.OriginDirect})
	volume := uuid.MustParse("b0000000-0000-0000-0000-000000000015")
	edition := uuid.MustParse("b0000000-0000-0000-0000-000000000014")
	containsVol := uuid.MustParse("c0000000-0000-0000-0000-000000000015")

	it, err := st.CreateItem(actx, model.ItemCreate{ItemTypeID: volume, Attributes: json.RawMessage(`{"number":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpdateItem(actx, it.ID, model.ItemUpdate{Attributes: json.RawMessage(`{"number":2}`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpdateItem(actx, it.ID, model.ItemUpdate{Attributes: json.RawMessage(`{"number":2}`)}); err != nil {
		t.Fatal(err)
	}
	ed, err := st.CreateItem(ctx, model.ItemCreate{ItemTypeID: edition, Attributes: json.RawMessage(`{"title":"Glénat"}`)})
	if err != nil {
		t.Fatal(err)
	}
	rel, err := st.CreateRelation(store.WithOrigin(actx, store.OriginExternal, "bd-manga"),
		model.ItemRelationCreate{RelationTypeID: containsVol, SourceItemID: ed.ID, TargetItemID: it.ID})
	if err != nil {
		t.Fatal(err)
	}

	h, err := st.GetItemHistory(ctx, it.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Entries) != 3 {
		t.Fatalf("entries (unchanged update must add nothing): %+v", h.Entries)
	}
	if e := h.Entries[0]; e.Kind != "relation" || e.Origin != store.OriginExternal || *e.OriginRef != "bd-manga" || e.Relation.SourceItemID != ed.ID {
		t.Errorf("relation entry: %+v", e)
	}
	if e := h.Entries[1]; e.Version != 2 || string(e.Before) != `{"number": 1}` || string(e.After) != `{"number": 2}` || *e.UserID != contributor.ID {
		t.Errorf("update entry: %+v %s %s", e, e.Before, e.After)
	}
	if h.Created == nil || h.Created.Version != 1 || string(h.Created.Before) != "null" {
		t.Errorf("created: %+v", h.Created)
	}
	if eh, _ := st.GetItemHistory(ctx, ed.ID); eh.Created == nil || eh.Created.Origin != "system" || eh.Created.UserID != nil {
		t.Errorf("write without actor must be `system`: %+v", eh.Created)
	}

	// Retour à la version 1, historisé `revert`.
	attrs, err := st.GetItemVersion(ctx, it.ID, 1)
	if err != nil || string(attrs) != `{"number": 1}` {
		t.Fatalf("version 1: %v %s", err, attrs)
	}
	if _, err := st.UpdateItem(store.WithOrigin(actx, store.OriginRevert, "1"), it.ID, model.ItemUpdate{Attributes: attrs}); err != nil {
		t.Fatal(err)
	}
	if h, _ = st.GetItemHistory(ctx, it.ID); h.Entries[0].Origin != store.OriginRevert || h.Entries[0].Version != 3 {
		t.Errorf("revert entry: %+v", h.Entries[0])
	}

	// Suppression : l'historique reste ; la version de suppression ne se restaure pas.
	if err := st.DeleteRelation(actx, rel.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteItem(actx, it.ID); err != nil {
		t.Fatal(err)
	}
	h, _ = st.GetItemHistory(ctx, it.ID)
	if h.Entries[0].Kind != "item" || string(h.Entries[0].After) != "null" || h.Entries[1].Kind != "relation" {
		t.Errorf("deletions: %+v", h.Entries[:2])
	}
	if _, err := st.GetItemVersion(ctx, it.ID, h.Entries[0].Version); !errors.Is(err, store.ErrDeletedVersion) {
		t.Errorf("deletion version: %v", err)
	}
	if _, err := st.GetItemVersion(ctx, it.ID, 99); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown version: %v", err)
	}
}
