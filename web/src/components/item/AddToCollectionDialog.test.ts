import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Collection, Item, ItemCollectionMembership } from "@/api/types";

const api = vi.hoisted(() => ({
  list: vi.fn(),
  addItem: vi.fn(),
  updateLot: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  collectionApi: api,
  extractApiError: () => ({ message: "" }),
}));

import AddToCollectionDialog from "./AddToCollectionDialog.vue";

const schema = {
  type: "object",
  properties: { condition: { type: "string" } },
};
const cards = {
  id: "c1",
  name: "Mes cartes",
  collection_type: { id: "d1", name: "Ownership", attributes_schema: schema },
} as unknown as Collection;
const wishlist = {
  id: "c2",
  name: "Envies",
  collection_type: { id: "d3", name: "Wishlist", attributes_schema: {} },
} as unknown as Collection;
const item = { id: "e1", item_type_id: "b1", attributes: {} } as Item;
const memberships: ItemCollectionMembership[] = [
  {
    collection: cards,
    lot_id: "l1",
    quantity: 5,
    attributes: { condition: "good" },
  },
];

async function open() {
  const w = mount(AddToCollectionDialog, {
    props: { visible: false, item, memberships },
    global: { plugins: [ElementPlus] },
    attachTo: document.body,
  });
  await w.setProps({ visible: true });
  await flushPromises();
  return w;
}

function choose(value: string) {
  const input = document.querySelector<HTMLInputElement>(
    `input[type="radio"][value="${value}"]`,
  );
  input!.click();
  return flushPromises();
}

function submit() {
  document
    .querySelector<HTMLButtonElement>(".el-dialog__footer .el-button--primary")!
    .click();
  return flushPromises();
}

describe("AddToCollectionDialog (P7-1)", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    Object.values(api).forEach((m) => m.mockReset());
    api.list.mockResolvedValue({ data: [cards, wishlist], total: 2 });
    api.addItem.mockResolvedValue({});
    api.updateLot.mockResolvedValue({});
  });

  it("propose une collection qui contient déjà l'item, et demande quel lot augmenter", async () => {
    const w = await open();
    await choose("c1");
    expect(document.querySelectorAll('[data-test="lot-option"]')).toHaveLength(
      1,
    );
    const add = document.querySelector<HTMLButtonElement>(
      ".el-dialog__footer .el-button--primary",
    )!;
    expect(add.disabled).toBe(true); // choix explicite exigé
    await choose("l1");
    await submit();
    expect(api.updateLot).toHaveBeenCalledWith("c1", "l1", {
      attributes: { condition: "good" },
      quantity: 6,
    });
    expect(w.emitted("added")).toHaveLength(1);
  });

  it("crée un nouveau lot dans une collection qui ne contient pas l'item", async () => {
    await open();
    await choose("c2");
    expect(document.querySelector('[data-test="lot-option"]')).toBeNull();
    await submit();
    expect(api.addItem).toHaveBeenCalledWith("c2", {
      item_id: "e1",
      attributes: {},
      quantity: 1,
    });
  });
});
