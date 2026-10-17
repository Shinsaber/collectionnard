//go:build integration

package store_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
)

// Statistiques d'une collection : agrégats du schéma « Possession », complétude par
// contains_volume (marquée x-completeness par le paquet books), synthèse de l'utilisateur.
func TestCollectionStats(t *testing.T) {
	st := freshStore(t, "colec_test_stats")
	if _, err := st.InstallDomains(ctx, "collections", "books"); err != nil {
		t.Fatal(err)
	}
	user, err := st.CreateUser(ctx, "camille", "camille@example.org", "x", model.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	var (
		editionType   = uuid.MustParse("b0000000-0000-0000-0000-000000000014")
		volumeType    = uuid.MustParse("b0000000-0000-0000-0000-000000000015")
		containsVol   = uuid.MustParse("c0000000-0000-0000-0000-000000000015")
		ownershipType = uuid.MustParse("d0000000-0000-0000-0000-000000000001")
	)
	item := func(typ uuid.UUID, attrs string) uuid.UUID {
		t.Helper()
		it, err := st.CreateItem(ctx, model.ItemCreate{ItemTypeID: typ, Attributes: json.RawMessage(attrs)})
		if err != nil {
			t.Fatal(err)
		}
		return it.ID
	}
	edition := item(editionType, `{"title":"Dragon Ball (Glénat)"}`)
	var volumes []uuid.UUID
	for n := 1; n <= 4; n++ {
		v := item(volumeType, `{"number":`+string(rune('0'+n))+`}`)
		volumes = append(volumes, v)
		if _, err := st.CreateRelation(ctx, model.ItemRelationCreate{RelationTypeID: containsVol, SourceItemID: edition, TargetItemID: v}); err != nil {
			t.Fatal(err)
		}
	}

	col, err := st.CreateCollection(ctx, user.ID, model.CollectionCreate{CollectionTypeID: ownershipType, Name: "Mangas"})
	if err != nil {
		t.Fatal(err)
	}
	for i, attrs := range []string{
		`{"condition":"new","purchase_price":6.9,"acquisition_date":"2026-09-01"}`,
		`{"condition":"good","purchase_price":3.1}`,
		`{}`,
	} {
		if _, err := st.AddCollectionItem(ctx, col.ID, model.CollectionItemAdd{ItemID: volumes[i], Attributes: json.RawMessage(attrs)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.CreateCollection(ctx, user.ID, model.CollectionCreate{CollectionTypeID: ownershipType, Name: "Vide"}); err != nil {
		t.Fatal(err)
	}

	stats, err := st.GetCollectionStats(ctx, col.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalItems != 3 || len(stats.ItemsByType) != 1 || stats.ItemsByType[0].TypeID != volumeType || stats.ItemsByType[0].Count != 3 {
		t.Errorf("totals: %d %+v", stats.TotalItems, stats.ItemsByType)
	}
	if len(stats.Numbers) != 1 || stats.Numbers[0].Field != "purchase_price" || stats.Numbers[0].Count != 2 || stats.Numbers[0].Sum != 10 {
		t.Errorf("numbers: %+v", stats.Numbers)
	}
	if len(stats.Distributions) != 1 || stats.Distributions[0].Missing != 1 {
		t.Errorf("distributions: %+v", stats.Distributions)
	}
	if len(stats.Timelines) != 1 || len(stats.Timelines[0].Months) != 1 || len(stats.Added) != 1 {
		t.Errorf("timelines: %+v, added: %+v", stats.Timelines, stats.Added)
	}
	if len(stats.Completeness) != 1 {
		t.Fatalf("completeness: %+v", stats.Completeness)
	}
	c := stats.Completeness[0]
	if c.Item.ID != edition || c.Item.ItemTypeID != editionType || c.Owned != 3 || c.Total != 4 || stats.CompletenessTruncated {
		t.Errorf("completeness: %+v", c)
	}

	me, err := st.GetUserStats(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if me.TotalCollections != 2 || me.TotalEntries != 3 || me.DistinctItems != 3 || me.Collections[0].ID != col.ID || me.Collections[1].ItemCount != 0 {
		t.Errorf("me: %+v", me)
	}
	if len(me.ItemsByType) != 1 || len(me.Added) != 1 || me.Added[0].Count != 3 {
		t.Errorf("me by type / added: %+v %+v", me.ItemsByType, me.Added)
	}

	nobody, err := st.GetUserStats(ctx, uuid.New())
	if err != nil || nobody.Collections == nil || nobody.ItemsByType == nil || nobody.Added == nil {
		t.Errorf("empty user: %v %+v", err, nobody)
	}
}
