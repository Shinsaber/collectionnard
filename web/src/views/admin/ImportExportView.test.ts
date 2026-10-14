import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import { useLang } from "@/composables/useLang";
import type { ImportReport } from "@/api/types";
import ImportExportView from "./ImportExportView.vue";

const { importMock } = vi.hoisted(() => ({ importMock: vi.fn() }));
vi.mock("@/api/client", () => ({
  importExportApi: { importCatalog: importMock, exportCatalog: vi.fn() },
  extractApiError: () => ({ message: "" }),
}));

async function mountWithFile() {
  const wrapper = mount(ImportExportView, {
    global: { plugins: [ElementPlus] },
  });
  const input = wrapper.find('input[type="file"]');
  const file = new File(["{}"], "catalog.json", { type: "application/json" });
  Object.defineProperty(input.element, "files", { value: [file] });
  await input.trigger("change");
  return { wrapper, file };
}

function simulateButton(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll("button").find((b) => b.text().includes("Simuler"))!;
}

describe("ImportExportView", () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    importMock.mockReset();
    useLang().setLang("fr");
    await nextTick();
  });

  it("affiche les compteurs d'une simulation sans erreur et propose l'import", async () => {
    const report: ImportReport = {
      dry_run: true,
      applied: false,
      counts: {
        item_types: { created: 2, updated: 1, unchanged: 0 },
        items: { created: 12, updated: 0, unchanged: 3 },
      },
      errors: [],
    };
    importMock.mockResolvedValue(report);
    const { wrapper, file } = await mountWithFile();

    await simulateButton(wrapper).trigger("click");
    await flushPromises();

    expect(importMock).toHaveBeenCalledWith(file, true);
    const text = wrapper.find('[data-test="import-report"]').text();
    expect(text).toContain("Résultat de la simulation");
    expect(text).toContain("Types d'items");
    expect(text).toContain("Types de relations");
    expect(text).toContain("12");
    expect(text).toContain("Simulation sans erreur");
    expect(wrapper.find('[data-test="import-errors"]').exists()).toBe(false);
    expect(
      wrapper.findAll("button").some((b) => b.text().includes("Importer")),
    ).toBe(true);
  });

  it("affiche les erreurs (entité traduite, position, message brut) sans proposer l'import", async () => {
    importMock.mockResolvedValue({
      dry_run: true,
      applied: false,
      counts: {},
      errors: [
        {
          entity: "items",
          index: 4,
          id: "e0000000-0000-0000-0000-000000000005",
          message: "attributes: missing property 'title'",
        },
        { entity: "file", index: -1, message: "unsupported format" },
      ],
    } satisfies ImportReport);
    const { wrapper } = await mountWithFile();

    await simulateButton(wrapper).trigger("click");
    await flushPromises();

    expect(wrapper.find(".el-alert--error").text()).toContain("2 erreurs");
    const errors = wrapper.find('[data-test="import-errors"]').text();
    expect(errors).toContain("Items");
    expect(errors).toContain("5");
    expect(errors).toContain("e0000000-0000-0000-0000-000000000005");
    expect(errors).toContain("attributes: missing property 'title'");
    expect(errors).toContain("Fichier");
    expect(errors).toContain("fichier");
    expect(wrapper.findAll("button").some((b) => b.text() === "Importer")).toBe(
      false,
    );
  });

  it("changer de fichier réinitialise le rapport", async () => {
    importMock.mockResolvedValue({
      dry_run: true,
      applied: false,
      counts: {},
      errors: [],
    } satisfies ImportReport);
    const { wrapper } = await mountWithFile();
    await simulateButton(wrapper).trigger("click");
    await flushPromises();
    expect(wrapper.find('[data-test="import-report"]').exists()).toBe(true);

    await wrapper.find('input[type="file"]').trigger("change");
    expect(wrapper.find('[data-test="import-report"]').exists()).toBe(false);
  });
});
