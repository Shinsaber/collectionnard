import { describe, expect, it } from "vitest";
import { attributeRows } from "./attributeView";
import type { Schema } from "./schema";

const schema: Schema = {
  type: "object",
  "x-order": ["title", "isbn", "external_ids", "links", "website"],
  "x-labels": { fr: { title: "Titre", external_ids: "Identifiants externes" } },
  properties: {
    title: { type: "object", "x-widget": "i18n-text" },
    isbn: { type: "string", "x-identifier": true },
    page_count: { type: "integer" },
    external_ids: {
      type: "object",
      properties: { wikidata: { type: "string", "x-identifier": true } },
    },
    links: { type: "array", "x-widget": "links" },
    website: { type: "string", format: "uri" },
    credits: { type: "array", items: { type: "object" } },
    images: { type: "object", "x-widget": "image-gallery" },
  },
};

describe("attributeRows", () => {
  const rows = attributeRows(
    schema,
    {
      title: { fr: "Tome 3", en: "Volume 3" },
      isbn: "978",
      page_count: 192,
      external_ids: { wikidata: "Q1", bnf_ark: "ark:/12148/x" },
      links: [{ url: "https://www.wikidata.org/wiki/Q1", label: "Wikidata" }],
      website: "https://editeur.example/tome-3",
      credits: [{ role: "dessin", name: "A" }],
      images: { front: ["/u/a.jpg"] },
      empty: "",
    },
    "fr",
  );

  it("suit le schéma, saute la galerie et les valeurs vides, lisibles sans traduction", () => {
    expect(rows.map((r) => r.key)).toEqual([
      "title",
      "isbn",
      "external_ids",
      "links",
      "website",
      "page_count",
      "credits",
    ]);
    expect(rows.find((r) => r.key === "page_count")!.label).toBe("Page Count");
    expect(rows[0]!.view).toEqual({ kind: "text", text: "Tome 3" });
    expect(rows[1]!.mono).toBe(true);
  });

  it("déplie un objet en clé : valeur et une liste d'objets", () => {
    const ext = rows.find((r) => r.key === "external_ids")!;
    expect(ext.label).toBe("Identifiants externes");
    expect(ext.view.kind).toBe("object");
    const sub = ext.view.kind === "object" ? ext.view.rows : [];
    expect(sub.map((r) => [r.key, r.mono])).toEqual([
      ["wikidata", true],
      ["bnf_ark", false],
    ]);
    const credits = rows.find((r) => r.key === "credits")!.view;
    expect(credits.kind === "list" && credits.items[0]!.kind).toBe("object");
  });

  it("montre les liens par leur libellé, une URL par son site", () => {
    expect(rows.find((r) => r.key === "links")!.view).toEqual({
      kind: "links",
      links: [{ url: "https://www.wikidata.org/wiki/Q1", label: "Wikidata" }],
    });
    expect(rows.find((r) => r.key === "website")!.view).toEqual({
      kind: "links",
      links: [{ url: "https://editeur.example/tome-3" }],
    });
  });
});
