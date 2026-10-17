import { describe, expect, it } from "vitest";
import fixtures from "../../tests/fixtures/seed-schemas.json";
import {
  checkImage,
  defaultImage,
  displayValue,
  enumLabel,
  fieldLabel,
  fieldValue,
  i18nText,
  imageCategories,
  isImageGallery,
  isLinks,
  itemLabel,
  linkText,
  linksOf,
  linksProperty,
  orderedKeys,
  parseRatio,
  renderLabel,
  safeURL,
  typeDescription,
  typeTitle,
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
      "original_title",
      "countries",
      "original_language",
      "season_count",
      "episode_count",
      "images",
      "external_ids",
      "sources",
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
  it("lit les contrôles d'une catégorie et ignore les valeurs mal typées", () => {
    const schema = {
      properties: {
        g: {
          "x-image-categories": {
            cover: {
              ratio: "2/3",
              tolerance: 10,
              strict: true,
              min_width: 300,
            },
            back: { ratio: "2/3", tolerance: "10", strict: "yes" },
          },
        },
      },
    } as Schema;
    expect(imageCategories(schema, "g")).toEqual([
      {
        name: "cover",
        ratio: "2/3",
        tolerance: 10,
        strict: true,
        min_width: 300,
      },
      { name: "back", ratio: "2/3" },
    ]);
  });

  it("contrôle ratio et taille minimale d'une image", () => {
    expect(parseRatio("2/3")).toBeCloseTo(0.667, 3);
    expect(parseRatio("16 / 9")).toBeCloseTo(1.778, 3);
    expect(parseRatio("1.5")).toBe(1.5);
    expect(parseRatio("abc")).toBeNaN();

    const cover = { name: "cover", ratio: "2/3", tolerance: 10 };
    // 700 × 963 : ratio 0,727, écart 9 % → toléré à 10 %, refusé à 5 %.
    expect(checkImage(cover, 700, 963)).toEqual({ issues: [], block: false });
    const off = checkImage({ ...cover, tolerance: 5 }, 700, 963);
    expect(off.block).toBe(false);
    expect(off.issues).toMatchObject([{ kind: "ratio", expected: "2/3" }]);
    // Sans tolérance, le ratio n'est pas contrôlé.
    expect(checkImage({ name: "c", ratio: "2/3" }, 1000, 100).issues).toEqual(
      [],
    );
    // Taille minimale, et refus quand la catégorie est stricte.
    const small = checkImage(
      { name: "c", ratio: "1/1", min_width: 500, strict: true },
      400,
      400,
    );
    expect(small).toEqual({
      issues: [{ kind: "size", minWidth: 500, minHeight: 0 }],
      block: true,
    });
  });

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

describe("i18n : objets multilingues, enums, noms de types", () => {
  it("i18nText : langue demandée, défaut, langues connues puis autres clés triées", () => {
    expect(i18nText({ fr: "Bonjour", en: "Hello" }, "en")).toBe("Hello");
    expect(i18nText({ fr: "Bonjour", en: "" }, "en")).toBe("Bonjour");
    expect(i18nText({ en: "Hello" }, "fr")).toBe("Hello");
    expect(i18nText({ es: "Hola", de: "Hallo" }, "fr")).toBe("Hallo");
    expect(i18nText({}, "fr")).toBe("");
  });

  const schema: Schema = {
    type: "object",
    properties: {
      type: {
        type: "string",
        enum: ["tv", "web"],
        "x-enum-labels": { fr: { tv: "Télévision" }, en: { tv: "TV" } },
      },
      tags: {
        type: "array",
        "x-enum-labels": { fr: { a: "Un" } },
      },
    },
  };

  it("enumLabel / fieldValue traduisent les valeurs d'enum", () => {
    expect(enumLabel(schema, "type", "tv", "en")).toBe("TV");
    expect(enumLabel(schema, "type", "tv", "fr")).toBe("Télévision");
    expect(enumLabel(schema, "type", "web", "fr")).toBe("web");
    expect(fieldValue(schema, "type", "tv", "fr")).toBe("Télévision");
    expect(fieldValue(schema, "tags", ["a", "b"], "en")).toBe("Un, b");
    expect(fieldValue(schema, "other", { fr: "x" }, "en")).toBe("x");
  });

  it("fieldValue met en forme décimaux et dates selon la langue, pas les entiers", () => {
    const s: Schema = {
      type: "object",
      properties: {
        price: { type: "number" },
        year: { type: "integer" },
        bought: { type: "string", format: "date" },
        note: { type: "string" },
      },
    };
    expect(fieldValue(s, "price", 6.5, "fr")).toBe("6,5");
    expect(fieldValue(s, "price", 6.5, "en")).toBe("6.5");
    expect(fieldValue(s, "price", 1234.567, "fr")).toBe("1234,57");
    expect(fieldValue(s, "year", 2008, "fr")).toBe("2008");
    expect(fieldValue(s, "bought", "2021-03-10", "fr")).toBe("10 mars 2021");
    expect(fieldValue(s, "bought", "2021-03-10", "en")).toBe("Mar 10, 2021");
    expect(fieldValue(s, "bought", "pas une date", "fr")).toBe("pas une date");
    expect(fieldValue(s, "note", "2021-03-10", "fr")).toBe("2021-03-10");
  });

  it("typeTitle / typeDescription lisent x-title / x-description avec repli", () => {
    const t = {
      name: "Series",
      description: "Série télévisée",
      attributes_schema: { "x-title": { fr: "Série", en: "Series" } },
    };
    expect(typeTitle(t, "fr")).toBe("Série");
    expect(typeTitle({ name: "Raw" }, "fr")).toBe("Raw");
    expect(typeDescription(t, "en")).toBe("Série télévisée");
    expect(
      typeDescription(
        { ...t, attributes_schema: { "x-description": { en: "TV series" } } },
        "en",
      ),
    ).toBe("TV series");
  });
});

describe("liens (x-widget: links)", () => {
  const publisher = fixtures.find((f) => f.name === "Publisher")!
    .schema as Schema;

  it("reconnaît le champ et n'affiche que les liens http(s)", () => {
    expect(isLinks(publisher, "links")).toBe(true);
    expect(isLinks(publisher, "website")).toBe(false);
    const links = linksOf([
      { label: "Wikipédia", url: "https://fr.wikipedia.org/wiki/Nintendo" },
      { url: "https://www.nintendo.com/fr-fr/" },
      { label: "piège", url: "javascript:alert(1)" },
      { label: "vide" },
    ]);
    expect(links).toEqual([
      { label: "Wikipédia", url: "https://fr.wikipedia.org/wiki/Nintendo" },
      { url: "https://www.nintendo.com/fr-fr/" },
    ]);
    expect(links.map(linkText)).toEqual(["Wikipédia", "www.nintendo.com"]);
    expect(safeURL("data:text/html,x")).toBe("");
  });

  it("fieldValue : les libellés (ou noms de site), séparés par des virgules", () => {
    expect(
      fieldValue(publisher, "links", [
        { label: "Site", url: "https://a.example" },
        { url: "https://b.example/x" },
      ]),
    ).toBe("Site, b.example");
  });

  it("linksProperty : le schéma écrit par l'éditeur de types", () => {
    expect(linksProperty()).toEqual(
      (publisher.properties as Record<string, unknown>).links,
    );
  });
});
