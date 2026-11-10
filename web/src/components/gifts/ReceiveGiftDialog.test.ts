import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Collection, CollectionItem, GiftReservation } from "@/api/types";
import ReceiveGiftDialog from "./ReceiveGiftDialog.vue";

const api = vi.hoisted(() => ({ list: vi.fn(), receive: vi.fn() }));
vi.mock("@/api/client", () => ({
  collectionApi: { list: api.list },
  giftApi: { receive: api.receive },
  extractApiError: () => ({ message: "" }),
}));

const col = (id: string, over: Record<string, unknown> = {}) =>
  ({
    id,
    name: id,
    item_type_ids: [],
    collection_type: {
      id: "t",
      name: "T",
      attributes_schema: {},
      item_type_ids: [],
    },
    ...over,
  }) as unknown as Collection;
const lot = {
  id: "l1",
  quantity: 2,
  item: { id: "e1", item_type_id: "b1", attributes: {} },
} as unknown as CollectionItem;
const gift = { id: "g1", quantity: 1 } as GiftReservation;

async function open() {
  const w = mount(ReceiveGiftDialog, {
    props: { collectionId: "wish", lot, gift, label: "Tome 1", visible: false },
    global: {
      plugins: [ElementPlus],
      stubs: {
        DynamicForm: { template: "<div />", methods: { validate: () => true } },
      },
    },
    attachTo: document.body,
  });
  await w.setProps({ visible: true });
  await flushPromises();
  return w;
}
const click = (sel: string) =>
  (document.querySelector(sel) as HTMLElement).click();

describe("ReceiveGiftDialog", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    api.receive.mockReset().mockResolvedValue({});
    api.list.mockReset().mockResolvedValue({
      data: [
        col("wish"),
        col("films", { item_type_ids: ["b9"] }),
        col("autre"),
        col("possession", {
          collection_type: {
            id: "d1",
            name: "Ownership",
            attributes_schema: { type: "object", "x-lendable": true },
            item_type_ids: [],
          },
        }),
      ],
      total: 4,
    });
  });

  it("propose mes autres collections qui acceptent l'item, une possession d'abord", async () => {
    const w = await open();
    const options = [
      ...document.querySelectorAll('[data-test^="receive-into-"]'),
    ].map((e) => e.getAttribute("data-test"));
    expect(options).toEqual(["receive-into-possession", "receive-into-autre"]);
    click('[data-test="receive-submit"]');
    await flushPromises();
    expect(api.receive).toHaveBeenCalledWith("wish", "g1", {
      collection_id: "possession",
      attributes: {},
    });
    w.unmount();
  });

  it("« le retirer seulement » n'envoie aucune collection", async () => {
    const w = await open();
    (
      document.querySelector(
        '[data-test="receive-remove-only"] input',
      ) as HTMLInputElement
    ).click();
    await flushPromises();
    click('[data-test="receive-submit"]');
    await flushPromises();
    expect(api.receive).toHaveBeenCalledWith("wish", "g1", {});
    w.unmount();
  });
});
