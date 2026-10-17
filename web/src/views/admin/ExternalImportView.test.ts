import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import { useLang } from "@/composables/useLang";
import type {
  ExternalEditionHit,
  ExternalImportReport,
  ExternalWorkHit,
} from "@/api/types";
import fixtures from "../../../tests/fixtures/seed-schemas.json";
import { dragonBallPlan, MANGA } from "../../../tests/fixtures/external-plan";
import ExternalImportView from "./ExternalImportView.vue";

const api = vi.hoisted(() => ({
  connectors: vi.fn(async () => [{ id: "bd-manga", domain: "books" }]),
  searchWorks: vi.fn(),
  searchEditions: vi.fn(),
  preview: vi.fn(),
  importSelection: vi.fn(),
}));
const page = <T>(data: T[]) => ({
  data,
  total: data.length,
  page: 1,
  per_page: 100,
  total_pages: 1,
});
vi.mock("@/api/client", () => ({
  externalApi: api,
  extractApiError: () => ({ message: "" }),
  itemTypeApi: {
    list: async () =>
      page(
        fixtures
          .filter((f) => f.id.startsWith("b"))
          .map((f) => ({
            id: f.id,
            name: f.name,
            attributes_schema: f.schema,
          })),
      ),
  },
  relationTypeApi: {
    list: async () =>
      page(
        fixtures
          .filter((f) => f.id.startsWith("c"))
          .map((f) => ({
            id: f.id,
            name: f.name,
            attributes_schema: f.schema,
          })),
      ),
  },
  collectionTypeApi: { list: async () => page([]) },
}));
vi.mock("@/composables/useConfirmDialog", () => ({
  useConfirmDialog: () => ({ confirm: async () => true }),
}));

const work: ExternalWorkHit = {
  wikidata_id: "Q15051307",
  label: "Dragon Ball",
  description: "série de manga",
  item_type_id: MANGA,
  start_year: 1984,
  source_url: "https://www.wikidata.org/wiki/Q15051307",
};
const edition: ExternalEditionHit = {
  bnf_ark: "ark:/12148/cb343384394",
  title: "Dragon Ball",
  publishers: ["Glénat"],
  start_year: 1993,
  source_url: "https://catalogue.bnf.fr/ark:/12148/cb343384394",
};

function report(dryRun: boolean): ExternalImportReport {
  return {
    dry_run: dryRun,
    applied: !dryRun,
    counts: {
      items: { created: 4, updated: 1, unchanged: 1 },
      relations: { created: 2, updated: 0, unchanged: 0 },
    },
    errors: [],
    item_ids: { "wikidata:Q15051307": "e0000000-0000-0000-0000-000000000900" },
  };
}

function button(w: ReturnType<typeof mount>, text: string) {
  const b = w.findAll("button").find((x) => x.text().includes(text));
  if (!b) throw new Error(`no button « ${text} »`);
  return b;
}

describe("ExternalImportView", () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    Object.values(api).forEach((m) => m.mockReset());
    useLang().setLang("fr");
    await nextTick();
  });

  it("recherche, aperçu, simulation puis import de la sélection", async () => {
    api.searchWorks.mockResolvedValue([work]);
    api.searchEditions.mockResolvedValue([edition]);
    api.preview.mockResolvedValue(dragonBallPlan);
    api.importSelection.mockImplementation(
      async (_connector: string, _req: unknown, dryRun: boolean) =>
        report(dryRun),
    );
    const w = mount(ExternalImportView, {
      global: { plugins: [ElementPlus], stubs: { RouterLink: true } },
    });
    await flushPromises();
    expect(button(w, "Aperçu").attributes("disabled")).toBeDefined();

    // 1. Œuvre : recherche puis choix de la ligne ; la recherche d'éditions suit.
    await w.findAll("input")[0]!.setValue("dragon ball");
    await w.findAll("form")[0]!.trigger("submit");
    await flushPromises();
    expect(api.searchWorks).toHaveBeenCalledWith("bd-manga", "dragon ball");
    const workRow = w.find('[data-test="work-hits"] tbody tr');
    expect(workRow.text()).toContain("Manga"); // titre du type (x-title)
    await workRow.trigger("click");
    await flushPromises();
    expect(w.find('[data-test="chosen-work"]').text()).toContain("Dragon Ball");
    expect(api.searchEditions).toHaveBeenCalledWith("bd-manga", {
      q: "Dragon Ball",
    });

    // Édition cochée, puis aperçu.
    await w
      .find('[data-test="edition-hits"] input[type="checkbox"]')
      .setValue(true);
    expect(w.find('[data-test="chosen-editions"]').text()).toContain("Glénat");
    await button(w, "Aperçu").trigger("click");
    await flushPromises();
    expect(api.preview).toHaveBeenCalledWith("bd-manga", {
      work: "Q15051307",
      editions: ["ark:/12148/cb343384394"],
    });

    // 2. Sélection : libellés x-label, rôles traduits, états, réimpression décochée.
    const plan = w.find('[data-test="external-plan"]').text();
    expect(plan).toContain("Akira Toriyama");
    expect(plan).toContain("Auteur, Dessinateur");
    expect(plan).toContain("Édition : Dragon Ball");
    expect(plan).toContain("1. Sangoku");
    expect(plan).toContain("Réimpression");
    expect(plan).toContain("À compléter");
    expect(plan).toContain("Licence Ouverte");
    expect(w.find('[data-test="plan-count"]').text()).toContain(
      "Sélection : 6 / 7 (dont 5 à écrire)",
    );

    // Décocher tous les tomes, puis simuler.
    await button(w, "Tout décocher").trigger("click");
    await button(w, "Simuler").trigger("click");
    await flushPromises();
    const [connector, req, dryRun] = api.importSelection.mock.calls[0]!;
    expect(connector).toBe("bd-manga");
    expect(dryRun).toBe(true);
    expect([...req.selected].sort()).toEqual([
      "bnf:cb343384394",
      "publisher:glénat",
      "wikidata:Q15051307",
      "wikidata:Q208582",
    ]);
    expect(w.find('[data-test="import-report"]').text()).toContain(
      "Résultat de la simulation",
    );

    // 3. Import (confirmation acceptée) puis lien vers l'œuvre.
    await button(w, "Importer").trigger("click");
    await flushPromises();
    expect(api.importSelection.mock.calls[1]![2]).toBe(false);
    expect(w.find('[data-test="import-report"]').text()).toContain(
      "Résultat de l'import",
    );
    expect(w.html()).toContain("/items/e0000000-0000-0000-0000-000000000900");
    // Les 4 étapes passent au vert.
    expect(
      w.findAll('[data-test="steps"] .el-step__head.is-success'),
    ).toHaveLength(4);
  });

  it("« Retour » depuis la sélection retrouve les résultats, sans relancer la recherche", async () => {
    api.searchWorks.mockResolvedValue([work]);
    api.searchEditions.mockResolvedValue([edition]);
    api.preview.mockResolvedValue(dragonBallPlan);
    const w = mount(ExternalImportView, {
      global: { plugins: [ElementPlus], stubs: { RouterLink: true } },
    });
    await flushPromises();
    await w.findAll("input")[0]!.setValue("dragon ball");
    await w.findAll("form")[0]!.trigger("submit");
    await flushPromises();
    await w.find('[data-test="work-hits"] tbody tr').trigger("click");
    await flushPromises();
    await button(w, "Aperçu").trigger("click");
    await flushPromises();
    expect(w.find('[data-test="external-plan"]').exists()).toBe(true);

    await button(w, "Retour").trigger("click");
    await flushPromises();
    expect(w.find('[data-test="external-plan"]').exists()).toBe(false);
    expect(w.findAll('[data-test="work-hits"] tbody tr')).toHaveLength(1);
    expect(w.findAll('[data-test="edition-hits"] tbody tr')).toHaveLength(1);
    expect((w.findAll("input")[0]!.element as HTMLInputElement).value).toBe(
      "dragon ball",
    );
    expect(api.searchWorks).toHaveBeenCalledTimes(1);
    expect(api.searchEditions).toHaveBeenCalledTimes(1);
  });

  it("par ISBN : affiche le tome trouvé dans son édition", async () => {
    api.searchEditions.mockResolvedValue([
      {
        ...edition,
        existing_item_id: "e0000000-0000-0000-0000-000000000108",
        matched_volume: {
          bnf_ark: "ark:/12148/cb356004006",
          number: 1,
          title: "Sangoku",
        },
      },
    ]);
    const w = mount(ExternalImportView, {
      global: { plugins: [ElementPlus], stubs: { RouterLink: true } },
    });
    await flushPromises();
    const form = w.findAll("form")[1]!;
    await form.findAll('input[type="radio"]')[1]!.setValue(true);
    await form.find('input[type="text"]').setValue("978-2-87695-205-8");
    await form.trigger("submit");
    await flushPromises();
    expect(api.searchEditions).toHaveBeenCalledWith("bd-manga", {
      isbn: "978-2-87695-205-8",
    });
    const hits = w.find('[data-test="edition-hits"]').text();
    expect(hits).toContain("Tome 1 : Sangoku");
    expect(hits).toContain("Déjà en base");
  });
});

describe("ExternalImportView — corrections avant la simulation", () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    Object.values(api).forEach((m) => m.mockReset());
    useLang().setLang("fr");
    await nextTick();
  });

  it("corrige un tome, le coche et envoie la correction", async () => {
    api.searchWorks.mockResolvedValue([work]);
    api.searchEditions.mockResolvedValue([]);
    api.preview.mockResolvedValue(dragonBallPlan);
    api.importSelection.mockResolvedValue(report(true));
    const w = mount(ExternalImportView, {
      global: {
        plugins: [ElementPlus],
        stubs: {
          RouterLink: true,
          // ElDialog se téléporte dans <body> : rendu en place pour le test.
          ElDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        },
      },
    });
    await flushPromises();
    await w.findAll("input")[0]!.setValue("dragon ball");
    await w.findAll("form")[0]!.trigger("submit");
    await flushPromises();
    await w.find('[data-test="work-hits"] tbody tr').trigger("click");
    await flushPromises();
    await button(w, "Aperçu").trigger("click");
    await flushPromises();

    // Le dernier tome (réimpression, décochée) : numéro corrigé de 2 à 3.
    const edits = w.findAll('[data-test="edit-node"]');
    expect(edits).toHaveLength(7);
    await edits[6]!.trigger("click");
    await flushPromises();
    const numberLabel = w
      .findAll("label")
      .find((l) => l.text().startsWith("Numéro"))!;
    const input = w.find(`#${numberLabel.attributes("for")}`);
    await input.setValue("3");
    await input.trigger("change");
    await button(w, "Appliquer").trigger("click");
    await flushPromises();

    expect(w.find('[data-test="external-plan"]').text()).toContain("Corrigé");
    expect(w.find('[data-test="external-plan"]').text()).toContain(
      "3. Dragon Ball",
    );
    expect(w.find('[data-test="plan-count"]').text()).toContain(
      "Sélection : 7 / 7 (dont 6 à écrire)",
    );

    await button(w, "Simuler").trigger("click");
    await flushPromises();
    const [, req] = api.importSelection.mock.calls[0]!;
    expect(req.selected).toContain("bnf:cb43462118n");
    expect(req.overrides).toEqual({
      "bnf:cb43462118n": { number: 3, title: { fr: "Dragon Ball" } },
    });
  });
});
