import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { GiftReservation } from "@/api/types";
import GiftCard from "./GiftCard.vue";

const api = vi.hoisted(() => ({
  update: vi.fn(),
  cancel: vi.fn(),
  handOver: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  giftApi: api,
  extractApiError: () => ({ message: "" }),
}));
const confirm = vi.hoisted(() => vi.fn());
vi.mock("@/composables/useConfirmDialog", () => ({
  useConfirmDialog: () => ({ confirm }),
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
  beforeEach(() => {
    Object.values(api).forEach((m) => m.mockReset().mockResolvedValue({}));
    confirm.mockReset().mockResolvedValue(true);
  });

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
      expires_at: "2026-12-24",
      note: "la bleue",
    });
  });

  it("« Je l'ai remis » demande confirmation, puis prévient le propriétaire", async () => {
    const w = mountCard({ ...gift, expired: false });
    await w.find('[data-test="gift-hand-over"]').trigger("click");
    await flushPromises();
    expect(confirm).toHaveBeenCalled();
    expect(api.handOver).toHaveBeenCalledWith("g1");
    expect(w.emitted("changed")).toBeTruthy();
  });

  it("remis : en attente, plus d'échéance ni de remise", () => {
    const w = mountCard({ ...gift, status: "given", expired: false });
    expect(w.find('[data-test="gift-given"]').exists()).toBe(true);
    expect(w.find('[data-test="gift-until"]').exists()).toBe(false);
    expect(w.find('[data-test="gift-hand-over"]').exists()).toBe(false);
  });

  it("reçu : la date de réception, et rien à retirer (P7-4c)", () => {
    const w = mountCard({
      ...gift,
      status: "received",
      expired: false,
      received_at: "2026-11-10T10:00:00Z",
    });
    expect(w.find('[data-test="gift-received"]').text()).toContain(
      "10/11/2026",
    );
    expect(w.find('[data-test="gift-cancel"]').exists()).toBe(false);
  });
});
