import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import DashboardView from "./DashboardView.vue";

const types = Array.from({ length: 13 }, (_, i) => ({
  type_id: `t${i}`,
  type_name: `Type${i}`,
  count: 13 - i, // l'ordre de l'API (ordre d'affichage) prime sur le nombre d'items
}));
vi.mock("@/api/client", () => ({
  statsApi: {
    get: () =>
      Promise.resolve({
        items_by_type: types,
        total_items: 78,
        total_collections: 0,
        total_relations: 0,
      }),
  },
  collectionApi: { list: () => Promise.resolve({ data: [], total: 0 }) },
  itemApi: { list: () => Promise.resolve({ data: [], total: 0 }) },
  itemTypeApi: { list: () => Promise.resolve({ data: [], total: 0 }) },
  relationTypeApi: { list: vi.fn() },
  collectionTypeApi: { list: vi.fn() },
  extractApiError: () => ({ message: "" }),
}));

describe("DashboardView — items par type", () => {
  beforeEach(() => setActivePinia(createPinia()));

  it("montre les 10 premiers types dans l'ordre d'affichage, puis tout à la demande", async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
    });
    const w = mount(DashboardView, {
      global: { plugins: [ElementPlus, router] },
    });
    await flushPromises();
    const rows = () => w.findAll('a[href^="/items?item_type_id="]');
    expect(rows()).toHaveLength(10);
    expect(rows().map((r) => r.attributes("href"))).toEqual(
      types.slice(0, 10).map((t) => `/items?item_type_id=${t.type_id}`),
    );
    expect(w.text()).not.toContain("Type10");
    await w.find('[data-test="types-toggle"]').trigger("click");
    expect(rows()).toHaveLength(13);
  });
});
