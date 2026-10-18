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

// Visibilité, champs cachés (absent / null / liste), liens de partage, annuaire et
// statistiques sans les champs cachés.
func TestVisibilityAndSharing(t *testing.T) {
	st := freshStore(t, "colec_test_sharing")
	if _, err := st.InstallDomains(ctx, "collections", "books"); err != nil {
		t.Fatal(err)
	}
	owner, err := st.CreateUser(ctx, "camille", "camille@example.org", "x", model.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if owner.ProfileVisibility != "private" {
		t.Errorf("default profile visibility: %q", owner.ProfileVisibility)
	}
	ownership := uuid.MustParse("d0000000-0000-0000-0000-000000000001")
	col, err := st.CreateCollection(ctx, owner.ID, model.CollectionCreate{CollectionTypeID: ownership, Name: "Mangas"})
	if err != nil {
		t.Fatal(err)
	}
	if col.Visibility != "private" || col.HiddenFields != nil {
		t.Errorf("defaults: %q %v", col.Visibility, col.HiddenFields)
	}

	// Liste, puis absent (inchangé), puis null (retour au défaut).
	vis := "public"
	col, err = st.UpdateCollection(ctx, col.ID, model.CollectionUpdate{Visibility: &vis,
		HiddenFields: model.OptionalStrings{Set: true, Value: []string{"condition"}}})
	if err != nil || col.Visibility != "public" || !slices.Equal(col.HiddenFields, []string{"condition"}) {
		t.Fatalf("set: %v %+v", err, col)
	}
	name := "Mes mangas"
	if col, _ = st.UpdateCollection(ctx, col.ID, model.CollectionUpdate{Name: &name}); !slices.Equal(col.HiddenFields, []string{"condition"}) {
		t.Errorf("absent changed the list: %v", col.HiddenFields)
	}
	if col, _ = st.UpdateCollection(ctx, col.ID, model.CollectionUpdate{HiddenFields: model.OptionalStrings{Set: true, Value: []string{}}}); col.HiddenFields == nil || len(col.HiddenFields) != 0 {
		t.Errorf("empty list must stay empty (show everything), got %v", col.HiddenFields)
	}
	if col, _ = st.UpdateCollection(ctx, col.ID, model.CollectionUpdate{HiddenFields: model.OptionalStrings{Set: true}}); col.HiddenFields != nil {
		t.Errorf("null must reset to the default, got %v", col.HiddenFields)
	}

	// Collections visibles.
	priv, err := st.CreateCollection(ctx, owner.ID, model.CollectionCreate{CollectionTypeID: ownership, Name: "Secrète"})
	if err != nil {
		t.Fatal(err)
	}
	visible, err := st.ListVisibleCollections(ctx, owner.ID, []string{"members", "public"})
	if err != nil || len(visible) != 1 || visible[0].ID != col.ID {
		t.Errorf("visible: %v %+v", err, visible)
	}

	// Liens de partage.
	link, err := st.CreateShareLink(ctx, priv.ID, "share-hash")
	if err != nil {
		t.Fatal(err)
	}
	if id, err := st.GetSharedCollectionID(ctx, "share-hash"); err != nil || id != priv.ID {
		t.Errorf("resolve: %v %v", err, id)
	}
	if links, _ := st.ListShareLinks(ctx, priv.ID); len(links) != 1 {
		t.Errorf("links: %+v", links)
	}
	if err := st.RevokeShareLink(ctx, col.ID, link.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("revoke through another collection: %v", err)
	}
	if err := st.RevokeShareLink(ctx, priv.ID, link.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetSharedCollectionID(ctx, "share-hash"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("revoked link still resolves: %v", err)
	}
	if links, _ := st.ListShareLinks(ctx, priv.ID); len(links) != 0 {
		t.Errorf("revoked link still listed: %+v", links)
	}

	// Annuaire : profil privé absent, puis présent une fois ouvert aux membres.
	if members, total, _ := st.ListMembers(ctx, model.Pagination{Page: 1, PerPage: 20}); total != 0 || len(members) != 0 {
		t.Errorf("private profile listed: %+v", members)
	}
	u, err := st.UpdateProfile(ctx, owner.ID, model.ProfileUpdate{ProfileVisibility: "members"})
	if err != nil || u.ProfileVisibility != "members" {
		t.Fatalf("profile: %v %+v", err, u)
	}
	if members, total, _ := st.ListMembers(ctx, model.Pagination{Page: 1, PerPage: 20}); total != 1 || members[0].ID != owner.ID {
		t.Errorf("members: %d %+v", total, members)
	}
	if u, _ = st.UpdateProfile(ctx, owner.ID, model.ProfileUpdate{}); u.ProfileVisibility != "private" {
		t.Errorf("empty visibility must mean private: %q", u.ProfileVisibility)
	}

	// Statistiques : un champ caché n'entre dans aucun agrégat ; types d'item joints.
	volume := uuid.MustParse("b0000000-0000-0000-0000-000000000015")
	it, err := st.CreateItem(ctx, model.ItemCreate{ItemTypeID: volume, Attributes: json.RawMessage(`{"number":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddCollectionItem(ctx, col.ID, model.CollectionItemAdd{ItemID: it.ID,
		Attributes: json.RawMessage(`{"condition":"new","purchase_price":9.5}`)}); err != nil {
		t.Fatal(err)
	}
	full, err := st.GetCollectionStats(ctx, col.ID, nil)
	if err != nil || len(full.Numbers) != 1 || len(full.ItemTypes) != 1 || full.ItemTypes[0].ID != volume {
		t.Fatalf("owner stats: %v %+v", err, full)
	}
	public, err := st.GetCollectionStats(ctx, col.ID, []string{"purchase_price"})
	if err != nil || len(public.Numbers) != 0 || len(public.Distributions) != 1 {
		t.Errorf("stats without the hidden field: %v %+v", err, public)
	}
}
