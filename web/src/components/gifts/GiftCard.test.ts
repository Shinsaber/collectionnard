import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { GiftReservation } from "@/api/types";
import GiftCard from "./GiftCard.vue";

const api = vi.hoisted(() => ({ update: vi.fn(), cancel: vi.fn() }));
vi.mock("@/api/client", () => ({
  giftApi: api,
  extractApiError: () => ({ message: "" }),
}));

const gift: GiftReservation = {
  id: "g1",
  lot_id: "l1",
  left_list: false,
  item_id: "e1",
  quantity: 1,
  status: "reserved",
  expires_at: "2026-11-01",
  expired: true,
  mine: true,
  note: "la bleue",
  owner: { username: "user", display_name: "Sam" },
  created_at: "",
  updated_at: "",
};

function mountCard(g: GiftReservation) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  return mount(GiftCard, {
    props: { gift: g },
    global: { plugins: [ElementPlus, router] },
  });
}

describe("GiftCard", () => {
  beforeEach(() => api.update.mockReset().mockResolvedValue({}));

  it("une réservation expirée se signale et se prolonge, le reste gardé", async () => {
    const w = mountCard(gift);
    expect(w.text()).toContain("Pour Sam");
    expect(w.find('[data-test="gift-expired"]').exists()).toBe(true);
    const input = w.find(
      '[data-test="gift-until"] input, input[data-test="gift-until"]',
    );
    await input.setValue("2026-12-24");
    await input.trigger("change");
    await flushPromises();
    expect(api.update).toHaveBeenCalledWith("g1", {
      quantity: 1,
      status: "reserved",
      expires_at: "2026-12-24",
      note: "la bleue",
    });
  });

  it("offert : plus d'échéance à prolonger", () => {
    const w = mountCard({ ...gift, status: "given", expired: false });
    expect(w.find('[data-test="gift-until"]').exists()).toBe(false);
    expect(w.text()).toContain("Offert");
  });
});
