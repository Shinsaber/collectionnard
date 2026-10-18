import { beforeEach, describe, expect, it } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { createMemoryHistory, createRouter } from "vue-router";
import { defineComponent } from "vue";
import { isMoreActive, isNavActive } from "./navigation";
import AppTabBar from "./AppTabBar.vue";

describe("isNavActive", () => {
  it("n'active l'accueil que sur lui-même", () => {
    expect(isNavActive("/", "/")).toBe(true);
    expect(isNavActive("/", "/items")).toBe(false);
  });

  it("active une section sur ses sous-routes, pas sur un préfixe homonyme", () => {
    expect(isNavActive("/items", "/items")).toBe(true);
    expect(isNavActive("/items", "/items/42/graph")).toBe(true);
    expect(isNavActive("/collections", "/collections-archive")).toBe(false);
  });

  it("rattache l'administration, les statistiques et le hors ligne à l'onglet « Plus »", () => {
    expect(isMoreActive("/admin/item-types")).toBe(true);
    expect(isMoreActive("/stats")).toBe(true);
    expect(isMoreActive("/offline")).toBe(true);
    expect(isMoreActive("/items")).toBe(false);
  });
});

describe("AppTabBar", () => {
  beforeEach(() => setActivePinia(createPinia()));
  const Empty = defineComponent({ template: "<div />" });

  async function mountAt(path: string, moreOpen = false) {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: "/:any(.*)*", component: Empty }],
    });
    await router.push(path);
    await router.isReady();
    return mount(AppTabBar, {
      props: { moreOpen },
      global: { plugins: [ElementPlus, router] },
    });
  }

  it("met en avant l'onglet de la section courante", async () => {
    const w = await mountAt("/items/42");
    const active = w.findAll("a").filter((a) => a.classes("font-bold"));
    expect(active.map((a) => a.text())).toEqual(["Catalogue"]);
  });

  it("active « Plus » sur une page d'administration et signale la feuille", async () => {
    const w = await mountAt("/admin/item-types");
    const more = w.find("button");
    expect(more.text()).toBe("Plus");
    expect(more.classes()).toContain("font-bold");
    expect(more.attributes("aria-haspopup")).toBe("dialog");
    await more.trigger("click");
    expect(w.emitted("openMore")).toHaveLength(1);
  });
});
