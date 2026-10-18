import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import NotificationBell from "./NotificationBell.vue";

const api = vi.hoisted(() => ({
  list: vi.fn(),
  read: vi.fn(),
  reviewCount: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  notificationApi: { list: api.list, read: api.read },
  proposalApi: { reviewCount: api.reviewCount },
  extractApiError: () => ({ message: "" }),
}));

async function mountBell(role: "user" | "contributor") {
  useAuthStore().user = {
    id: "u",
    username: "u",
    email: "",
    role,
    created_at: "",
    updated_at: "",
  };
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  await router.push("/");
  const w = mount(NotificationBell, {
    global: { plugins: [ElementPlus, router] },
    attachTo: document.body,
  });
  await flushPromises();
  return w;
}

describe("NotificationBell", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    document.body.innerHTML = "";
    Object.values(api).forEach((m) => m.mockReset());
    api.read.mockResolvedValue(undefined);
    api.reviewCount.mockResolvedValue(3);
    api.list.mockResolvedValue({
      data: [
        {
          id: "n1",
          kind: "proposal_accepted",
          payload: { proposal_id: "p1", title: "Tome 3", by: "dominique" },
          created_at: "2026-10-18T10:00:00Z",
        },
      ],
      total: 1,
      page: 1,
      per_page: 10,
      unread: 1,
    });
  });

  it("compte les non-lues, et les propositions à relire pour un modérateur", async () => {
    const w = await mountBell("contributor");
    expect(
      w.find('[data-test="notification-bell"]').attributes("aria-label"),
    ).toBe("Notifications (4 non lues)");
    await w.find('[data-test="notification-bell"]').trigger("click");
    await flushPromises();
    const panel = document.querySelector('[data-test="notification-panel"]')!;
    expect(panel.textContent).toContain("dominique a accepté « Tome 3 »");
    expect(panel.textContent).toContain("3 propositions à relire");
  });

  it("un membre ne demande pas la file à relire ; un clic marque comme lu", async () => {
    const w = await mountBell("user");
    expect(api.reviewCount).not.toHaveBeenCalled();
    await w.find('[data-test="notification-bell"]').trigger("click");
    await flushPromises();
    document
      .querySelector<HTMLElement>('[data-test="notification-panel"] a')!
      .click();
    await flushPromises();
    expect(api.read).toHaveBeenCalledWith(["n1"]);
  });
});
