import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ImportReport } from "@/api/types";

const importCollection = vi.fn();
vi.mock("@/api/client", () => ({
  importExportApi: {
    importCollection: (...args: unknown[]) => importCollection(...args),
  },
  extractApiError: () => ({ message: "" }),
}));

import CollectionImportDialog from "./CollectionImportDialog.vue";

function report(partial: Partial<ImportReport>): ImportReport {
  return {
    dry_run: true,
    applied: false,
    counts: {
      collection: { created: 0, updated: 0, unchanged: 1 },
      items: { created: 1, updated: 0, unchanged: 2, removed: 3 },
    },
    errors: [],
    ...partial,
  };
}

async function mountWithFile() {
  const w = mount(CollectionImportDialog, {
    props: { visible: true },
    global: {
      plugins: [ElementPlus],
      // ElDialog se téléporte dans <body> : rendu en place pour le test.
      stubs: {
        ElDialog: { template: '<div><slot /><slot name="footer" /></div>' },
      },
    },
  });
  const input = w.find('input[type="file"]');
  const file = new File(["{}"], "mes-dvd.json", { type: "application/json" });
  Object.defineProperty(input.element, "files", { value: [file] });
  await input.trigger("change");
  return w;
}

function buttonByText(w: ReturnType<typeof mount>, text: string) {
  return w.findAll("button").find((b) => b.text().includes(text));
}

describe("CollectionImportDialog", () => {
  beforeEach(() => importCollection.mockReset());

  it("simule l'import et affiche les items retirés", async () => {
    importCollection.mockResolvedValue(report({}));
    const w = await mountWithFile();
    await buttonByText(w, "Simuler")!.trigger("click");
    await flushPromises();
    expect(importCollection).toHaveBeenCalledWith(expect.any(File), true);
    const table = w.find('[data-test="import-report"]');
    expect(table.text()).toContain("Retirés");
    expect(table.text()).toContain("3");
    expect(buttonByText(w, "Importer")).toBeTruthy();
    w.unmount();
  });

  it("ne propose pas l'import quand la simulation a des erreurs", async () => {
    importCollection.mockResolvedValue(
      report({
        errors: [
          {
            entity: "items",
            index: 1,
            message: "unknown item_id (not in the catalogue)",
          },
        ],
      }),
    );
    const w = await mountWithFile();
    await buttonByText(w, "Simuler")!.trigger("click");
    await flushPromises();
    expect(w.find('[data-test="import-errors"]').text()).toContain(
      "unknown item_id",
    );
    expect(buttonByText(w, "Importer")).toBeUndefined();
    w.unmount();
  });
});
