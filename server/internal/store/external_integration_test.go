//go:build integration

package store_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
)

func TestFindItemsByAttributeAndRelationsBetween(t *testing.T) {
	it, err := s.CreateItemType(ctx, model.ItemTypeCreate{Name: "ExtPublisher", AttributesSchema: json.RawMessage(`{"type":"object"}`)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DeleteItemType(ctx, it.ID) })
	glenat, err := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: it.ID, Tags: []string{"bd"},
		Attributes: json.RawMessage(`{"name":"Glénat","external_ids":{"wikidata":"Q1454296"}}`)})
	if err != nil {
		t.Fatal(err)
	}
	kana, err := s.CreateItem(ctx, model.ItemCreate{ItemTypeID: it.ID,
		Attributes: json.RawMessage(`{"name":"Kana","external_ids":{"wikidata":"Q3192366"}}`)})
	if err != nil {
		t.Fatal(err)
	}
	// Items created here must not change the counts of the other tests.
	t.Cleanup(func() { _ = s.DeleteItem(ctx, glenat.ID); _ = s.DeleteItem(ctx, kana.ID) })

	found, err := s.FindItemsByAttribute(ctx, it.ID, []string{"external_ids", "wikidata"}, []string{"Q1454296", "Q9"}, false)
	if err != nil || len(found) != 1 || found[0].ID != glenat.ID || len(found[0].Tags) != 1 || found[0].Tags[0].Name != "bd" {
		t.Errorf("by QID: %v %+v", err, found)
	}
	// Case- and accent-insensitive (f_unaccent).
	found, err = s.FindItemsByAttribute(ctx, it.ID, []string{"name"}, []string{"GLENAT"}, true)
	if err != nil || len(found) != 1 || found[0].ID != glenat.ID {
		t.Errorf("folded name: %v %+v", err, found)
	}
	if found, _ = s.FindItemsByAttribute(ctx, it.ID, []string{"name"}, []string{"GLENAT"}, false); len(found) != 0 {
		t.Errorf("exact name must not fold: %+v", found)
	}
	// Another type, or a path leading nowhere: nothing.
	if found, _ = s.FindItemsByAttribute(ctx, typeSerie, []string{"name"}, []string{"Glénat"}, true); len(found) != 0 {
		t.Errorf("other type: %+v", found)
	}
	if found, _ = s.FindItemsByAttribute(ctx, it.ID, []string{"name", "x"}, []string{"Glénat"}, false); len(found) != 0 {
		t.Errorf("deep path on a string: %+v", found)
	}

	rt, err := s.CreateRelationType(ctx, model.RelationTypeCreate{Name: "ext_partner", SourceItemTypeIDs: []uuid.UUID{it.ID},
		TargetItemTypeIDs: []uuid.UUID{it.ID}, AttributesSchema: json.RawMessage(`{"type":"object"}`)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DeleteRelationType(ctx, rt.ID) })
	rel, err := s.CreateRelation(ctx, model.ItemRelationCreate{RelationTypeID: rt.ID, SourceItemID: glenat.ID, TargetItemID: kana.ID})
	if err != nil {
		t.Fatal(err)
	}
	rels, err := s.ListRelationsBetween(ctx, []uuid.UUID{glenat.ID, kana.ID, itemSerie})
	if err != nil || len(rels) != 1 || rels[0].ID != rel.ID {
		t.Errorf("between: %v %+v", err, rels)
	}
	if rels, _ = s.ListRelationsBetween(ctx, []uuid.UUID{glenat.ID}); len(rels) != 0 {
		t.Errorf("one end only: %+v", rels)
	}
}
