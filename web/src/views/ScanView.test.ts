import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { Item } from "@/api/types";
import BarcodeScannerDialog from "@/components/common/BarcodeScannerDialog.vue";
import ScanView from "./ScanView.vue";

const api = vi.hoisted(() => ({
  list: vi.fn(),
  collections: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  itemApi: api,
  extractApiError: () => ({ message: "" }),
}));
vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({ canEditCatalog: true }),
}));
vi.mock("@/stores/metaTypes", () => ({
  useMetaTypes: () => ({
    itemTypes: [],
    loadItemTypes: () => Promise.resolve(),
  }),
}));

const volume = {
  id: "i1",
  item_type_id: "t1",
  attributes: { title: "Tome 1", isbn: "9782505000945" },
  tags: [],
  item_type: {
    id: "t1",
    name: "Volume",
    attributes_schema: {
      type: "object",
      "x-label": "{title}",
      "x-title": { fr: "Tome" },
      properties: {
        title: { type: "string" },
        isbn: { type: "string", "x-identifier": true },
      },
    },
  },
} as unknown as Item;

const stubs = {
  BarcodeScannerDialog: true,
  ItemCover: true,
  AddToCollectionDialog: true,
  ItemEditDialog: true,
};

async function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: "/:any(.*)*",
        name: "external-import",
        component: { template: "<div />" },
      },
    ],
  });
  const w = mount(ScanView, {
    global: { plugins: [ElementPlus, router], stubs },
  });
  await flushPromises();
  return w;
}

async function scan(w: Awaited<ReturnType<typeof mountView>>, code: string) {
  w.findComponent(BarcodeScannerDialog).vm.$emit("detected", code, "ean_13");
  await flushPromises();
}

beforeEach(() => {
  vi.clearAllMocks();
  api.collections.mockResolvedValue([
    { collection: { id: "c1", name: "Mes BD" }, attributes: {} },
  ]);
});

describe("ScanView", () => {
  it("affiche l'item trouvé, ses collections et le résultat en direct", async () => {
    api.list.mockResolvedValue({ data: [volume], total: 1 });
    const w = await mountView();
    expect(w.text()).toContain("Aucun code scanné");

    await scan(w, "9782505000945");
    expect(api.list).toHaveBeenCalledWith({
      search: "9782505000945",
      per_page: 10,
    });
    const entries = w.find('[data-test="scan-entries"]');
    expect(entries.text()).toContain("9782505000945");
    expect(entries.text()).toContain("Au catalogue");
    expect(entries.text()).toContain("Tome 1");
    expect(entries.text()).toContain("Dans 1 de mes collections");
    expect(w.text()).toContain("1 code");
  });

  it("propose les sources externes et la création pour un code inconnu", async () => {
    api.list.mockResolvedValue({ data: [], total: 0 });
    const w = await mountView();
    await scan(w, "5099902988216");
    const entries = w.find('[data-test="scan-entries"]');
    expect(entries.text()).toContain("Absent du catalogue");
    expect(entries.text()).toContain("Chercher dans les sources externes");
    expect(entries.text()).toContain("Créer l'item");
    expect(api.collections).not.toHaveBeenCalled();
  });

  it("remonte un code déjà lu en tête sans nouvelle requête", async () => {
    api.list.mockResolvedValue({ data: [], total: 0 });
    const w = await mountView();
    await scan(w, "1111111111116");
    await scan(w, "2222222222222");
    await scan(w, "1111111111116");
    expect(api.list).toHaveBeenCalledTimes(2);
    const codes = w
      .findAll('[data-test="scan-entries"] li .font-mono')
      .map((n) => n.text());
    expect(codes).toEqual(["1111111111116", "2222222222222"]);
    expect(w.text()).toContain("2 codes");
  });
});
