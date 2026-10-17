import { describe, expect, it } from "vitest";
import fixtures from "../../tests/fixtures/seed-schemas.json";
import {
  activeAttrCount,
  filterFields,
  fromQuery,
  sortFields,
  toQuery,
} from "./itemSearch";
import type { Schema } from "./schema";

const schemaOf = (name: string) =>
  fixtures.find((f) => f.name === name)!.schema as Schema;

describe("champs filtrables et triables", () => {
  it("dérive les contrôles du schéma, dans l'ordre x-order, sans la galerie", () => {
    expect(filterFields(schemaOf("Series"))).toEqual([
      { key: "title", control: "text" },
      { key: "synopsis", control: "i18nText" },
      {
        key: "start_year",
        control: "numberRange",
        integer: true,
        minimum: 1900,
        maximum: undefined,
      },
      {
        key: "end_year",
        control: "numberRange",
        integer: true,
        minimum: 1900,
        maximum: undefined,
      },
      { key: "original_title", control: "text" },
      { key: "original_language", control: "text" },
      {
        key: "season_count",
        control: "numberRange",
        integer: true,
        minimum: 1,
        maximum: undefined,
      },
      {
        key: "episode_count",
        control: "numberRange",
        integer: true,
        minimum: 1,
        maximum: undefined,
      },
    ]);
    expect(filterFields(schemaOf("Broadcast"))).toContainEqual({
      key: "type",
      control: "enum",
      options: ["tv", "streaming", "cable", "web"],
    });
  });

  it("gère dates, booléens et tableaux d'enum", () => {
    const fields = filterFields({
      type: "object",
      properties: {
        aired: { type: "string", format: "date" },
        owned: { type: "boolean" },
        genres: { type: "array", items: { type: "string", enum: ["a", "b"] } },
        free: { type: "array", items: { type: "string" } },
      },
    });
    expect(fields.map((f) => [f.key, f.control])).toEqual([
      ["aired", "dateRange"],
      ["owned", "boolean"],
      ["genres", "enum"],
    ]);
  });

  it("ne propose au tri que les scalaires", () => {
    expect(sortFields(schemaOf("Series"))).toEqual([
      { key: "title", numeric: false },
      { key: "start_year", numeric: true },
      { key: "end_year", numeric: true },
      { key: "original_title", numeric: false },
      { key: "original_language", numeric: false },
      { key: "season_count", numeric: true },
      { key: "episode_count", numeric: true },
    ]);
  });
});

describe("état ↔ query", () => {
  it("aller-retour, valeurs vides omises, filtres répétés", () => {
    const query = toQuery({
      item_type_id: "b1",
      search: "  breaking ",
      tag: "",
      sort: "-attr.start_year",
      page: 2,
      attrs: {
        "attr.type": ["tv", "cable"],
        "attr.start_year.gte": ["2008"],
        "attr.end_year.lte": [""],
      },
    });
    expect(query).toEqual({
      item_type_id: "b1",
      search: "breaking",
      sort: "-attr.start_year",
      page: "2",
      "attr.type": ["tv", "cable"],
      "attr.start_year.gte": "2008",
    });
    const back = fromQuery(query);
    expect(back).toEqual({
      item_type_id: "b1",
      search: "breaking",
      tag: undefined,
      sort: "-attr.start_year",
      related_to: undefined,
      relation_type_id: undefined,
      page: 2,
      attrs: { "attr.type": ["tv", "cable"], "attr.start_year.gte": ["2008"] },
    });
    expect(activeAttrCount(back)).toBe(2);
  });

  it("ignore une page invalide et les paramètres inconnus", () => {
    const s = fromQuery({ page: "abc", foo: "bar", "attr.x": null });
    expect(s.page).toBeUndefined();
    expect(s.attrs).toEqual({});
  });
});
