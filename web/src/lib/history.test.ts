import { describe, expect, it } from "vitest";
import { changeKind, changedFields, latestItemVersion } from "./history";
import type { HistoryEntry } from "@/api/types";

const entry = (over: Partial<HistoryEntry>): HistoryEntry => ({
  kind: "item",
  entity_id: "i1",
  version: 1,
  origin: "direct",
  author: { kind: "system" },
  before: null,
  after: {},
  created_at: "",
  ...over,
});

describe("history", () => {
  it("liste les champs ajoutés, retirés et modifiés", () => {
    expect(
      changedFields(
        { title: "A", year: 2001, tags: ["x"] },
        { title: "B", tags: ["x"], isbn: "978" },
      ),
    ).toEqual([
      { key: "title", before: "A", after: "B" },
      { key: "year", before: 2001, after: undefined },
      { key: "isbn", before: undefined, after: "978" },
    ]);
    expect(changedFields(null, { a: 1 })).toEqual([
      { key: "a", before: undefined, after: 1 },
    ]);
  });

  it("qualifie une écriture et trouve la dernière version de la fiche", () => {
    expect(changeKind(entry({}))).toBe("created");
    expect(changeKind(entry({ before: {}, after: null }))).toBe("deleted");
    expect(changeKind(entry({ before: {}, after: {} }))).toBe("updated");
    expect(
      latestItemVersion([
        entry({ kind: "relation", version: 7 }),
        entry({ version: 3 }),
        entry({ version: 2 }),
      ]),
    ).toBe(3);
  });
});
