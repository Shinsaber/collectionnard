import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { Friendship } from "@/api/types";
import FriendsPanel from "./FriendsPanel.vue";

const api = vi.hoisted(() => ({
  list: vi.fn(),
  requests: vi.fn(),
  request: vi.fn(),
  accept: vi.fn(),
  remove: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  friendApi: api,
  friendLinkApi: { list: vi.fn().mockResolvedValue([]) },
  extractApiError: () => ({ message: "" }),
}));

const friendship = (over: Partial<Friendship>): Friendship => ({
  id: "f1",
  status: "accepted",
  incoming: false,
  user: { username: "camille", display_name: "Camille" },
  profile_visible: true,
  created_at: "2026-11-01T00:00:00Z",
  accepted_at: "2026-11-02T00:00:00Z",
  ...over,
});
const page = (data: Friendship[]) => ({
  data,
  total: data.length,
  page: 1,
  per_page: 20,
  total_pages: 1,
});

async function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  const w = mount(FriendsPanel, {
    global: {
      plugins: [ElementPlus, router],
      stubs: { UserAvatar: true },
    },
  });
  await flushPromises();
  return w;
}

describe("FriendsPanel", () => {
  beforeEach(() => {
    Object.values(api).forEach((m) => m.mockReset());
    api.list.mockResolvedValue(
      page([
        friendship({}),
        friendship({
          id: "f2",
          user: { username: "dom" },
          profile_visible: false,
        }),
      ]),
    );
    api.requests.mockResolvedValue(
      page([
        friendship({
          id: "r1",
          status: "pending",
          incoming: true,
          user: { username: "eve" },
        }),
        friendship({
          id: "r2",
          status: "pending",
          user: { username: "zoe" },
        }),
      ]),
    );
  });

  it("sépare les demandes reçues et envoyées, et ne lie que les profils ouverts", async () => {
    const w = await mountView();
    const received = w.find('[data-test="friend-requests-received"]');
    expect(received.text()).toContain("eve");
    expect(received.find('[data-test="friend-accept"]').exists()).toBe(true);
    const sent = w.find('[data-test="friend-requests-sent"]');
    expect(sent.text()).toContain("zoe");
    expect(sent.find('[data-test="friend-cancel"]').exists()).toBe(true);

    const friends = w.findAll('[data-test="friend"]');
    expect(friends).toHaveLength(2);
    expect(friends[0]!.find("a").attributes("href")).toBe("/u/camille");
    expect(friends[1]!.find("a").exists()).toBe(false);
  });

  it("invite par nom d'utilisateur puis recharge", async () => {
    api.request.mockResolvedValue(
      friendship({ id: "r3", status: "pending", user: { username: "lou" } }),
    );
    const w = await mountView();
    await w.find("form input").setValue("  lou ");
    await w.find("form").trigger("submit");
    await flushPromises();
    expect(api.request).toHaveBeenCalledWith("lou");
    expect(api.list).toHaveBeenCalledTimes(2);
  });

  it("accepte une demande reçue", async () => {
    api.accept.mockResolvedValue(friendship({ id: "r1" }));
    const w = await mountView();
    await w.find('[data-test="friend-accept"]').trigger("click");
    await flushPromises();
    expect(api.accept).toHaveBeenCalledWith("r1");
  });

  it("sans ami, invite à en ajouter", async () => {
    api.list.mockResolvedValue(page([]));
    api.requests.mockResolvedValue(page([]));
    const w = await mountView();
    expect(w.find('[data-test="friends-none"]').exists()).toBe(true);
    expect(w.find('[data-test="friend-requests-received"]').exists()).toBe(
      false,
    );
  });
});
