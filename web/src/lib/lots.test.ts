import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ItemCollectionMembership } from "@/api/types";

const api = vi.hoisted(() => ({ addItem: vi.fn(), updateLot: vi.fn() }));
vi.mock("@/api/client", () => ({ collectionApi: api }));

import { addCopies, collectionCount, lotSummary, lotsIn } from "./lots";

const schema = {
  type: "object",
  "x-order": ["condition", "photos"],
  "x-labels": { fr: { condition: "État" } },
  properties: {
    condition: {
      type: "string",
      enum: ["good", "poor"],
      "x-enum-labels": { fr: { good: "Bon", poor: "Mauvais" } },
    },
    photos: { type: "object", "x-widget": "image-gallery" },
  },
};

function lot(
  collectionId: string,
  lotId: string,
  quantity: number,
  attributes: Record<string, unknown> = {},
): ItemCollectionMembership {
  return {
    collection: { id: collectionId } as ItemCollectionMembership["collection"],
    lot_id: lotId,
    quantity,
    attributes,
  };
}

describe("lots (P7-1)", () => {
  beforeEach(() => {
    api.addItem.mockReset().mockResolvedValue({});
    api.updateLot.mockReset().mockResolvedValue({});
  });

  const memberships = [
    lot("c1", "l1", 5, { condition: "good" }),
    lot("c1", "l2", 1, { condition: "poor" }),
    lot("c2", "l3", 1),
  ];

  it("regroupe les lots par collection", () => {
    expect(lotsIn(memberships, "c1").map((l) => l.lot_id)).toEqual([
      "l1",
      "l2",
    ]);
    expect(collectionCount(memberships)).toBe(2);
  });

  it("résume les attributs d'un lot, sans les galeries", () => {
    expect(
      lotSummary(schema, { condition: "good", photos: { a: ["x"] } }, "fr"),
    ).toBe("État : Bon");
    expect(lotSummary(schema, {}, "fr")).toBe("");
  });

  it("augmente un lot existant en gardant ses attributs", async () => {
    await addCopies("c1", "e1", "l1", 2, { condition: "poor" }, memberships);
    expect(api.updateLot).toHaveBeenCalledWith("c1", "l1", {
      attributes: { condition: "good" },
      quantity: 7,
    });
    expect(api.addItem).not.toHaveBeenCalled();
  });

  it("crée un nouveau lot avec sa quantité et ses attributs", async () => {
    await addCopies("c1", "e1", "new", 3, { condition: "poor" }, memberships);
    expect(api.addItem).toHaveBeenCalledWith("c1", {
      item_id: "e1",
      attributes: { condition: "poor" },
      quantity: 3,
    });
  });
});

describe("lotQueryParams", () => {
  it("n'envoie que les critères renseignés", async () => {
    const { emptyLotQuery, lotQueryParams } = await import("./lots");
    expect(lotQueryParams(emptyLotQuery())).toEqual({});
    expect(
      lotQueryParams({
        search: "  sonic ",
        item_type_id: "b1",
        sort: "attr.number",
        multiple: true,
      }),
    ).toEqual({
      search: "sonic",
      item_type_id: "b1",
      sort: "attr.number",
      multiple: "true",
    });
  });
});
