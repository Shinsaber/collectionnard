import { mount } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import InputText from "primevue/inputtext";
import Select from "primevue/select";
import Slider from "primevue/slider";
import Textarea from "primevue/textarea";
import { describe, expect, it } from "vitest";
import fixtures from "../../../tests/fixtures/seed-schemas.json";
import DynamicForm from "./DynamicForm.vue";

type Schema = Record<string, unknown>;

const serieSchema = fixtures.find((f) => f.name === "Series")!.schema as Schema;

function mountForm(schema: Schema, modelValue: Record<string, unknown> = {}) {
  return mount(DynamicForm, {
    props: { schema, modelValue },
    global: { plugins: [[PrimeVue, { unstyled: true }]] },
  });
}

function labels(wrapper: ReturnType<typeof mountForm>): string[] {
  return wrapper.findAll("label.text-sm").map((l) =>
    l
      .text()
      .replace(/\s*\*$/, "")
      .trim(),
  );
}

describe("DynamicForm", () => {
  it("rend les champs du schéma Series avec les libellés x-labels (fr) et les widgets", () => {
    const w = mountForm(serieSchema);

    expect(labels(w)).toEqual([
      "Titre",
      "Synopsis",
      "Année de début",
      "Année de fin",
      "Images",
    ]);
    // titre est requis → astérisque
    const titre = w.findAll("label.text-sm")[0]!;
    expect(titre.find("span.text-red-500").exists()).toBe(true);
    // synopsis : widget i18n-text → deux zones de texte FR / EN
    expect(w.findAllComponents(Textarea)).toHaveLength(2);
    // images : galerie avec les 3 catégories déclarées dans x-image-categories
    const cats = w.findAll("label.capitalize").map((l) => l.text());
    expect(cats).toEqual(["poster", "screenshot", "banner"]);
  });

  it("respecte x-order puis ajoute les clés restantes", () => {
    const w = mountForm({
      type: "object",
      "x-order": ["b", "a"],
      properties: {
        a: { type: "string" },
        b: { type: "string" },
        c_long_name: { type: "string" },
      },
    });
    expect(labels(w)).toEqual(["B", "A", "C Long Name"]);
  });

  it("choisit le widget selon le type : enum → Select, min+max → Slider, texte → InputText", () => {
    const w = mountForm({
      type: "object",
      properties: {
        kind: { type: "string", enum: ["TV", "web"] },
        note: { type: "integer", minimum: 0, maximum: 10 },
        name: { type: "string" },
      },
    });
    expect(w.findComponent(Select).exists()).toBe(true);
    expect(w.findComponent(Slider).exists()).toBe(true);
    expect(w.findComponent(InputText).exists()).toBe(true);
  });

  it("émet update:modelValue en fusionnant la valeur modifiée", async () => {
    const w = mountForm(
      { type: "object", properties: { titre: { type: "string" } } },
      { titre: "Avant", autre: 1 },
    );
    await w.findComponent(InputText).vm.$emit("update:modelValue", "Après");
    const emitted = w.emitted("update:modelValue");
    expect(emitted).toBeTruthy();
    expect(emitted![0]![0]).toEqual({ titre: "Après", autre: 1 });
  });

  it("validate() signale les champs requis manquants et les URL invalides", async () => {
    const w = mountForm(
      {
        type: "object",
        required: ["titre"],
        properties: {
          titre: { type: "string" },
          site: { type: "string", format: "uri" },
        },
      },
      { site: "pas une url" },
    );
    const vm = w.vm as unknown as {
      validate: () => boolean;
      setErrors: (e: Record<string, string>) => void;
    };
    expect(vm.validate()).toBe(false);
    await w.vm.$nextTick();
    const texts = w.text();
    expect(texts).toContain("Ce champ est requis");
    expect(texts).toContain("URL invalide");

    await w.setProps({
      modelValue: { titre: "Ok", site: "https://example.org" },
    });
    expect(vm.validate()).toBe(true);

    vm.setErrors({ titre: "déjà pris" });
    await w.vm.$nextTick();
    expect(w.text()).toContain("déjà pris");
  });

  it("affiche un message quand le schéma n'a aucun attribut", () => {
    const w = mountForm({ type: "object" });
    expect(w.text()).toContain("Aucun attribut défini");
  });
});
