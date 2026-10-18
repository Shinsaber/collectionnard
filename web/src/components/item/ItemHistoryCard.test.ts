import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import type { HistoryEntry, Item, ItemHistory } from "@/api/types";
import ItemHistoryCard from "./ItemHistoryCard.vue";

const api = vi.hoisted(() => ({ history: vi.fn(), revert: vi.fn() }));
vi.mock("@/api/client", () => ({
  itemApi: api,
  itemTypeApi: { list: vi.fn() },
  relationTypeApi: { list: () => Promise.resolve({ data: [], total: 0 }) },
  collectionTypeApi: { list: vi.fn() },
  extractApiError: () => ({ message: "" }),
}));
vi.mock("@/composables/useConfirmDialog", () => ({
  useConfirmDialog: () => ({ confirm: () => Promise.resolve(true) }),
}));

const item = {
  id: "i1",
  item_type_id: "t1",
  attributes: { title: "B" },
  item_type: {
    id: "t1",
    name: "Volume",
    attributes_schema: {
      type: "object",
      "x-labels": { fr: { title: "Titre" } },
      properties: { title: { type: "string" } },
    },
  },
} as unknown as Item;

const entry = (over: Partial<HistoryEntry>): HistoryEntry => ({
  kind: "item",
  entity_id: "i1",
  version: 1,
  origin: "direct",
  author: {
    kind: "user",
    user: { username: "camille", display_name: "Camille" },
  },
  before: null,
  after: { title: "A" },
  created_at: "2026-10-01T10:00:00Z",
  ...over,
});

const history: ItemHistory = {
  entries: [
    entry({
      version: 2,
      before: { title: "A" },
      after: { title: "B" },
      author: { kind: "member" },
    }),
    entry({}),
  ],
  truncated: false,
  created: entry({}),
};

async function mountCard(role: "user" | "contributor") {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  useAuthStore().user = {
    id: "u",
    username: "u",
    email: "",
    role,
    created_at: "",
    updated_at: "",
  };
  const w = mount(ItemHistoryCard, {
    props: { item },
    global: { plugins: [ElementPlus, router] },
  });
  await flushPromises();
  return w;
}

describe("ItemHistoryCard", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    api.history.mockReset().mockResolvedValue(history);
    api.revert
      .mockReset()
      .mockResolvedValue({ ...item, attributes: { title: "A" } });
  });

  it("attribue la fiche et montre les différences champ par champ", async () => {
    const w = await mountCard("user");
    expect(w.find('[data-test="attribution"]').text()).toContain(
      "Ajouté par Camille",
    );
    expect(w.find('[data-test="attribution"]').text()).toContain("un membre");
    const first = w.findAll('[data-test="history-entry"]')[0]!;
    expect(first.text()).toContain("Titre");
    expect(first.find("del").text()).toBe("A");
    expect(first.find("ins").text()).toBe("B");
    expect(w.find('[data-test="revert-1"]').exists()).toBe(false);
  });

  it("un contributeur revient à une version antérieure (pas à la dernière)", async () => {
    const w = await mountCard("contributor");
    expect(w.find('[data-test="revert-2"]').exists()).toBe(false);
    await w.find('[data-test="revert-1"]').trigger("click");
    await flushPromises();
    expect(api.revert).toHaveBeenCalledWith("i1", 1);
    expect(w.emitted("reverted")).toBeTruthy();
  });
});
