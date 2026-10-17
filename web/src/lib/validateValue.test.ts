import { describe, expect, it } from "vitest";
import fixtures from "../../tests/fixtures/seed-schemas.json";
import { checkValue, type Schema } from "./validateValue";

const prop = (type: string, name: string) =>
  (
    fixtures.find((f) => f.name === type)!.schema as unknown as {
      properties: Record<string, Schema>;
    }
  ).properties[name];

describe("checkValue (schémas des paquets)", () => {
  it("motif d'un texte : code-barres d'une édition musicale", () => {
    const barcode = prop("Release", "barcode");
    expect(checkValue(barcode, "634904078126")).toBe(null);
    expect(checkValue(barcode, "63490-4078")).toEqual({
      key: "pattern",
      params: { hint: "^[0-9]{8,14}$" },
      path: "",
    });
    // Valeur absente : laissée à `required`.
    expect(checkValue(barcode, "")).toBe(null);
  });

  it("motif dans un objet JSON : ARK BnF des identifiants d'un film", () => {
    const ids = prop("Film", "external_ids");
    expect(
      checkValue(ids, {
        wikidata: "Q484048",
        bnf_ark: "ark:/12148/cb14404106q",
      }),
    ).toBe(null);
    expect(checkValue(ids, { wikidata: "484048" })).toMatchObject({
      key: "pattern",
      path: "wikidata",
    });
    expect(checkValue(ids, { tmdb: "194" })).toEqual({
      key: "unknownKey",
      params: { key: "tmdb" },
      path: "",
    });
  });

  it("éléments d'une liste : pays d'origine d'une série", () => {
    const countries = prop("Series", "countries");
    expect(checkValue(countries, ["US", "FR"])).toBe(null);
    expect(checkValue(countries, ["US", "fr"])).toMatchObject({
      key: "pattern",
      path: "2",
    });
    expect(checkValue(countries, ["US", "US"])).toEqual({
      key: "uniqueItems",
      path: "",
    });
  });

  it("type, bornes, longueur, date, clé requise", () => {
    expect(checkValue({ type: "integer", minimum: 1 }, 0)).toMatchObject({
      key: "minimum",
    });
    expect(checkValue({ type: "integer" }, 1.5)).toMatchObject({
      key: "type",
      params: { type: "integer" },
    });
    expect(checkValue({ type: "string", minLength: 3 }, "ab")).toMatchObject({
      key: "minLength",
    });
    expect(
      checkValue({ type: "string", format: "date" }, "2026-02-30"),
    ).toMatchObject({ key: "date" });
    expect(checkValue({ type: "string", format: "date" }, "2026-02-28")).toBe(
      null,
    );
    const sources = prop("Film", "sources");
    expect(checkValue(sources, [{ retrieved_at: "2026-10-17" }])).toEqual({
      key: "requiredKey",
      params: { key: "source" },
      path: "1",
    });
  });

  it("texte multilingue : clés de langue", () => {
    const title = prop("Film", "title");
    expect(checkValue(title, { fr: "Amélie", en: "Amélie" })).toBe(null);
    expect(checkValue(title, { Français: "Amélie" })).toMatchObject({
      key: "keyPattern",
    });
  });
});
