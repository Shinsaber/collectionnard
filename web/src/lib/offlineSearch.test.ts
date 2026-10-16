import { describe, expect, it } from "vitest";
import { buildRows, normalize, searchRows } from "./offlineSearch";
import type { OfflineSnapshot } from "@/api/types";

const snap = {
  generated_at: "2026-10-16T12:00:00Z",
  user: { id: "u1" },
  item_types: [
    {
      id: "t-vol",
      name: "Volume",
      attributes_schema: {
        "x-label": "{number}. {title}",
        "x-title": { fr: "Tome" },
        properties: {},
      },
    },
  ],
  collection_types: [{ id: "ct", name: "Ownership", attributes_schema: {} }],
  collections: [
    { id: "c1", name: "Mes BD", collection_type_id: "ct" },
    { id: "c2", name: "À lire", collection_type_id: "ct" },
  ],
  items: [
    {
      id: "i1",
      item_type_id: "t-vol",
      attributes: {
        number: 1,
        title: "Bienvenue à Pandorient",
        isbn: "9782800173191",
        images: { cover: ["/api/v1/uploads/x.jpg"] },
      },
    },
    {
      id: "i2",
      item_type_id: "t-vol",
      attributes: { number: 2, title: "Le Secret de Pandorient" },
    },
  ],
  entries: [
    { collection_id: "c1", item_id: "i1", attributes: {}, added_at: "" },
    { collection_id: "c2", item_id: "i1", attributes: {}, added_at: "" },
    { collection_id: "c1", item_id: "i2", attributes: {}, added_at: "" },
  ],
} as unknown as OfflineSnapshot;

describe("recherche hors ligne", () => {
  const rows = buildRows(snap, "fr");

  it("assemble libellé, type et entrées par item", () => {
    expect(rows.map((r) => r.label)).toEqual([
      "1. Bienvenue à Pandorient",
      "2. Le Secret de Pandorient",
    ]);
    expect(rows[0]!.type).toBe("Tome");
    expect(rows[0]!.entries.map((e) => e.collection.name)).toEqual([
      "Mes BD",
      "À lire",
    ]);
  });

  it("ignore accents et casse, reconnaît un ISBN avec tirets", () => {
    expect(normalize("Bienvenue À")).toBe("bienvenue a");
    expect(searchRows(rows, "bienvenue a").map((r) => r.item.id)).toEqual([
      "i1",
    ]);
    expect(searchRows(rows, "pandorient").length).toBe(2);
    expect(searchRows(rows, "978-2-8001-7319-1").map((r) => r.item.id)).toEqual(
      ["i1"],
    );
    expect(searchRows(rows, "tome secret").map((r) => r.item.id)).toEqual([
      "i2",
    ]);
    // Les URLs (galeries d'images) ne sont pas cherchées.
    expect(searchRows(rows, "uploads")).toEqual([]);
  });

  it("filtre par collection", () => {
    expect(searchRows(rows, "", "c2").map((r) => r.item.id)).toEqual(["i1"]);
    expect(searchRows(rows, "", null).length).toBe(2);
  });
});
