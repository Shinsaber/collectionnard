import { mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { afterEach, describe, expect, it } from "vitest";
import CameraCaptureDialog from "./CameraCaptureDialog.vue";

afterEach(() => {
  document.body.innerHTML = "";
});

describe("CameraCaptureDialog", () => {
  it("sans caméra démarrée, le déclencheur est inactif et l'appareil du téléphone reste proposé", async () => {
    const w = mount(CameraCaptureDialog, {
      props: { visible: true, category: "poster" },
      global: { plugins: [ElementPlus] },
      attachTo: document.body,
    });
    await w.vm.$nextTick();
    // Le dialogue est `append-to-body` : son contenu vit dans le document.
    const shutter = document.body.querySelector<HTMLButtonElement>(
      "[data-test=camera-shutter]",
    );
    expect(shutter).not.toBeNull();
    expect(shutter!.disabled).toBe(true);
    expect(document.body.textContent).toContain("Photo – poster");
    expect(document.body.textContent).toContain("Appareil photo du téléphone");

    // Photo venue de l'appareil du téléphone : rendue telle quelle, le dialogue se ferme.
    const input = document.body.querySelector<HTMLInputElement>(
      "input[type=file][capture]",
    )!;
    const file = new File(["x"], "IMG_0001.jpg", { type: "image/jpeg" });
    Object.defineProperty(input, "files", { value: [file] });
    input.dispatchEvent(new Event("change"));
    await w.vm.$nextTick();
    expect(w.emitted("done")).toEqual([[file]]);
    expect(w.emitted("update:visible")).toEqual([[false]]);
    w.unmount();
  });
});
