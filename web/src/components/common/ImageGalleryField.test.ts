import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus, { ElSwitch } from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent } from "vue";
import ImageGalleryField from "./ImageGalleryField.vue";

// Recadrage sur ordinateur : case « Recadrer avant l'envoi » et bouton « Recadrer » d'une image
// déjà envoyée (le résultat s'ajoute comme une nouvelle image, l'original reste).
const api = vi.hoisted(() => ({ upload: vi.fn(), file: vi.fn() }));
vi.mock("@/api/client", () => ({
  uploadApi: api,
  extractApiError: () => ({ message: "" }),
}));
vi.mock("@/lib/images", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/images")>()),
  imageSize: () => Promise.resolve({ width: 200, height: 300 }),
}));
// Le dialogue réel charge cropperjs : remplacé par un faux qui expose ce qu'il reçoit.
vi.mock("./ImageCropDialog.vue", () => ({
  __esModule: true, // defineAsyncComponent prend alors `default`
  default: defineComponent({
    name: "ImageCropDialog",
    props: {
      file: { type: Object, default: null },
      category: { type: Object, default: null },
      reason: { type: String, default: undefined },
      straighten: { type: Boolean, default: false },
      visible: { type: Boolean, default: false },
    },
    emits: ["done", "update:visible"],
    template: '<div data-test="crop-dialog" />',
  }),
}));

const categories = [{ name: "poster", ratio: "2/3" }];
const OLD = "/api/v1/uploads/aaa.jpg";

function mountGallery(modelValue: Record<string, unknown> = {}) {
  return mount(ImageGalleryField, {
    props: { categories, modelValue },
    global: { plugins: [ElementPlus] },
  });
}

async function chooseFile(w: ReturnType<typeof mountGallery>, file: File) {
  const input = w.find('input[type="file"]');
  Object.defineProperty(input.element, "files", { value: [file] });
  await input.trigger("change");
  await flushPromises();
}

function lastValue(w: ReturnType<typeof mountGallery>) {
  const events = w.emitted("update:modelValue")!;
  return events[events.length - 1]![0] as Record<string, unknown>;
}

const photo = () => new File(["x"], "cover.jpg", { type: "image/jpeg" });

describe("ImageGalleryField — recadrage volontaire", () => {
  beforeEach(() => {
    localStorage.removeItem("colec-crop-before-upload");
    api.upload
      .mockReset()
      .mockResolvedValue({ url: "/api/v1/uploads/new.jpg" });
    api.file.mockReset();
  });

  it("sans la case, un fichier conforme est envoyé tel quel", async () => {
    const w = mountGallery();
    await chooseFile(w, photo());
    expect(api.upload).toHaveBeenCalledTimes(1);
    expect(w.find('[data-test="crop-dialog"]').exists()).toBe(false);
    expect(lastValue(w)).toEqual({
      poster: ["/api/v1/uploads/new.jpg"],
      default_image: "/api/v1/uploads/new.jpg",
    });
  });

  it("avec la case, chaque fichier passe par le recadrage, et le choix est mémorisé", async () => {
    const w = mountGallery();
    w.findComponent(ElSwitch).vm.$emit("update:modelValue", true);
    await flushPromises();
    expect(localStorage.getItem("colec-crop-before-upload")).toBe("1");

    const original = photo();
    await chooseFile(w, original);
    expect(api.upload).not.toHaveBeenCalled();
    const dialog = w.findComponent({ name: "ImageCropDialog" });
    expect(dialog.props("file")).toBe(original);
    expect(dialog.props("category")).toEqual(categories[0]);

    const cropped = new File(["y"], "cover.jpg", { type: "image/jpeg" });
    dialog.vm.$emit("done", cropped);
    await flushPromises();
    expect(api.upload).toHaveBeenCalledWith(cropped);
    expect(lastValue(w).poster).toEqual(["/api/v1/uploads/new.jpg"]);

    // Remonté plus tard : la case est cochée d'office.
    const again = mountGallery();
    expect(again.findComponent(ElSwitch).props("modelValue")).toBe(true);
  });

  it("« Recadrer » une image envoyée ajoute le résultat en fin de liste et garde l'original", async () => {
    const reloaded = new File(["z"], "aaa.jpg", { type: "image/jpeg" });
    api.file.mockResolvedValue(reloaded);
    const w = mountGallery({ poster: [OLD], default_image: OLD });
    await w.find('[data-test="gallery-recrop"]').trigger("click");
    await flushPromises();
    expect(api.file).toHaveBeenCalledWith(OLD);
    const dialog = w.findComponent({ name: "ImageCropDialog" });
    expect(dialog.props("file")).toBe(reloaded);

    dialog.vm.$emit("done", new File(["w"], "aaa.jpg", { type: "image/jpeg" }));
    await flushPromises();
    expect(lastValue(w)).toEqual({
      poster: [OLD, "/api/v1/uploads/new.jpg"],
      default_image: OLD, // l'image par défaut ne change pas
    });
  });

  it("pas de bouton « Recadrer » pour une image hors du serveur", () => {
    const w = mountGallery({ poster: ["https://exemple.org/c.jpg"] });
    expect(w.find('[data-test="gallery-recrop"]').exists()).toBe(false);
  });
});
