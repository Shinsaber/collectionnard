import { mount } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import { describe, expect, it } from "vitest";
import fixtures from "../../../tests/fixtures/seed-schemas.json";
import SchemaBuilder from "./SchemaBuilder.vue";

type Schema = Record<string, unknown>;
type Props = Record<string, Record<string, unknown>>;

/** Monte le builder sur un schéma et renvoie le schéma qu'il ré-émet. */
async function roundTrip(schema: Schema): Promise<Schema> {
  const w = mount(SchemaBuilder, {
    props: { modelValue: schema },
    global: { plugins: [[PrimeVue, { unstyled: true }]] },
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
