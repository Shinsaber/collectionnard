import { mount } from "@vue/test-utils";
import ElementPlus, { ElInput, ElSelect, ElSlider } from "element-plus";
import { describe, expect, it } from "vitest";
import fixtures from "../../../tests/fixtures/seed-schemas.json";
import DynamicForm from "./DynamicForm.vue";

type Schema = Record<string, unknown>;

const serieSchema = fixtures.find((f) => f.name === "Series")!.schema as Schema;

function mountForm(schema: Schema, modelValue: Record<string, unknown> = {}) {
  return mount(DynamicForm, {
    props: { schema, modelValue },
    global: { plugins: [ElementPlus] },
  });
}

/** Dernière valeur émise par `update:modelValue`. */
function lastEmitted(w: ReturnType<typeof mountForm>): unknown {
  const events = w.emitted("update:modelValue")!;
  return events[events.length - 1]![0];
}

function labels(wrapper: ReturnType<typeof mountForm>): string[] {
  return wrapper.findAll('label[id$="-label"]').map((l) =>
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
      "Titre original",
      "Pays d'origine",
      "Langue originale",
      "Saisons",
      "Épisodes",
      "Images",
      "Liens",
      "Identifiants externes",
      "Sources",
    ]);
    // titre est requis → astérisque
    const titre = w.findAll('label[id$="-label"]')[0]!;
    expect(titre.find('span[aria-hidden="true"]').text()).toBe("*");
    // synopsis : widget i18n-text → deux zones de texte FR / EN ; pays, identifiants et sources : JSON
    expect(w.findAll("textarea")).toHaveLength(5);
    // images : galerie avec les 3 catégories déclarées dans x-image-categories (dans l'ordre de
    // l'objet, celui que PostgreSQL rend : les clés d'un JSONB sont triées)
    const cats = w
      .findAll('input[type="file"]')
      .map((i) => i.attributes("aria-label"));
    expect([...cats].sort()).toEqual([
      "Fichier image pour banner",
      "Fichier image pour poster",
      "Fichier image pour screenshot",
    ]);
  });

  it("texte multilingue : un onglet par langue, ● si renseignée, une seule zone visible", async () => {
    const w = mountForm(serieSchema, {
      synopsis: { fr: "Un professeur de chimie…" },
    });
    const tabs = w.findAll('[role="tab"]');
    expect(tabs.map((b) => b.text())).toEqual(["● fr", "○ en"]);
    expect(tabs[0]!.attributes("aria-selected")).toBe("true");
    const panels = w.findAll('[role="tabpanel"]');
    expect(panels[0]!.attributes("style") ?? "").not.toContain("display: none");
    expect(panels[1]!.attributes("style")).toContain("display: none");

    await tabs[1]!.trigger("click");
    expect(tabs[1]!.attributes("aria-selected")).toBe("true");
    expect(panels[1]!.attributes("style") ?? "").not.toContain("display: none");
    expect(panels[0]!.attributes("style")).toContain("display: none");
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
    expect(w.findComponent(ElSelect).exists()).toBe(true);
    expect(w.findComponent(ElSlider).exists()).toBe(true);
    expect(w.findComponent(ElInput).exists()).toBe(true);
  });

  it("émet update:modelValue en fusionnant la valeur modifiée", async () => {
    const w = mountForm(
      { type: "object", properties: { titre: { type: "string" } } },
      { titre: "Avant", autre: 1 },
    );
    await w.findComponent(ElInput).vm.$emit("update:modelValue", "Après");
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

  it("vérifie un motif à la sortie du champ, puis à chaque saisie ; signale un JSON invalide", async () => {
    const w = mountForm(
      {
        type: "object",
        properties: {
          code: {
            type: "string",
            pattern: "^[0-9]{3}$",
            description: "trois chiffres",
          },
          ids: {
            type: "object",
            additionalProperties: false,
            properties: { wd: { type: "string", pattern: "^Q[0-9]+$" } },
          },
        },
      },
      { code: "12a" },
    );
    const code = w.find("input");
    expect(w.text()).not.toContain("Format attendu");
    await code.trigger("focus");
    await code.trigger("blur");
    expect(w.text()).toContain("Format attendu : trois chiffres");
    // En erreur : revérifié à chaque saisie, le message disparaît dès que la valeur convient.
    await code.setValue("123");
    expect(w.text()).not.toContain("trois chiffres");

    const json = w.find("textarea");
    await json.setValue('{"wd": ');
    expect(w.text()).toContain("JSON invalide");
    expect((json.element as HTMLTextAreaElement).value).toBe('{"wd": '); // saisie gardée
    expect(
      w.emitted("update:modelValue")!.some((e) => "ids" in (e[0] as object)),
    ).toBe(false);
    await json.setValue('{"wd": "x"}');
    expect(w.text()).toContain("wd : Format attendu : ^Q[0-9]+$");

    const vm = w.vm as unknown as { validate: () => boolean };
    await w.setProps({ modelValue: { code: "123", ids: { wd: "Q1" } } });
    await json.setValue('{"wd": "Q1"}');
    expect(vm.validate()).toBe(true);
  });

  it("champ liens : ajoute, vérifie l'adresse et retire une ligne", async () => {
    const publisher = fixtures.find((f) => f.name === "Publisher")!
      .schema as Schema;
    const w = mountForm(publisher, {
      name: "Nintendo",
      links: [
        { label: "Wikipédia", url: "https://fr.wikipedia.org/wiki/Nintendo" },
      ],
    });
    expect(w.findAll('[data-test="link-row"]')).toHaveLength(1);
    const add = w
      .findAll("button")
      .find((b) => b.text() === "Ajouter un lien")!;
    await add.trigger("click");
    const added = lastEmitted(w) as Record<string, unknown>;
    expect(added.links).toEqual([
      { label: "Wikipédia", url: "https://fr.wikipedia.org/wiki/Nintendo" },
      { url: "" },
    ]);

    // Adresse invalide : signalée avant l'envoi, avec la ligne concernée.
    await w.setProps({
      modelValue: { name: "Nintendo", links: [{ url: "nintendo.com" }] },
    });
    const vm = w.vm as unknown as { validate: () => boolean };
    expect(vm.validate()).toBe(false);
    await w.vm.$nextTick();
    expect(w.text()).toContain("1.url : URL invalide");

    // Retirer le dernier lien : champ absent (pas de tableau vide).
    await w.find('[aria-label="Retirer le lien 1"]').trigger("click");
    expect((lastEmitted(w) as Record<string, unknown>).links).toBe(undefined);
  });

  it("affiche un message quand le schéma n'a aucun attribut", () => {
    const w = mountForm({ type: "object" });
    expect(w.text()).toContain("Aucun attribut défini");
  });
});

describe("DynamicForm — tableau d'enum (rôles)", () => {
  const rolesSchema = fixtures.find((f) => f.name === "created_by")!
    .schema as Schema;

  it("rend une sélection multiple ; la vider rend le champ absent", async () => {
    const w = mountForm(rolesSchema, { roles: ["writer", "artist"] });
    const select = w.findComponent(ElSelect);
    expect(select.props("multiple")).toBe(true);
    expect(select.props("modelValue")).toEqual(["writer", "artist"]);

    await select.vm.$emit("update:modelValue", []);
    const emitted = w.emitted("update:modelValue")!;
    expect(emitted[emitted.length - 1]![0]).toEqual({ roles: undefined });
  });
});
