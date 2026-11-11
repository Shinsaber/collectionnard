import { describe, expect, it } from "vitest";
import type { Collection } from "@/api/types";
import {
  acceptedTypeIds,
  lotSortOptions,
  specializedTypeId,
} from "./collections";

function col(own: string[], limit: string[]) {
  return {
    item_type_ids: own,
    collection_type: { item_type_ids: limit },
  } as unknown as Collection;
}

describe("types acceptés (P7-1b)", () => {
  it("resserre le type de collection, sans jamais l'élargir", () => {
    expect(acceptedTypeIds(col([], []))).toEqual([]);
    expect(acceptedTypeIds(col([], ["a", "b"]))).toEqual(["a", "b"]);
    expect(acceptedTypeIds(col(["a"], []))).toEqual(["a"]);
    expect(acceptedTypeIds(col(["a", "c"], ["a", "b"]))).toEqual(["a"]);
    // Liste vidée par un resserrement du type : celle du type.
    expect(acceptedTypeIds(col(["c"], ["a", "b"]))).toEqual(["a", "b"]);
  });

  it("une collection qui n'accepte qu'un type est spécialisée", () => {
    expect(specializedTypeId(col(["a"], []))).toBe("a");
    expect(specializedTypeId(col([], ["b"]))).toBe("b");
    expect(specializedTypeId(col(["a", "b"], []))).toBeUndefined();
    expect(specializedTypeId(col([], []))).toBeUndefined();
  });

  it("propose les tris communs, plus les attributs du type", () => {
    const t = (k: string) => k;
    expect(lotSortOptions(t, undefined, "fr").map((o) => o.value)).toEqual([
      "-created_at",
      "created_at",
      "-quantity",
    ]);
    const schema = {
      type: "object",
      "x-labels": { fr: { number: "Numéro" } },
      properties: { number: { type: "integer" }, photos: { type: "object" } },
    };
    const options = lotSortOptions(t, schema, "fr");
    expect(options.slice(3)).toEqual([
      { value: "attr.number", label: "Numéro ↑" },
      { value: "-attr.number", label: "Numéro ↓" },
    ]);
  });
});
