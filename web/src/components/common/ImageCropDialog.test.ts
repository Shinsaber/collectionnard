import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { describe, expect, it, vi } from "vitest";
import { defineComponent } from "vue";
import ImageCropDialog from "./ImageCropDialog.vue";

// Sélecteur d'étapes « Redresser | Recadrer » en tête du dialogue.
vi.mock("cropperjs", () => ({
  default: class {
    getCropperImage() {
      return null;
    }
    getCropperSelection() {
      return null;
    }
    getCropperCanvas() {
      return null;
    }
  },
}));

const Perspective = defineComponent({
  name: "PerspectiveEditor",
  props: { file: { type: Object, default: null } },
  emits: ["done", "skip"],
  template: '<div data-test="perspective-stub" />',
});

const file = new File(["x"], "cover.jpg", { type: "image/jpeg" });

function mountDialog() {
  return mount(ImageCropDialog, {
    props: {
      visible: true,
      file,
      category: { name: "poster", ratio: "2/3" },
    },
    attachTo: document.body,
    global: {
      plugins: [ElementPlus],
      stubs: { PerspectiveEditor: Perspective, teleport: true },
    },
  });
}

function stepInput(w: ReturnType<typeof mountDialog>, step: string) {
  return w.find(`[data-test="crop-steps"] input[value="${step}"]`);
}

describe("ImageCropDialog — étapes", () => {
  it("passe du recadrage au redressement et retour par le sélecteur", async () => {
    const w = mountDialog();
    await flushPromises();
    expect(w.find('[data-test="crop-steps"]').exists()).toBe(true);
    expect(w.find('[data-test="perspective-stub"]').exists()).toBe(false);

    await stepInput(w, "straighten").setValue(true);
    await flushPromises();
    const editor = w.findComponent({ name: "PerspectiveEditor" });
    expect(editor.exists()).toBe(true);
    expect(editor.props("file")).toBe(file); // repart de l'image d'origine

    await stepInput(w, "crop").setValue(true);
    await flushPromises();
    expect(w.find('[data-test="perspective-stub"]').exists()).toBe(false);
    expect(w.find('[data-test="crop-container"]').isVisible()).toBe(true);
    w.unmount();
  });

  it("une image redressée passe au recadrage avec le sélecteur sur « Recadrer »", async () => {
    const w = mountDialog();
    await flushPromises();
    await stepInput(w, "straighten").setValue(true);
    await flushPromises();
    w.findComponent({ name: "PerspectiveEditor" }).vm.$emit(
      "done",
      new File(["y"], "cover.jpg", { type: "image/jpeg" }),
    );
    await flushPromises();
    expect(w.find('[data-test="perspective-stub"]').exists()).toBe(false);
    expect((stepInput(w, "crop").element as HTMLInputElement).checked).toBe(
      true,
    );
    w.unmount();
  });
});
