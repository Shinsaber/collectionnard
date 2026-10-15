import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import AppLogo from "./AppLogo.vue";

describe("AppLogo", () => {
  it("affiche le nom à côté de l'étiquette, décorative", () => {
    const w = mount(AppLogo);
    expect(w.text()).toBe("collectionnard");
    expect(w.find("svg").attributes("aria-hidden")).toBe("true");
  });

  it("donne un nom accessible à l'étiquette seule", () => {
    const w = mount(AppLogo, { props: { wordmark: false, size: 40 } });
    const svg = w.find("svg");
    expect(w.text()).toBe("");
    expect(svg.attributes("role")).toBe("img");
    expect(svg.attributes("aria-label")).toBe("Collectionnard");
    expect(svg.attributes("width")).toBe("40");
  });
});
