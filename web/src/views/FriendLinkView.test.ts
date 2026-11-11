import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import FriendLinkView from "./FriendLinkView.vue";

const api = vi.hoisted(() => ({ open: vi.fn(), accept: vi.fn() }));
vi.mock("@/api/client", () => ({
  friendLinkApi: api,
  extractApiError: () => ({ message: "" }),
}));

const link = {
  id: "l1",
  user: { username: "camille", display_name: "Camille" },
  created_at: "2026-11-08T00:00:00Z",
  expires_at: "2026-11-15T00:00:00Z",
};

async function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  const push = vi.spyOn(router, "push");
  const w = mount(FriendLinkView, {
    props: { token: "tok" },
    global: {
      plugins: [ElementPlus, router],
      stubs: { PageHeader: true, UserAvatar: true },
    },
  });
  await flushPromises();
  return { w, push };
}

describe("FriendLinkView", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    api.open.mockReset();
    api.accept.mockReset();
  });

  it("montre qui invite, puis accepte et mène aux amis", async () => {
    api.open.mockResolvedValue(link);
    api.accept.mockResolvedValue({});
    const { w, push } = await mountView();
    expect(api.open).toHaveBeenCalledWith("tok");
    expect(w.find('[data-test="friend-link-invites"]').text()).toContain(
      "Camille",
    );
    await w.find('[data-test="friend-link-accept"]').trigger("click");
    await flushPromises();
    expect(api.accept).toHaveBeenCalledWith("tok");
    expect(push).toHaveBeenCalledWith("/members");
  });

  it("lien invalide : un seul message", async () => {
    api.open.mockRejectedValue({ response: { status: 404 } });
    const { w } = await mountView();
    expect(w.find('[data-test="friend-link-invalid"]').exists()).toBe(true);
  });

  it("son propre lien ne s'accepte pas", async () => {
    api.open.mockResolvedValue(link);
    useAuthStore().user = {
      username: "camille",
    } as unknown as ReturnType<typeof useAuthStore>["user"];
    const { w } = await mountView();
    expect(w.find('[data-test="friend-link-accept"]').exists()).toBe(false);
  });
});
