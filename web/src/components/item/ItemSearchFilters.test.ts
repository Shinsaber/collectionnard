import { mount } from "@vue/test-utils";
import ElementPlus, { ElInput, ElInputNumber, ElSelect } from "element-plus";
import { describe, expect, it } from "vitest";
import fixtures from "../../../tests/fixtures/seed-schemas.json";
import ItemSearchFilters from "./ItemSearchFilters.vue";

type Schema = Record<string, unknown>;
const schemaOf = (name: string) =>
  fixtures.find((f) => f.name === name)!.schema as Schema;

function mountFilters(schema: Schema, modelValue = {}) {
  return mount(ItemSearchFilters, {
    props: { schema, modelValue },
    global: { plugins: [ElementPlus] },
  });
}

/** Dernière valeur émise par `update:modelValue`. */
function lastEmitted(w: ReturnType<typeof mountFilters>): unknown {
  const events = w.emitted("update:modelValue")!;
  return events[events.length - 1]![0];
}

describe("ItemSearchFilters", () => {
  it("génère un contrôle par champ filtrable, libellés x-labels, sans la galerie", () => {
    const w = mountFilters(schemaOf("Series"));
    const labels = w.findAll("label").map((l) => l.text());
    expect(labels).toEqual([
      "Titre",
      "Synopsis",
      "Année de début",
      "Année de fin",
    ]);
    expect(w.findAllComponents(ElInputNumber)).toHaveLength(4); // 2 intervalles
  });

  it("traduit les valeurs d'enum et émet un filtre répétable", async () => {
    const w = mountFilters(schemaOf("Broadcast"));
    const select = w.findComponent(ElSelect);
    expect(select.props("multiple")).toBe(true);
    await select.vm.$emit("update:modelValue", ["tv", "cable"]);
    expect(lastEmitted(w)).toEqual({
      "attr.type": ["tv", "cable"],
    });
  });

  it("émet une borne numérique et n'émet le texte qu'à la validation", async () => {
    const w = mountFilters(schemaOf("Series"));
    await w
      .findAllComponents(ElInputNumber)[0]!
      .vm.$emit("update:modelValue", 2008);
    expect(lastEmitted(w)).toEqual({
      "attr.start_year.gte": ["2008"],
    });

    const title = w.findAllComponents(ElInput)[0]!;
    await title.vm.$emit("update:modelValue", "bad");
    expect(w.emitted("update:modelValue")).toHaveLength(1);
    await title.vm.$emit("change", "bad");
    expect(lastEmitted(w)).toEqual({
      "attr.start_year.gte": ["2008"],
      "attr.title.contains": ["bad"],
    });
  });
});
