import { mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { describe, expect, it } from "vitest";
import fixtures from "../../../tests/fixtures/seed-schemas.json";
import SchemaBuilder from "./SchemaBuilder.vue";

type Schema = Record<string, unknown>;
type Props = Record<string, Record<string, unknown>>;

/** Monte le builder sur un schéma et renvoie le schéma qu'il ré-émet. */
async function roundTrip(schema: Schema): Promise<Schema> {
  const w = mount(SchemaBuilder, {
    props: { modelValue: schema },
    global: { plugins: [ElementPlus] },
  });
  await w.vm.$nextTick();
  await w.vm.$nextTick();
  const emitted = w.emitted("update:modelValue");
  expect(emitted, "le builder doit ré-émettre le schéma").toBeTruthy();
  return emitted![emitted!.length - 1]![0] as Schema;
}

/** Sous-ensemble du schéma que le builder doit conserver à l'identique. */
function comparable(schema: Schema) {
  const props = (schema.properties ?? {}) as Props;
  const keep = [
    "type",
    "enum",
    "format",
    "minimum",
    "maximum",
    "minLength",
    "maxLength",
    "pattern",
    "x-widget",
    "x-image-categories",
  ];
  const properties: Record<string, Record<string, unknown>> = {};
  for (const [key, prop] of Object.entries(props)) {
    const out: Record<string, unknown> = {};
    for (const k of keep) if (prop[k] !== undefined) out[k] = prop[k];
    if (prop.type === "array") out.items = prop.items;
    properties[key] = out;
  }
  return {
    required: [...((schema.required as string[]) ?? [])].sort(),
    "x-labels": schema["x-labels"] ?? undefined,
    properties,
  };
}

describe("SchemaBuilder — aller-retour sur les schémas du seed", () => {
  for (const f of fixtures) {
    it(`${f.name} : types, contraintes, required, x-labels et widgets conservés`, async () => {
      const out = await roundTrip(f.schema as Schema);
      expect(out.type).toBe("object");
      expect(comparable(out)).toEqual(comparable(f.schema as Schema));
      // Le builder matérialise toujours l'ordre des champs.
      expect(out["x-order"]).toEqual(
        Object.keys((f.schema as Schema).properties as Props),
      );
    });
  }

  it("conserve la description des propriétés", async () => {
    const out = await roundTrip({
      type: "object",
      properties: { a: { type: "string", description: "Un champ" } },
    });
    expect(
      ((out.properties as Props).a as Record<string, unknown>).description,
    ).toBe("Un champ");
  });
});

import { it as itTop } from "vitest";

// Les mots-clés inconnus du builder (futurs `x-*`, `default` sur un objet…) et le
// gabarit racine `x-label` sont conservés à la reconstruction (lot 8).
itTop("conserve une extension x-* inconnue du builder", async () => {
  const out = await roundTrip({
    type: "object",
    properties: { a: { type: "string", "x-custom": 42 } },
  });
  expect(
    ((out.properties as Props).a as Record<string, unknown>)["x-custom"],
  ).toBe(42);
});

itTop(
  "conserve le gabarit racine x-label et les clés racine inconnues",
  async () => {
    const out = await roundTrip({
      type: "object",
      "x-label": "{number}. {title}",
      "x-future": { a: 1 },
      properties: { number: { type: "integer" }, title: { type: "string" } },
    });
    expect(out["x-label"]).toBe("{number}. {title}");
    expect(out["x-future"]).toEqual({ a: 1 });
  },
);

// Internationalisation : nom/description du type par langue, libellés d'enum, libellés
// de langues que l'interface ne supporte pas encore (conservés), objet i18n-text extensible.
itTop(
  "conserve x-title, x-description, x-enum-labels et les langues inconnues",
  async () => {
    const out = await roundTrip({
      type: "object",
      "x-title": { fr: "Série", en: "Series", de: "Serie" },
      "x-description": { fr: "Série télévisée" },
      "x-labels": { fr: { kind: "Genre" }, de: { kind: "Art" } },
      properties: {
        kind: {
          type: "string",
          enum: ["tv", "web"],
          "x-enum-labels": { fr: { tv: "Télévision" }, en: { tv: "TV" } },
        },
      },
    });
    expect(out["x-title"]).toEqual({ fr: "Série", en: "Series", de: "Serie" });
    expect(out["x-description"]).toEqual({ fr: "Série télévisée" });
    expect(out["x-labels"]).toEqual({
      fr: { kind: "Genre" },
      de: { kind: "Art" },
    });
    expect(
      ((out.properties as Props).kind as Record<string, unknown>)[
        "x-enum-labels"
      ],
    ).toEqual({ fr: { tv: "Télévision" }, en: { tv: "TV" } });
  },
);

itTop("génère un i18n-text ouvert aux langues futures", async () => {
  const out = await roundTrip({
    type: "object",
    properties: {
      synopsis: {
        type: "object",
        "x-widget": "i18n-text",
        properties: { fr: { type: "string" } },
        additionalProperties: false,
      },
    },
  });
  const prop = (out.properties as Props).synopsis as Record<string, unknown>;
  expect(Object.keys(prop.properties as object)).toEqual(["fr", "en"]);
  expect(prop.additionalProperties).toEqual({ type: "string" });
  expect(prop.propertyNames).toBeTruthy();
});

// Lot 13 : schémas BD / Manga — objets génériques (`external_ids`), tableaux d'objets
// (`sources`) et tableaux d'enum (`roles`) conservés intégralement.
itTop(
  "conserve external_ids, sources et les rôles (tableau d'enum)",
  async () => {
    const byName = (n: string) =>
      fixtures.find((f) => f.name === n)!.schema as Schema;
    for (const name of ["Manga", "created_by"]) {
      const src = byName(name);
      const out = await roundTrip(src);
      const outProps = out.properties as Props;
      const srcProps = src.properties as Props;
      for (const key of Object.keys(srcProps).filter((k) =>
        ["external_ids", "sources", "roles"].includes(k),
      )) {
        expect(outProps[key], `${name}.${key}`).toEqual(srcProps[key]);
      }
    }
  },
);

describe("SchemaBuilder — liste repliable des champs", () => {
  const schema: Schema = {
    type: "object",
    properties: { number: { type: "integer" }, title: { type: "string" } },
    "x-order": ["number", "title"],
  };

  function rows(w: ReturnType<typeof mount>) {
    return w.findAll("li > button[aria-expanded]");
  }

  it("un champ déplié le reste quand on le déplace", async () => {
    const w = mount(SchemaBuilder, {
      props: { modelValue: schema },
      global: { plugins: [ElementPlus] },
    });
    expect(rows(w).map((r) => r.attributes("aria-expanded"))).toEqual([
      "false",
      "false",
    ]);

    await rows(w)[0]!.trigger("click");
    expect(rows(w)[0]!.attributes("aria-expanded")).toBe("true");

    await w.find('button[aria-label="Descendre le champ"]').trigger("click");
    const [first, second] = rows(w);
    expect(first!.text()).toContain("title");
    expect(first!.attributes("aria-expanded")).toBe("false");
    expect(second!.text()).toContain("number");
    expect(second!.attributes("aria-expanded")).toBe("true");
  });

  it("un champ ajouté s'ouvre pour être saisi", async () => {
    const w = mount(SchemaBuilder, {
      props: { modelValue: schema },
      global: { plugins: [ElementPlus] },
    });
    const add = w
      .findAll("button")
      .find((b) => b.text().includes("Ajouter un champ"))!;
    await add.trigger("click");
    const all = rows(w);
    expect(all).toHaveLength(3);
    expect(all[2]!.attributes("aria-expanded")).toBe("true");
  });
});
