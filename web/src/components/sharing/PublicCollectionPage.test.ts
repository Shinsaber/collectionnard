import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { AxiosError, type AxiosResponse } from "axios";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type {
  CollectionItem,
  PaginatedResponse,
  PublicCollection,
} from "@/api/types";
import PublicCollectionPage from "./PublicCollectionPage.vue";

vi.mock("@/api/client", () => ({ extractApiError: () => ({ message: "" }) }));

const col: PublicCollection = {
  id: "c1",
  name: "Mes DVD",
  item_count: 1,
  visibility: "public",
  owner: { username: "camille", display_name: "Camille" },
  created_at: "",
  updated_at: "",
  collection_type: {
    id: "t1",
    name: "Ownership",
    item_type_ids: [],
    attributes_schema: {
      type: "object",
      "x-labels": { fr: { condition: "État" } },
      properties: { condition: { type: "string" } },
    },
    created_at: "",
    updated_at: "",
  },
};

const page = {
  data: [
    {
      id: "ci1",
      collection_id: "c1",
      item_id: "i1",
      attributes: { condition: "new" },
      created_at: "",
      updated_at: "",
      item: { id: "i1", item_type_id: "it", attributes: { title: "Amélie" } },
    },
  ],
  total: 1,
  page: 1,
  per_page: 20,
  total_pages: 1,
} as unknown as PaginatedResponse<CollectionItem>;

async function mountPage(loadCollection: () => Promise<PublicCollection>) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  const w = mount(PublicCollectionPage, {
    props: {
      loadCollection,
      loadItems: () => Promise.resolve(page),
      loadStats: vi.fn(),
    },
    global: { plugins: [ElementPlus, router] },
  });
  await flushPromises();
  return w;
}

describe("PublicCollectionPage", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
  });

  it("affiche la collection en lecture seule, sans lien vers les fiches pour un anonyme", async () => {
    const w = await mountPage(() => Promise.resolve(col));
    expect(w.text()).toContain("Mes DVD");
    expect(w.text()).toContain("Amélie");
    expect(w.text()).toContain("État");
    expect(w.find('[data-test="public-owner"]').attributes("href")).toBe(
      "/u/camille",
    );
    expect(w.find('a[href="/items/i1"]').exists()).toBe(false);
    expect(w.find('[aria-label="Modifier les attributs"]').exists()).toBe(
      false,
    );
  });

  it("une collection invisible ou inconnue affiche un message, sans erreur", async () => {
    const notFound = new AxiosError("404", "ERR", undefined, null, {
      status: 404,
      data: {},
      statusText: "",
      headers: {},
      config: { headers: {} },
    } as AxiosResponse);
    const w = await mountPage(() => Promise.reject(notFound));
    expect(w.find('[data-test="public-not-found"]').exists()).toBe(true);
  });
});
