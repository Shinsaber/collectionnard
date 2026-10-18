import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { Item } from "@/api/types";
import ItemEditDialog from "./ItemEditDialog.vue";

const api = vi.hoisted(() => ({
  duplicates: vi.fn(),
  update: vi.fn(),
  create: vi.fn(),
  propose: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  itemApi: {
    duplicates: api.duplicates,
    update: api.update,
    create: api.create,
  },
  proposalApi: { create: api.propose },
  extractApiError: () => ({ message: "" }),
}));

const item = {
  id: "i1",
  item_type_id: "t1",
  attributes: { title: "Tome 1", isbn: "978" },
  tags: [{ id: "g", name: "manga" }],
  item_type: {
    id: "t1",
    name: "Volume",
    attributes_schema: {
      type: "object",
      "x-label": "{title}",
      properties: {
        title: { type: "string" },
        isbn: { type: "string", "x-identifier": true },
      },
    },
  },
} as unknown as Item;

async function mountDialog(propose: boolean) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  const w = mount(ItemEditDialog, {
    props: { visible: false, item, itemTypes: [], propose },
    global: { plugins: [ElementPlus, router] },
    attachTo: document.body,
  });
  await w.setProps({ visible: true });
  await flushPromises();
  return w;
}

async function edit(title: string) {
  const input =
    document.querySelector<HTMLInputElement>(
      "input[id$='title']:not(#proposal-title)",
    ) ??
    [...document.querySelectorAll<HTMLInputElement>(".el-dialog input")].find(
      (i) => i.value === "Tome 1",
    )!;
  input.value = title;
  input.dispatchEvent(new Event("input"));
  await flushPromises();
}

const save = async () => {
  document.querySelector<HTMLElement>('[data-test="item-save"]')!.click();
  await flushPromises();
};

describe("ItemEditDialog", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    document.body.innerHTML = "";
    Object.values(api).forEach((m) => m.mockReset());
    api.duplicates.mockResolvedValue([]);
    api.propose.mockResolvedValue({ id: "p1" });
  });

  it("un membre envoie une correction en proposition (tags compris), sans écrire", async () => {
    const w = await mountDialog(true);
    await edit("Tome 1 (corrigé)");
    await save();
    expect(api.update).not.toHaveBeenCalled();
    expect(api.propose).toHaveBeenCalledWith({
      kind: "catalog",
      title: "Correction : Tome 1 (corrigé)",
      payload: {
        items: [
          {
            id: "i1",
            item_type_id: "t1",
            attributes: { title: "Tome 1 (corrigé)", isbn: "978" },
            tags: ["manga"],
          },
        ],
        relations: [],
      },
      submit: true,
    });
    expect(w.emitted("saved")).toBeTruthy();
  });

  it("signale un doublon au premier clic, enregistre au second", async () => {
    api.duplicates.mockResolvedValue([
      { field: "isbn", value: "978", item: { ...item, id: "i2" } },
    ]);
    await mountDialog(false);
    await edit("Tome 1 bis");
    await save();
    expect(api.duplicates).toHaveBeenCalledWith(
      "t1",
      { title: "Tome 1 bis", isbn: "978" },
      "i1",
    );
    expect(document.querySelector('[data-test="duplicates"]')).not.toBeNull();
    expect(api.update).not.toHaveBeenCalled();
    await save();
    expect(api.update).toHaveBeenCalledTimes(1);
  });
});
