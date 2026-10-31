import { mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { afterEach, describe, expect, it } from "vitest";
import GalleryThumbs from "./GalleryThumbs.vue";

const urls = [
  "/api/v1/uploads/a.jpg",
  "/api/v1/uploads/b.jpg",
  "/x/c.jpg",
  "/x/d.jpg",
  "/x/e.jpg",
];

function mountThumbs() {
  return mount(GalleryThumbs, {
    props: { urls, label: "Photos", max: 4 },
    global: { plugins: [ElementPlus] },
    attachTo: document.body,
  });
}

afterEach(() => {
  document.body.innerHTML = "";
});

describe("GalleryThumbs", () => {
  it("montre les premières vignettes (réduites pour les envois) puis « +n »", () => {
    const w = mountThumbs();
    const imgs = w.findAll("img");
    expect(imgs).toHaveLength(4);
    expect(imgs[0]!.attributes("src")).toBe("/api/v1/uploads/a.jpg?w=240");
    expect(imgs[2]!.attributes("src")).toBe("/x/c.jpg");
    expect(w.text()).toContain("+1");
    expect(w.find("button").attributes("aria-label")).toBe(
      "Agrandir : Photos 1",
    );
    w.unmount();
  });

  it("ouvre la visionneuse sur toute la galerie à partir de la vignette cliquée", async () => {
    const w = mountThumbs();
    expect(document.body.querySelector("[data-test=lightbox]")).toBeNull();
    await w.findAll("button")[1]!.trigger("click");
    await w.vm.$nextTick();
    const viewer = document.body.querySelector("[data-test=lightbox]");
    expect(viewer).not.toBeNull();
    // Image d'origine (pas la vignette) : la deuxième ; compteur sur les cinq.
    expect(
      viewer!.querySelector("[data-test=lightbox-image]")?.getAttribute("src"),
    ).toBe("/api/v1/uploads/b.jpg");
    expect(
      viewer!.querySelector("[data-test=lightbox-counter]")?.textContent,
    ).toBe("2 / 5");
    // Suivante, puis fermeture.
    (
      viewer!.querySelector("[data-test=lightbox-next]") as HTMLButtonElement
    ).click();
    await w.vm.$nextTick();
    expect(
      viewer!.querySelector("[data-test=lightbox-image]")?.getAttribute("src"),
    ).toBe("/x/c.jpg");
    (
      viewer!.querySelector("[data-test=lightbox-close]") as HTMLButtonElement
    ).click();
    await w.vm.$nextTick();
    expect(document.body.querySelector("[data-test=lightbox]")).toBeNull();
    w.unmount();
  });

  it("« +n » ouvre la visionneuse sur la première image cachée", async () => {
    const w = mountThumbs();
    await w.findAll("button")[4]!.trigger("click");
    await w.vm.$nextTick();
    const shown = document.body.querySelector("[data-test=lightbox-image]");
    expect(shown?.getAttribute("src")).toBe("/x/e.jpg");
    w.unmount();
  });

  it("ne rend rien sans image", () => {
    const w = mount(GalleryThumbs, {
      props: { urls: [], label: "Photos" },
      global: { plugins: [ElementPlus] },
    });
    expect(w.find("[data-test=gallery-thumbs]").exists()).toBe(false);
  });
});
