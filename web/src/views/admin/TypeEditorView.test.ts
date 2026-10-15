import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { defineComponent } from "vue";

const api = vi.hoisted(() => ({
  get: vi.fn(),
  update: vi.fn(),
  create: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  itemTypeApi: api,
  relationTypeApi: {},
  collectionTypeApi: {},
  extractApiError: () => ({ message: "" }),
}));

import TypeEditorView from "./TypeEditorView.vue";

const volume = {
  id: "b15",
  name: "Volume",
  description: "Volume physique",
  attributes_schema: {
    type: "object",
    properties: { number: { type: "integer" } },
    "x-title": { fr: "Tome", en: "Volume" },
  },
};

async function mountEditor(id?: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: "/:any(.*)*",
        component: defineComponent({ template: "<div />" }),
      },
    ],
  });
  await router.push("/admin/item-types/b15");
  const w = mount(TypeEditorView, {
    props: { kind: "item", id },
    global: { plugins: [ElementPlus, router] },
  });
  await flushPromises();
  return { w, router };
}

function button(w: ReturnType<typeof mount>, text: string) {
  return w.findAll("button").find((b) => b.text() === text)!;
}

describe("TypeEditorView", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    Object.values(api).forEach((m) => m.mockReset());
  });

  it("charge le type, signale les changements et enregistre la mise à jour", async () => {
    api.get.mockResolvedValue(volume);
    api.update.mockResolvedValue(volume);
    const { w, router } = await mountEditor("b15");
    await new Promise((r) => setTimeout(r));

    expect(api.get).toHaveBeenCalledWith("b15");
    expect(w.find("h1").text()).toBe("Tome");
    expect(w.text()).not.toContain("Modifications non enregistrées");

    const name = w.find('input[placeholder^="Ex."]');
    await name.setValue("Volume2");
    expect(w.text()).toContain("Modifications non enregistrées");

    const push = vi.spyOn(router, "push");
    await button(w, "Enregistrer").trigger("click");
    await flushPromises();
    const [id, payload] = api.update.mock.calls[0]!;
    expect(id).toBe("b15");
    expect(payload.name).toBe("Volume2");
    expect(payload.attributes_schema.properties).toHaveProperty("number");
    expect(push).toHaveBeenCalledWith("/admin/item-types");
  });

  it("nouveau type : formulaire vide et création", async () => {
    api.create.mockResolvedValue(volume);
    const { w } = await mountEditor();
    expect(api.get).not.toHaveBeenCalled();
    expect(w.find("h1").text()).toBe("Nouveau type");
    await w.find('input[placeholder^="Ex."]').setValue("Comic");
    await button(w, "Créer").trigger("click");
    await flushPromises();
    expect(api.create.mock.calls[0]![0].name).toBe("Comic");
  });
});
