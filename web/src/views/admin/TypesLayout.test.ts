import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { describe, expect, it } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import TypesLayout from "./TypesLayout.vue";

describe("TypesLayout", () => {
  it("suit l'URL et change de page au clic sur un onglet", async () => {
    const page = (text: string) => ({
      template: `<p data-test="page">${text}</p>`,
    });
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: "/admin/types",
          component: TypesLayout,
          children: [
            { path: "/admin/item-types", component: page("items") },
            { path: "/admin/relation-types", component: page("relations") },
            { path: "/admin/collection-types", component: page("collections") },
            { path: "/admin/review-types", component: page("reviews") },
          ],
        },
      ],
    });
    await router.push("/admin/review-types");
    await router.isReady();
    const w = mount(
      { template: "<RouterView />" },
      {
        global: { plugins: [ElementPlus, router] },
      },
    );
    await flushPromises();
    expect(w.find('[data-test="page"]').text()).toBe("reviews");
    const tabs = w.findAll('[role="tab"]');
    expect(tabs.map((tab) => tab.text())).toEqual([
      "Items",
      "Relations",
      "Collections",
      "Avis",
    ]);
    expect(tabs[3]!.attributes("aria-selected")).toBe("true");

    await tabs[1]!.trigger("click");
    await flushPromises();
    expect(router.currentRoute.value.path).toBe("/admin/relation-types");
    expect(w.find('[data-test="page"]').text()).toBe("relations");
  });
});
