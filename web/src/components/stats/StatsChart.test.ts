import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import StatsChart from "./StatsChart.vue";

describe("StatsChart", () => {
  it("dessine en SVG et résume les valeurs pour les lecteurs d'écran", async () => {
    const w = mount(StatsChart, {
      props: {
        kind: "bar",
        title: "État",
        labels: ["Neuf", "Bon"],
        values: [3, 1.5],
      },
      attachTo: document.body,
    });
    await flushPromises();
    expect(w.attributes("role")).toBe("img");
    expect(w.attributes("aria-label")).toBe("État : Neuf 3, Bon 1,5");
    expect(w.find("svg").exists()).toBe(true);
    w.unmount();
  });
});
