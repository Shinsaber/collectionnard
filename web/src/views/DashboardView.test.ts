import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import DashboardView from "./DashboardView.vue";

const types = Array.from({ length: 13 }, (_, i) => ({
  type_id: `t${i}`,
  type_name: `Type${i}`,
  count: i, // le plus fourni est le dernier de la liste de l'API
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

  it("montre les 10 types les plus fournis, puis tout à la demande", async () => {
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
    expect(rows()[0]!.text()).toContain("Type12");
    expect(w.text()).not.toContain("Type2");
    await w.find('[data-test="types-toggle"]').trigger("click");
    expect(rows()).toHaveLength(13);
  });
});
