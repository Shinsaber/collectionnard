import { describe, expect, it } from "vitest";
import {
  conflictFields,
  itemPayload,
  mergeConflict,
  resolvePayload,
} from "./proposals";
import type { ProposalConflict } from "@/api/types";

const conflict: ProposalConflict = {
  kind: "item",
  id: "i1",
  base: { title: "A", year: 2000, isbn: "1" },
  current: { title: "A2", year: 2001, isbn: "1" },
  proposed: { title: "A3", year: 2000, isbn: "2" },
};

describe("proposals", () => {
  it("ne retient en conflit que les champs changés des deux côtés", () => {
    expect(conflictFields(conflict)).toEqual(["title"]);
  });

  it("fusionne à trois volets : chaque côté garde ses changements", () => {
    expect(mergeConflict(conflict)).toEqual({
      title: "A3",
      year: 2001,
      isbn: "2",
    });
    expect(mergeConflict(conflict, { title: "current" })).toEqual({
      title: "A2",
      year: 2001,
      isbn: "2",
    });
  });

  it("une entité supprimée depuis garde la proposition", () => {
    expect(mergeConflict({ ...conflict, current: null }).title).toBe("A3");
  });

  it("applique la fusion à la charge et construit la charge d'un item", () => {
    const p = itemPayload({
      id: "i1",
      item_type_id: "t",
      attributes: { title: "A3" },
    });
    expect(p.items![0]!.tags).toEqual([]);
    const out = resolvePayload(p, [conflict], { i1: { title: "current" } });
    expect(out.items![0]!.attributes).toEqual({
      title: "A2",
      year: 2001,
      isbn: "2",
    });
    expect(
      itemPayload({ item_type_id: "t", attributes: {} }).items![0]!.id,
    ).toMatch(/^[0-9a-f-]{36}$/);
  });
});
