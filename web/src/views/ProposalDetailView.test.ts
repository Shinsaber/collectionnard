import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import type { Proposal, Role } from "@/api/types";
import ProposalDetailView from "./ProposalDetailView.vue";

const api = vi.hoisted(() => ({
  get: vi.fn(),
  conflicts: vi.fn(),
  accept: vi.fn(),
  reject: vi.fn(),
  comments: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  proposalApi: api,
  itemTypeApi: { list: () => Promise.resolve({ data: [], total: 0 }) },
  relationTypeApi: { list: () => Promise.resolve({ data: [], total: 0 }) },
  collectionTypeApi: { list: vi.fn() },
  extractApiError: (e: { message?: string }) => ({ message: e.message ?? "" }),
}));

const proposal: Proposal = {
  id: "p1",
  kind: "catalog",
  status: "pending",
  title: "Corriger le tome",
  author: { username: "camille" },
  comment_count: 0,
  created_at: "",
  updated_at: "",
  payload: {
    items: [
      {
        id: "i1",
        item_type_id: "t1",
        attributes: { title: "Proposé", year: 2000 },
      },
    ],
  },
  base: { items: { i1: { title: "Départ", year: 2000 } } },
};

async function mountAs(username: string, role: Role) {
  useAuthStore().user = {
    id: "u",
    username,
    email: "",
    role,
    created_at: "",
    updated_at: "",
  };
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  const w = mount(ProposalDetailView, {
    props: { id: "p1" },
    global: { plugins: [ElementPlus, router] },
  });
  await flushPromises();
  return w;
}

describe("ProposalDetailView", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    Object.values(api).forEach((m) => m.mockReset());
    api.get.mockResolvedValue(proposal);
    api.comments.mockResolvedValue([]);
    api.conflicts.mockResolvedValue([
      {
        kind: "item",
        id: "i1",
        base: { title: "Départ", year: 2000 },
        current: { title: "Actuel", year: 2001 },
        proposed: { title: "Proposé", year: 2000 },
      },
    ]);
    api.accept.mockResolvedValue({ ...proposal, status: "accepted" });
  });

  it("le modérateur tranche le conflit puis accepte la charge fusionnée", async () => {
    const w = await mountAs("dominique", "contributor");
    expect(w.find('[data-test="proposal-conflicts"]').exists()).toBe(true);
    await w.find('[data-test="pick-current-title"] input').setValue(true);
    await w.find('[data-test="proposal-accept"]').trigger("click");
    await flushPromises();
    expect(api.accept).toHaveBeenCalledWith("p1", {
      payload: {
        items: [
          {
            id: "i1",
            item_type_id: "t1",
            attributes: { title: "Actuel", year: 2001 },
          },
        ],
        relations: [],
      },
      conflicts_resolved: true,
      comment: "",
    });
  });

  it("l'auteur ne modère pas sa proposition ; un membre ne modère pas", async () => {
    let w = await mountAs("camille", "contributor");
    expect(w.find('[data-test="moderation"]').exists()).toBe(false);
    expect(w.find('[data-test="proposal-withdraw"]').exists()).toBe(true);
    expect(api.conflicts).not.toHaveBeenCalled();
    w.unmount();
    w = await mountAs("autre", "user");
    expect(w.find('[data-test="moderation"]').exists()).toBe(false);
  });
});
