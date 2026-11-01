import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import { createMemoryHistory, createRouter } from "vue-router";
import type { Collection, CollectionItem } from "@/api/types";
import CollectionItemList from "@/components/collection/CollectionItemList.vue";
import CollectionMoreMenu from "@/components/collection/CollectionMoreMenu.vue";
import { i18n } from "@/i18n";
import CollectionDetailView from "./CollectionDetailView.vue";

// Audit Q-005 : la page d'une collection (en-tête, exemplaires, menu « Plus ») n'avait aucun test.
const api = vi.hoisted(() => ({
  get: vi.fn(),
  listItems: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
  updateItem: vi.fn(),
  removeItem: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  collectionApi: api,
  importExportApi: { exportCollection: vi.fn() },
  statsApi: { collection: vi.fn() },
  extractApiError: () => ({ message: "" }),
}));

const schema = {
  type: "object",
  "x-title": { fr: "Possession" },
  "x-labels": { fr: { state: "État" } },
  properties: { state: { type: "string" } },
};
const collection = {
  id: "c1",
  user_id: "u1",
  collection_type_id: "d1",
  collection_type: {
    id: "d1",
    name: "Ownership",
    attributes_schema: schema,
    item_type_ids: [],
    created_at: "2026-10-17T00:00:00Z",
    updated_at: "2026-10-17T00:00:00Z",
  },
  name: "Mes mangas",
  description: "Shōnen surtout",
  item_count: 42,
  visibility: "private",
  hidden_fields: null,
  created_at: "2026-10-17T00:00:00Z",
  updated_at: "2026-10-17T00:00:00Z",
} as Collection;

const exemplaire = {
  id: "ci1",
  collection_id: "c1",
  item_id: "e1",
  item: {
    id: "e1",
    item_type_id: "b1",
    attributes: { title: "Tome 1" },
    tags: [],
    item_type: {
      id: "b1",
      name: "Volume",
      attributes_schema: { "x-label": "{title}" },
    },
  },
  attributes: { state: "neuf" },
  created_at: "2026-10-17T00:00:00Z",
  updated_at: "2026-10-17T00:00:00Z",
} as unknown as CollectionItem;

const stubs = {
  AppSpinner: true,
  CollectionItemList: true,
  CollectionShareDialog: true,
  CollectionAddItemDialog: true,
  CollectionMoreMenu: true,
  DynamicForm: true,
  PageHeader: {
    props: ["title", "meta"],
    template:
      '<div><h1>{{ title }}</h1><p data-test="meta">{{ meta }}</p><slot name="subtitle" /><slot name="actions" /></div>',
  },
};

async function mountDetail() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  const w = mount(CollectionDetailView, {
    props: { id: "c1" },
    global: { plugins: [ElementPlus, router], stubs },
  });
  await flushPromises();
  return w;
}

const t = i18n.global.t;

describe("CollectionDetailView", () => {
  beforeEach(() => {
    Object.values(api).forEach((m) => m.mockReset());
    api.get.mockResolvedValue(collection);
    api.listItems.mockResolvedValue({
      data: [exemplaire],
      total: 1,
      page: 1,
      per_page: 20,
      total_pages: 1,
    });
  });

  it("affiche l'en-tête (nom, type, description, compteur) et confie les exemplaires à la liste", async () => {
    const w = await mountDetail();
    expect(api.get).toHaveBeenCalledWith("c1");
    expect(api.listItems).toHaveBeenCalledWith("c1", { page: 1, per_page: 20 });
    expect(w.find("h1").text()).toBe("Mes mangas");
    expect(w.find('[data-test="meta"]').text()).toBe(
      t("collections.itemCount", 42),
    );
    expect(w.text()).toContain("Possession");
    expect(w.text()).toContain("Shōnen surtout");
    const list = w.findComponent(CollectionItemList);
    expect(list.props("items")).toEqual([exemplaire]);
    expect(list.props("schema")).toEqual(schema);
    // Collection privée : cadenas sur le bouton de partage.
    expect(w.find('[data-test="share-button"] i').classes()).toContain(
      "pi-lock",
    );
  });

  it("modifie nom et description depuis le menu « Plus »", async () => {
    api.update.mockResolvedValue({ ...collection, name: "Mes mangas 2026" });
    const w = await mountDetail();
    w.findComponent(CollectionMoreMenu).vm.$emit("edit");
    await nextTick();
    const name = w.find("#col-name");
    expect((name.element as HTMLInputElement).value).toBe("Mes mangas");
    const save = w.find('[data-test="collection-save"]');
    expect(save.attributes("disabled")).toBeDefined(); // rien de modifié : grisé
    await name.setValue("  Mes mangas 2026 ");
    await nextTick();
    expect(save.attributes("disabled")).toBeUndefined();
    await save.trigger("click");
    await flushPromises();
    expect(api.update).toHaveBeenCalledWith("c1", {
      name: "Mes mangas 2026",
      description: "Shōnen surtout",
    });
    expect(w.find("h1").text()).toBe("Mes mangas 2026");
  });

  it("n'affiche rien d'une collection introuvable", async () => {
    api.get.mockRejectedValue(new Error("404"));
    const w = await mountDetail();
    expect(w.find("h1").exists()).toBe(false);
    expect(api.listItems).not.toHaveBeenCalled();
  });
});
