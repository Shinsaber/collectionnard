import { mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { describe, expect, it } from "vitest";
import type { CollectionItem } from "@/api/types";
import CollectionItemList from "./CollectionItemList.vue";

function lot(quantity: number): CollectionItem {
  return {
    id: `l${quantity}`,
    collection_id: "c1",
    item_id: "e1",
    quantity,
    attributes: {},
    created_at: "",
    updated_at: "",
  };
}

/** Libellés des colonnes déclarées (jsdom ne dessine pas l'en-tête d'ElTable). */
function columns(w: ReturnType<typeof mountList>): string[] {
  return w
    .findAllComponents({ name: "ElTableColumn" })
    .map((c) => String(c.props("label") ?? ""));
}

function mountList(items: CollectionItem[]) {
  return mount(CollectionItemList, {
    props: { items, schema: { type: "object" }, loading: false },
    global: { plugins: [ElementPlus], stubs: { RouterLink: true } },
  });
}

describe("CollectionItemList (P7-1)", () => {
  it("masque la colonne « Qté » quand chaque lot compte un exemplaire", () => {
    const w = mountList([lot(1)]);
    expect(columns(w)).not.toContain("Qté");
    expect(w.find('[data-test="lot-quantity-cell"]').exists()).toBe(false);
  });

  it("montre la quantité dès qu'un lot en compte plusieurs", () => {
    const w = mountList([lot(1), lot(3)]);
    expect(columns(w)).toContain("Qté");
    expect(w.text()).toContain("3 exemplaires"); // pastille des cartes mobiles
  });
});
