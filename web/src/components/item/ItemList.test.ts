import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { createMemoryHistory, createRouter } from "vue-router";
import { defineComponent } from "vue";
import ItemList from "./ItemList.vue";
import type { Item } from "@/api/types";

const Empty = defineComponent({ template: "<div />" });

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: Empty }],
  });
}

const item: Item = {
  id: "e1",
  item_type_id: "b1",
  item_type: {
    id: "b1",
    name: "Series",
    attributes_schema: { "x-label": "{title}" },
  } as unknown as Item["item_type"],
  attributes: { title: "Breaking Bad" },
  tags: [
    { id: "t1", name: "drama" },
    { id: "t2", name: "thriller" },
  ] as Item["tags"],
  created_at: "2026-10-11T00:00:00Z",
  updated_at: "2026-10-11T00:00:00Z",
};

describe("ItemList", () => {
  it("rend chaque item en carte-lien sur mobile, avec libellé et tags", () => {
    const w = mount(ItemList, {
      props: { items: [item], loading: false, canEdit: false, emptyText: "-" },
      global: { plugins: [ElementPlus, makeRouter()] },
    });
    const card = w.find("ul a");
    expect(card.attributes("href")).toBe("/items/e1");
    expect(card.text()).toContain("Breaking Bad");
    expect(card.text()).toContain("drama, thriller");
    // Pas d'image : initiales décoratives à la place de la jaquette.
    expect(card.find("[aria-hidden='true']").text()).toBe("BB");
  });

  it("affiche le message vide fourni quand il n'y a aucun résultat", () => {
    const w = mount(ItemList, {
      props: {
        items: [],
        loading: false,
        canEdit: false,
        emptyText: "Aucun item ne correspond à ces critères.",
      },
      global: { plugins: [ElementPlus, makeRouter()] },
    });
    expect(w.find("ul li").text()).toBe(
      "Aucun item ne correspond à ces critères.",
    );
  });
});
