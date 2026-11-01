import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter, type Router } from "vue-router";
import type { Item, ItemType } from "@/api/types";
import { i18n } from "@/i18n";
import ItemsView from "./ItemsView.vue";

// Audit Q-005 : la page du catalogue (état dans l'URL, liste, filtres, pagination) n'avait aucun test.
const api = vi.hoisted(() => ({ list: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({
  itemApi: api,
  importExportApi: { exportItemsCsv: vi.fn() },
  extractApiError: () => ({ message: "" }),
}));
vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({ canEditCatalog: true, isAdmin: false }),
}));

const volumeType = {
  id: "t1",
  name: "Volume",
  attributes_schema: {
    type: "object",
    "x-label": "{title}",
    "x-title": { fr: "Tome" },
    "x-labels": { fr: { title: "Titre", year: "Année" } },
    properties: { title: { type: "string" }, year: { type: "integer" } },
  },
} as unknown as ItemType;
vi.mock("@/stores/metaTypes", () => ({
  useMetaTypes: () => ({
    itemTypes: [volumeType],
    loadItemTypes: () => Promise.resolve(),
  }),
}));

const item = {
  id: "e1",
  item_type_id: "t1",
  item_type: volumeType,
  attributes: { title: "Breaking Bad" },
  tags: [],
  created_at: "2026-10-11T00:00:00Z",
  updated_at: "2026-10-11T00:00:00Z",
} as unknown as Item;

function page(total: number) {
  return {
    data: [item],
    total,
    page: 1,
    per_page: 20,
    total_pages: Math.ceil(total / 20),
  };
}

const stubs = {
  ItemEditDialog: true,
  CatalogFilters: true,
  RelatedToTag: true,
  ExportButton: true,
  ScanButton: true,
  PageHeader: {
    props: ["title", "meta"],
    template:
      '<div><h1>{{ title }}</h1><p data-test="meta">{{ meta }}</p><slot name="actions" /></div>',
  },
};

async function mountCatalog(url: string) {
  const router: Router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: "/items", name: "items", component: ItemsView },
      { path: "/:any(.*)*", component: { template: "<div />" } },
    ],
  });
  await router.push(url);
  await router.isReady();
  const w = mount(ItemsView, {
    global: { plugins: [ElementPlus, router], stubs },
  });
  await flushPromises();
  return { w, router };
}

const t = i18n.global.t;

describe("ItemsView", () => {
  beforeEach(() => {
    api.list.mockReset().mockResolvedValue(page(1));
  });

  it("charge la liste d'après l'URL et affiche résultats, compteur et puce du type", async () => {
    const { w } = await mountCatalog("/items?search=bad&item_type_id=t1");
    expect(api.list).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 1,
        per_page: 20,
        search: "bad",
        item_type_id: "t1",
      }),
    );
    expect(w.text()).toContain("Breaking Bad");
    expect(w.find('[data-test="meta"]').text()).toBe(t("items.list.count", 1));
    expect(w.text()).toContain(t("items.search.typeChip", { type: "Tome" }));
    // Le champ de recherche reprend la valeur de l'URL ; le tri par pertinence est proposé.
    expect((w.find("input").element as HTMLInputElement).value).toBe("bad");
  });

  it("une nouvelle recherche met l'URL à jour et recharge la liste", async () => {
    const { w, router } = await mountCatalog("/items");
    const input = w.find("input");
    await input.setValue("walter");
    await input.trigger("keyup.enter");
    await flushPromises();
    expect(router.currentRoute.value.query).toEqual({ search: "walter" });
    expect(api.list).toHaveBeenLastCalledWith(
      expect.objectContaining({ search: "walter" }),
    );
    const last = api.list.mock.calls[api.list.mock.calls.length - 1]?.[0];
    expect(last).not.toHaveProperty("item_type_id");
  });

  it("retirer la puce du type enlève le type et ses filtres d'attributs de l'URL", async () => {
    const { w, router } = await mountCatalog(
      "/items?item_type_id=t1&attr.year=2008&tag=drame",
    );
    expect(api.list).toHaveBeenCalledWith(
      expect.objectContaining({
        item_type_id: "t1",
        "attr.year": "2008",
        tag: "drame",
      }),
    );
    const label = t("items.search.typeChip", { type: "Tome" });
    const remove = w
      .findAll("button[aria-label]")
      .find(
        (b) =>
          b.attributes("aria-label") ===
          t("items.search.removeFilter", { label }),
      );
    expect(remove).toBeDefined();
    await remove!.trigger("click");
    await flushPromises();
    expect(router.currentRoute.value.query).toEqual({ tag: "drame" });
    expect(api.list).toHaveBeenLastCalledWith(
      expect.objectContaining({ tag: "drame" }),
    );
  });

  it("la pagination passe par l'URL", async () => {
    api.list.mockResolvedValue(page(45));
    const { w, router } = await mountCatalog("/items");
    const second = w.findAll(".el-pager li").find((li) => li.text() === "2");
    expect(second).toBeDefined();
    await second!.trigger("click");
    await flushPromises();
    expect(router.currentRoute.value.query).toEqual({ page: "2" });
    expect(api.list).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 2 }),
    );
  });
});
