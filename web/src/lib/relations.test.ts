import { describe, expect, it } from "vitest";
import type { ItemRelation, ItemType, RelationType } from "@/api/types";
import { byId, partOf, withTypes } from "./relations";

const volume = {
  id: "b1",
  name: "Volume",
  attributes_schema: { "x-label": "{title}" },
} as unknown as ItemType;
const contains = {
  id: "c1",
  name: "contains_volume",
  attributes_schema: { type: "object" },
} as unknown as RelationType;

const rel = {
  id: "r1",
  relation_type_id: "c1",
  source_item_id: "e1",
  source_item: { id: "e1", item_type_id: "b1", attributes: { title: "Série" } },
  target_item_id: "e2",
  target_item: {
    id: "e2",
    item_type_id: "zz",
    attributes: { title: "Tome 1" },
  },
  attributes: { order: 1 },
  created_at: "2026-10-18T00:00:00Z",
  updated_at: "2026-10-18T00:00:00Z",
} as unknown as ItemRelation;

describe("lib/relations", () => {
  it("complète le type de la relation et des extrémités connues, laisse les autres intacts", () => {
    const out = withTypes(rel, byId([volume]), byId([contains]));
    expect(out.relation_type).toBe(contains);
    expect(out.source_item?.item_type).toBe(volume);
    expect(out.target_item?.item_type).toBeUndefined(); // type inconnu du cache
    expect(out.target_item).toBe(rel.target_item); // objet d'origine conservé
    expect(rel.relation_type).toBeUndefined(); // entrée non modifiée
  });

  it("garde les types déjà joints par l'API", () => {
    const other = {
      id: "c9",
      name: "other",
      attributes_schema: {},
    } as unknown as RelationType;
    const joined = { ...rel, relation_type: other };
    expect(withTypes(joined, {}, byId([contains])).relation_type).toBe(other);
  });
});

describe("lib/relations — partOf", () => {
  it("retient les relations entrantes x-completeness, avec leur ordre, libellées par l'item source", () => {
    const edition = {
      id: "ed1",
      item_type_id: "b2",
      attributes: { name: "Glénat" },
      item_type: {
        id: "b2",
        name: "Edition",
        attributes_schema: { "x-label": "{name}" },
      },
    } as unknown as ItemRelation["source_item"];
    const composition = {
      id: "c1",
      name: "contains_volume",
      attributes_schema: { type: "object", "x-completeness": true },
    } as unknown as RelationType;
    const other = {
      id: "c2",
      name: "published_by",
      attributes_schema: { type: "object" },
    } as unknown as RelationType;
    const rels = [
      {
        ...rel,
        id: "r1",
        relation_type: composition,
        source_item: edition,
        source_item_id: "ed1",
        target_item_id: "e2",
        attributes: { order: 3 },
      },
      {
        ...rel,
        id: "r2",
        relation_type: other,
        source_item: edition,
        source_item_id: "ed1",
        target_item_id: "e2",
        attributes: {},
      },
      {
        ...rel,
        id: "r3",
        relation_type: composition,
        source_item_id: "e2",
        target_item_id: "ed1",
        attributes: {},
      }, // sortante
    ] as ItemRelation[];
    expect(partOf(rels, "e2", "fr")).toEqual([
      { id: "ed1", label: "Glénat", order: 3 },
    ]);
    expect(partOf(rels, undefined, "fr")).toEqual([]);
  });
});
