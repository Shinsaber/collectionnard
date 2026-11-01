import { describe, expect, it } from "vitest";
import type { ItemRelation, ItemType, RelationType } from "@/api/types";
import { byId, withTypes } from "./relations";

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
