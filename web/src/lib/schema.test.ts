import { describe, expect, it } from "vitest";
import fixtures from "../../tests/fixtures/seed-schemas.json";
import {
  defaultImage,
  displayValue,
  fieldLabel,
  isImageGallery,
  itemLabel,
  orderedKeys,
  renderLabel,
  type Schema,
} from "./schema";

const series = fixtures.find((f) => f.name === "Series")!.schema as Schema;
const episode = fixtures.find((f) => f.name === "Episode")!.schema as Schema;

describe("renderLabel / itemLabel", () => {
  it("applique le gabarit x-label", () => {
    expect(renderLabel(episode, { number: 1, title: "Pilot" })).toBe(
      "1. Pilot",
    );
    expect(
      renderLabel(
        { "x-label": "{last_name} {first_name}" },
        { first_name: "Akira", last_name: "Toriyama" },
      ),
    ).toBe("Toriyama Akira");
  });
  it("réduit les espaces quand un placeholder manque", () => {
    expect(renderLabel(episode, { title: "Pilot" })).toBe(". Pilot");
  });
  it("choisit la langue d'un objet i18n", () => {
    const schema = { "x-label": "{title}" };
    expect(
      renderLabel(schema, { title: { fr: "Le Pilote", en: "Pilot" } }, "en"),
    ).toBe("Pilot");
    expect(renderLabel(schema, { title: { fr: "Le Pilote" } }, "en")).toBe(
      "Le Pilote",
    );
  });
  it("retombe sur les clés usuelles sans x-label", () => {
    expect(renderLabel({ type: "object" }, { name: "AMC" })).toBe("AMC");
    expect(renderLabel(undefined, { titre: "Breaking Bad" })).toBe(
      "Breaking Bad",
    );
    expect(renderLabel({}, { foo: "bar" })).toBe("");
  });
  it("itemLabel retombe sur l'id", () => {
    expect(
      itemLabel({ id: "abcdef12-0000", attributes: {}, item_type: null }),
    ).toBe("abcdef12");
    expect(itemLabel(null)).toBe("?");
    expect(
      itemLabel({
        id: "x",
        attributes: { title: "Breaking Bad" },
        item_type: { attributes_schema: series },
      }),
    ).toBe("Breaking Bad");
  });
});

describe("fieldLabel / displayValue / orderedKeys", () => {
  it("lit x-labels dans la langue demandée avec repli", () => {
    expect(fieldLabel(series, "start_year", "fr")).toBe("Année de début");
    expect(fieldLabel(series, "start_year", "en")).toBe("Start Year");
    expect(fieldLabel({ "x-labels": { en: { a: "A!" } } }, "a", "fr")).toBe(
      "A!",
    );
    expect(fieldLabel(series, "unknown_key")).toBe("unknown_key");
    expect(fieldLabel(series, "unknown_key", "fr", "pretty")).toBe(
      "Unknown Key",
    );
  });
  it("displayValue gère i18n, tableaux et scalaires", () => {
    expect(displayValue({ fr: "Bonjour", en: "Hello" }, "en")).toBe("Hello");
    expect(displayValue({ en: "Hello" }, "fr")).toBe("Hello");
    expect(displayValue(["a", "b"])).toBe("a, b");
    expect(displayValue(12.5)).toBe("12.5");
    expect(displayValue(null)).toBe("");
  });
  it("orderedKeys suit x-order puis le schéma puis les données", () => {
    expect(
      orderedKeys(series, { end_year: 2013, extra: 1, title: "BB" }),
    ).toEqual([
      "title",
      "synopsis",
      "start_year",
      "end_year",
      "images",
      "extra",
    ]);
    expect(
      orderedKeys({
        "x-order": ["b", "a"],
        properties: { a: {}, b: {}, c: {} },
      }),
    ).toEqual(["b", "a", "c"]);
  });
});

describe("galeries", () => {
  it("détecte le widget image-gallery et l'image par défaut", () => {
    expect(isImageGallery(series, "images")).toBe(true);
    expect(isImageGallery(series, "title")).toBe(false);
    expect(
      defaultImage(series, {
        images: { default_image: "/x.jpg", poster: ["/p.jpg"] },
      }),
    ).toBe("/x.jpg");
    expect(
      defaultImage(series, {
        images: { default_image: "", poster: ["/p.jpg"] },
      }),
    ).toBe("/p.jpg");
    expect(defaultImage(series, { images: { poster: [] } })).toBe("");
  });
});
