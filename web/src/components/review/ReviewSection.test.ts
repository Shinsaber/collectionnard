import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { defineComponent } from "vue";
import { useAuthStore } from "@/stores/auth";
import type { Review, ReviewSummary, User } from "@/api/types";

const api = vi.hoisted(() => ({
  list: vi.fn(),
  mine: vi.fn(),
  put: vi.fn(),
  remove: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  reviewApi: api,
  extractApiError: () => ({ message: "" }),
}));

import ReviewSection from "./ReviewSection.vue";

const SCHEMA = {
  type: "object",
  "x-order": ["story", "art"],
  "x-labels": { fr: { story: "Scénario", art: "Dessin" } },
  properties: {
    story: { type: "integer", minimum: 0, maximum: 10 },
    art: { type: "integer", minimum: 0, maximum: 10 },
  },
};

function summary(over: Partial<ReviewSummary> = {}): ReviewSummary {
  return {
    enabled: true,
    review_type: {
      id: "f1",
      name: "ComicReview",
      attributes_schema: SCHEMA,
      context_relation_type_ids: [],
      created_at: "",
      updated_at: "",
    },
    count: 2,
    rating: null,
    min_ratings: 5,
    note_only_count: 0,
    criteria: [],
    context_items: [],
    ...over,
  };
}

function review(over: Partial<Review> = {}): Review {
  return {
    id: "r1",
    item_id: "i1",
    rating: 8,
    body: "Très **bon**",
    spoiler: false,
    lang: "fr",
    visibility: "public",
    author: { username: "camille" },
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
    ...over,
  };
}

async function mountSection(s: ReviewSummary = summary(), profile = "private") {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: "/:any(.*)*",
        component: defineComponent({ template: "<div />" }),
      },
    ],
  });
  useAuthStore().user = {
    id: "u1",
    username: "moi",
    profile_visibility: profile,
  } as User;
  const w = mount(ReviewSection, {
    props: { itemId: "i1", summary: s },
    global: { plugins: [ElementPlus, router] },
  });
  await flushPromises();
  return w;
}

describe("ReviewSection", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    Object.values(api).forEach((m) => m.mockReset());
    api.mine.mockResolvedValue(null);
    api.list.mockResolvedValue({ data: [], total: 0 });
  });

  it("cache la moyenne sous le seuil et dit pourquoi", async () => {
    const w = await mountSection();
    expect(w.find('[data-test="review-average"]').exists()).toBe(false);
    expect(w.find('[data-test="review-threshold"]').text()).toContain(
      "2 sur 5",
    );
  });

  it("montre la moyenne et les critères au-dessus du seuil", async () => {
    const w = await mountSection(
      summary({
        count: 6,
        rating: 7.5,
        criteria: [
          { key: "story", count: 5, rating: 8 },
          { key: "art", count: 2, rating: null },
        ],
      }),
    );
    expect(w.find('[data-test="review-average"]').exists()).toBe(true);
    const averages = w.find('[data-test="review-criteria-averages"]').text();
    expect(averages).toContain("Scénario");
    expect(averages).not.toContain("Dessin");
  });

  it("n'affiche un auteur que s'il est signé, et masque le spoiler", async () => {
    api.list.mockResolvedValue({
      data: [
        review(),
        review({
          id: "r2",
          author: undefined,
          visibility: "anonymous",
          spoiler: true,
        }),
      ],
      total: 2,
    });
    const w = await mountSection(summary({ note_only_count: 3 }));
    const cards = w.findAll('[data-test="review-card"]');
    expect(cards[0]!.find('[data-test="review-author"]').text()).toContain(
      "camille",
    );
    expect(cards[0]!.find('[data-test="review-body"]').html()).toContain(
      "<strong>bon</strong>",
    );
    expect(cards[1]!.find('[data-test="review-author"]').exists()).toBe(false);
    expect(cards[1]!.find('[data-test="review-anonymous"]').exists()).toBe(
      true,
    );
    expect(cards[1]!.find('[data-test="review-body"]').exists()).toBe(false);
    await cards[1]!.find('[data-test="review-spoiler"]').trigger("click");
    expect(cards[1]!.find('[data-test="review-body"]').exists()).toBe(true);
    expect(w.find('[data-test="review-note-only"]').text()).toContain("3");
  });

  it("envoie une note globale, anonyme par défaut hors profil public", async () => {
    api.put.mockResolvedValue(review({ visibility: "anonymous" }));
    const w = await mountSection();
    await w.find('[data-test="review-write"]').trigger("click");
    await w.find('[data-test="review-form"]').trigger("submit");
    await flushPromises();
    expect(api.put).toHaveBeenCalledWith("i1", {
      rating: 5,
      body: "",
      spoiler: false,
      lang: "fr",
      visibility: "anonymous",
    });
    expect(w.emitted("changed")).toHaveLength(1);
    expect(w.find('[data-test="review-mine"]').exists()).toBe(true);
  });

  it("publie par défaut si le profil est public", async () => {
    api.put.mockResolvedValue(review());
    const w = await mountSection(summary(), "public");
    await w.find('[data-test="review-write"]').trigger("click");
    await w.find('[data-test="review-form"]').trigger("submit");
    await flushPromises();
    expect(api.put.mock.calls[0]![1].visibility).toBe("public");
  });

  it("envoie tous les critères en mode détaillé, sans note globale", async () => {
    api.put.mockResolvedValue(review());
    const w = await mountSection();
    await w.find('[data-test="review-write"]').trigger("click");
    const radios = w.findAll('[data-test="review-mode"] input');
    await radios[1]!.setValue(true);
    await w.find('[data-test="review-form"]').trigger("submit");
    await flushPromises();
    const payload = api.put.mock.calls[0]![1];
    expect(payload.criteria).toEqual({ story: 5, art: 5 });
    expect(payload).not.toHaveProperty("rating");
  });

  it("propose l'édition utilisée avec son libellé, et prévient en mode anonyme", async () => {
    api.put.mockResolvedValue(review());
    const s = summary({
      context_items: [
        {
          id: "e1",
          item_type_id: "b41",
          attributes: { title: "Édition Switch" },
          created_at: "",
          updated_at: "",
        },
      ],
    });
    s.review_type!.context_label = { fr: "Joué sur", en: "Played on" };
    const w = await mountSection(s);
    expect(w.find('[data-test="review-context-filter"]').exists()).toBe(true);
    await w.find('[data-test="review-write"]').trigger("click");
    const form = w.find('[data-test="review-form"]');
    expect(form.text()).toContain("Joué sur");
    const contextSelect = w
      .findAllComponents({ name: "ElSelect" })
      .find((c) => c.attributes("data-test") === "review-context")!;
    contextSelect.vm.$emit("update:modelValue", "e1");
    await w.vm.$nextTick();
    expect(w.find('[data-test="review-context-help"]').text()).toContain(
      "anonyme",
    );
    await form.trigger("submit");
    await flushPromises();
    expect(api.put.mock.calls[0]![1].context_item_id).toBe("e1");
  });

  it("affiche l'édition utilisée sur un avis", async () => {
    const s = summary();
    s.review_type!.context_label = { fr: "Joué sur" };
    api.list.mockResolvedValue({
      data: [
        review({
          context_item_id: "e1",
          context_item: {
            id: "e1",
            item_type_id: "b41",
            attributes: { title: "Édition Switch" },
            created_at: "",
            updated_at: "",
          },
        }),
      ],
      total: 1,
    });
    const w = await mountSection(s);
    expect(w.find('[data-test="review-context"]').text()).toContain(
      "Joué sur : Édition Switch",
    );
  });

  it("n'offre que la note globale sur un type sans critère", async () => {
    const empty = summary();
    empty.review_type!.attributes_schema = { type: "object", properties: {} };
    const w = await mountSection(empty);
    await w.find('[data-test="review-write"]').trigger("click");
    expect(w.find('[data-test="review-mode"]').exists()).toBe(false);
  });

  it("reprend un avis détaillé existant à la modification", async () => {
    api.mine.mockResolvedValue(
      review({
        criteria: { story: 9, art: 7 },
        rating: 8,
        review_type_id: "f1",
      }),
    );
    const w = await mountSection();
    expect(w.find('[data-test="review-criteria"]').text()).toContain(
      "Scénario",
    );
    await w.find('[data-test="review-edit"]').trigger("click");
    expect(w.find('[data-test="review-criteria-form"]').exists()).toBe(true);
  });
});
