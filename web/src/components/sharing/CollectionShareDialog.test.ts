import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { Collection } from "@/api/types";
import CollectionShareDialog from "./CollectionShareDialog.vue";

const api = vi.hoisted(() => ({
  update: vi.fn(),
  list: vi.fn(),
  create: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  collectionApi: { update: api.update },
  shareLinkApi: { list: api.list, create: api.create, revoke: vi.fn() },
  extractApiError: () => ({ message: "" }),
}));

function collection(over: Partial<Collection> = {}): Collection {
  return {
    id: "c1",
    user_id: "u1",
    collection_type_id: "t1",
    name: "Mes DVD",
    item_count: 2,
    visibility: "private",
    hidden_fields: null,
    created_at: "2026-10-01T00:00:00Z",
    updated_at: "2026-10-01T00:00:00Z",
    collection_type: {
      id: "t1",
      name: "Ownership",
      item_type_ids: [],
      attributes_schema: {
        type: "object",
        "x-order": ["condition", "purchase_price"],
        "x-labels": { fr: { condition: "État", purchase_price: "Prix" } },
        properties: {
          condition: { type: "string" },
          purchase_price: { type: "number", "x-private": true },
        },
      },
      created_at: "",
      updated_at: "",
    },
    ...over,
  } as Collection;
}

async function mountDialog(c: Collection) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  const w = mount(CollectionShareDialog, {
    props: { collection: c, visible: true },
    global: { plugins: [ElementPlus, router] },
    attachTo: document.body,
  });
  await flushPromises();
  return w;
}

const checkbox = (key: string) =>
  document.querySelector<HTMLInputElement>(`[data-test="hide-${key}"] input`)!;
const click = async (sel: string) => {
  document.querySelector<HTMLElement>(sel)!.click();
  await flushPromises();
};

describe("CollectionShareDialog", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    Object.values(api).forEach((m) => m.mockReset());
    api.list.mockResolvedValue([]);
    api.update.mockImplementation((_id, body) =>
      Promise.resolve(collection(body)),
    );
  });

  it("part du réglage du type (x-private) et l'envoie comme null", async () => {
    const w = await mountDialog(collection());
    expect(checkbox("purchase_price").checked).toBe(true);
    expect(checkbox("condition").checked).toBe(false);
    expect(
      document.querySelector('[data-test="hidden-mode"]')!.textContent,
    ).toContain("Réglage du type");
    await click('[data-test="visibility-public"] input');
    await click('[data-test="share-save"]');
    expect(api.update).toHaveBeenCalledWith("c1", {
      visibility: "public",
      hidden_fields: null,
    });
    expect(w.emitted("saved")).toBeTruthy();
  });

  it("un choix propre à la collection est envoyé en liste, et se réinitialise", async () => {
    await mountDialog(collection());
    checkbox("purchase_price").click();
    checkbox("condition").click();
    await flushPromises();
    expect(
      document.querySelector('[data-test="hidden-mode"]')!.textContent,
    ).toContain("Choix propre");
    await click('[data-test="share-save"]');
    expect(api.update).toHaveBeenLastCalledWith("c1", {
      visibility: "private",
      hidden_fields: ["condition"],
    });

    await click('[data-test="hidden-reset"]');
    expect(checkbox("purchase_price").checked).toBe(true);
    await click('[data-test="share-save"]');
    expect(api.update).toHaveBeenLastCalledWith("c1", {
      visibility: "private",
      hidden_fields: null,
    });
  });

  it("reprend une liste déjà choisie (même vide : tout montrer)", async () => {
    await mountDialog(collection({ hidden_fields: [] }));
    expect(checkbox("purchase_price").checked).toBe(false);
    expect(
      document.querySelector('[data-test="hidden-mode"]')!.textContent,
    ).toContain("Choix propre");
  });

  it("montre le lien de partage une seule fois, construit côté client", async () => {
    api.create.mockResolvedValue({
      share_link: {
        id: "l1",
        collection_id: "c1",
        created_at: "2026-10-18T00:00:00Z",
      },
      token: "abc+/=",
    });
    await mountDialog(collection());
    await click('[data-test="create-share-link"]');
    const input = document.querySelector<HTMLInputElement>(
      'input[data-test="one-time-link"]',
    )!;
    expect(input.value).toBe(
      `${window.location.origin}/share?token=abc%2B%2F%3D`,
    );
  });
});
