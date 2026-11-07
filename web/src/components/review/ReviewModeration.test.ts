import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Review } from "@/api/types";

const api = vi.hoisted(() => ({ react: vi.fn(), report: vi.fn() }));
vi.mock("@/api/client", () => ({
  reviewApi: { react: api.react },
  reportApi: { create: api.report },
  extractApiError: () => ({ message: "" }),
}));

import ReviewCard from "./ReviewCard.vue";

function review(over: Partial<Review> = {}): Review {
  return {
    id: "r1",
    item_id: "i1",
    rating: 8,
    body: "Très bon",
    spoiler: false,
    lang: "fr",
    visibility: "public",
    author: { username: "camille" },
    reactions: { interesting: 2 },
    my_reactions: [],
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
    ...over,
  };
}

function mountCard(props: {
  review: Review;
  reportable?: boolean;
  showReactions?: boolean;
}) {
  return mount(ReviewCard, {
    props,
    global: {
      plugins: [ElementPlus],
      stubs: { RouterLink: { template: "<a><slot /></a>" } },
    },
  });
}

describe("ReviewCard — réactions et signalement (I-1)", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    api.react.mockReset().mockResolvedValue(undefined);
    api.report.mockReset().mockResolvedValue({ id: "p1" });
  });

  it("ajoute puis retire une réaction, compteur mis à jour", async () => {
    const w = mountCard({
      review: review(),
      reportable: true,
      showReactions: true,
    });
    const btn = w.find('[data-test="reaction-interesting"]');
    expect(btn.text()).toContain("2");
    await btn.trigger("click");
    await flushPromises();
    expect(api.react).toHaveBeenCalledWith("r1", "interesting", true);
    expect(btn.text()).toContain("3");
    expect(btn.attributes("aria-pressed")).toBe("true");
    await btn.trigger("click");
    await flushPromises();
    expect(api.react).toHaveBeenLastCalledWith("r1", "interesting", false);
    expect(btn.text()).toContain("2");
  });

  it("en lecture seule, ne montre que les réactions reçues et pas de signalement", () => {
    const w = mountCard({ review: review(), showReactions: true });
    expect(w.findAll("button[data-test^='reaction-']")).toHaveLength(0);
    expect(w.find('[data-test="reaction-interesting"]').exists()).toBe(true);
    expect(w.find('[data-test="reaction-funny"]').exists()).toBe(false);
    expect(w.find('[data-test="review-report"]').exists()).toBe(false);
  });

  it("signale avec un motif et des précisions", async () => {
    const w = mountCard({
      review: review(),
      reportable: true,
      showReactions: true,
    });
    await w.find('[data-test="review-report"]').trigger("click");
    await flushPromises();
    const dialog = document.body;
    const radio = [
      ...dialog.querySelectorAll<HTMLInputElement>("input[type=radio]"),
    ].find((r) => r.value === "spoiler")!;
    radio.click();
    await flushPromises();
    (
      dialog.querySelector('[data-test="report-form"]') as HTMLFormElement
    ).dispatchEvent(new Event("submit"));
    await flushPromises();
    expect(api.report).toHaveBeenCalledWith({
      target_kind: "review",
      target_id: "r1",
      reason: "spoiler",
      details: "",
    });
    w.unmount();
  });

  it("montre à son auteur qu'un avis est masqué, avec le motif", () => {
    const w = mountCard({
      review: review({
        hidden_at: "2026-11-07T00:00:00Z",
        hidden_reason: "Propos injurieux",
      }),
    });
    expect(w.find('[data-test="review-hidden"]').text()).toContain(
      "Propos injurieux",
    );
  });
});
