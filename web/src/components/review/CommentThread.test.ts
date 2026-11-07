import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Comment, Review } from "@/api/types";

const api = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
  setClosed: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  commentApi: api,
  reportApi: { create: vi.fn() },
  extractApiError: () => ({ message: "" }),
}));

import CommentThread from "./CommentThread.vue";

const review = {
  id: "r1",
  item_id: "i1",
  rating: 8,
  body: "x",
  spoiler: false,
  lang: "fr",
  visibility: "anonymous",
  comments_closed: false,
  created_at: "",
  updated_at: "",
} as Review;

function comment(over: Partial<Comment>): Comment {
  return {
    id: "c" + Math.random(),
    target_kind: "review",
    target_id: "r1",
    target_author: false,
    mine: false,
    body: "texte",
    created_at: "2026-11-07T10:00:00Z",
    updated_at: "2026-11-07T10:00:00Z",
    ...over,
  };
}

function mountThread(
  props: Partial<{
    canWrite: boolean;
    ownTarget: boolean;
    target: Review;
  }> = {},
) {
  return mount(CommentThread, {
    props: { target: review, canWrite: true, ...props },
    global: { plugins: [ElementPlus] },
  });
}

describe("CommentThread (I-2)", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    Object.values(api).forEach((m) => m.mockReset());
  });

  it("nomme les membres, jamais l'auteur d'un avis anonyme", async () => {
    api.list.mockResolvedValue({
      data: [
        comment({ author: { username: "camille" }, body: "Bien vu" }),
        comment({ target_author: true, body: "Merci" }),
      ],
      total: 2,
    });
    const w = mountThread();
    await flushPromises();
    const authors = w
      .findAll('[data-test="comment-author"]')
      .map((a) => a.text());
    expect(authors).toEqual(["camille", "Auteur de l'avis"]);
    expect(w.find('[data-test="comment-body"]').html()).toContain("Bien vu");
  });

  it("publie un commentaire puis relit le fil", async () => {
    api.list.mockResolvedValue({ data: [], total: 0 });
    api.create.mockResolvedValue(comment({ mine: true }));
    const w = mountThread();
    await flushPromises();
    await w
      .find('[data-test="comment-form"] textarea')
      .setValue("  Bonne remarque ");
    await w.find('[data-test="comment-form"]').trigger("submit");
    await flushPromises();
    expect(api.create).toHaveBeenCalledWith(
      "review",
      "r1",
      "  Bonne remarque ",
    );
    expect(api.list).toHaveBeenCalledTimes(2);
  });

  it("un fil fermé ne propose plus d'écrire ; l'auteur le rouvre", async () => {
    api.list.mockResolvedValue({ data: [], total: 0 });
    api.setClosed.mockResolvedValue(undefined);
    const w = mountThread({
      target: { ...review, comments_closed: true },
      ownTarget: true,
    });
    await flushPromises();
    expect(w.find('[data-test="comment-form"]').exists()).toBe(false);
    expect(w.find('[data-test="comments-closed"]').exists()).toBe(true);
    await w.find('[data-test="comments-toggle-closed"]').trigger("click");
    await flushPromises();
    expect(api.setClosed).toHaveBeenCalledWith("review", "r1", false);
    expect(w.find('[data-test="comment-form"]').exists()).toBe(true);
  });

  it("montre « Un membre » pour un auteur au nom retiré par le serveur", async () => {
    api.list.mockResolvedValue({
      data: [comment({ body: "discret" })],
      total: 1,
    });
    const w = mountThread();
    await flushPromises();
    expect(w.find('[data-test="comment-author"]').text()).toBe("Un membre");
  });

  it("garde la trace d'un commentaire supprimé ou masqué, et marque les modifications", async () => {
    api.list.mockResolvedValue({
      data: [
        comment({ deleted_at: "2026-11-07T11:00:00Z", body: "" }),
        comment({ hidden_at: "2026-11-07T11:00:00Z", body: "" }),
        comment({
          author: { username: "camille" },
          edited_at: "2026-11-07T12:00:00Z",
        }),
      ],
      total: 3,
    });
    const w = mountThread();
    await flushPromises();
    const traces = w
      .findAll('[data-test="comment-trace"]')
      .map((e) => e.text());
    expect(traces[0]).toContain("supprimé par son auteur");
    expect(traces[1]).toContain("masqué par la modération");
    expect(w.findAll('[data-test="comment-author"]')).toHaveLength(1);
    expect(w.find('[data-test="comment-edited"]').text()).toBe("(modifié)");
    expect(w.emitted("count")?.[0]).toEqual([1]);
  });

  it("en visiteur, ni formulaire ni signalement", async () => {
    api.list.mockResolvedValue({
      data: [comment({ author: { username: "camille" } })],
      total: 1,
    });
    const w = mountThread({ canWrite: false });
    await flushPromises();
    expect(w.find('[data-test="comment-form"]').exists()).toBe(false);
    expect(w.find('[data-test="comment-report"]').exists()).toBe(false);
  });
});
