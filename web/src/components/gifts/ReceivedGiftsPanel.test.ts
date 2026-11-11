import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { GiftReservation } from "@/api/types";
import ReceivedGiftsPanel from "./ReceivedGiftsPanel.vue";

const api = vi.hoisted(() => ({ received: vi.fn() }));
vi.mock("@/api/client", () => ({
  giftApi: api,
  extractApiError: () => ({ message: "" }),
}));
const decline = vi.hoisted(() => vi.fn());
vi.mock("@/composables/useGiftDecline", () => ({
  useGiftDecline: () => ({ decline }),
}));

const gift = (over: Partial<GiftReservation>): GiftReservation =>
  ({
    id: "g",
    item_id: "e1",
    quantity: 1,
    status: "received",
    left_list: true,
    expired: false,
    mine: false,
    expires_at: "2027-01-01",
    giver: { username: "cam", display_name: "Camille" },
    received_at: "2026-11-10T10:00:00Z",
    created_at: "",
    updated_at: "",
    ...over,
  }) as GiftReservation;
const page = (data: GiftReservation[]) => ({
  data,
  total: data.length,
  page: 1,
  per_page: 20,
  total_pages: 1,
});

async function mountPanel() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  const w = mount(ReceivedGiftsPanel, {
    global: {
      plugins: [ElementPlus, router],
      stubs: { ReceiveGiftDialog: true, ItemCover: true },
    },
  });
  await flushPromises();
  return w;
}

describe("ReceivedGiftsPanel", () => {
  beforeEach(() => {
    api.received.mockReset();
    decline.mockReset().mockResolvedValue(true);
    api.received.mockImplementation(({ status }: { status: string }) =>
      Promise.resolve(
        page(
          status === "given"
            ? [
                gift({
                  id: "g1",
                  status: "given",
                  left_list: false,
                  collection_id: "wish",
                  given_at: "2026-11-09T10:00:00Z",
                }),
              ]
            : [gift({ id: "g2" })],
        ),
      ),
    );
  });

  it("montre les remises à confirmer, puis l'historique des cadeaux reçus", async () => {
    const w = await mountPanel();
    const pending = w.find('[data-test="gifts-to-confirm"]');
    expect(pending.text()).toContain("Offert par Camille");
    expect(pending.text()).toContain("à confirmer");
    expect(pending.find('[data-test="gift-receive"]').exists()).toBe(true);
    const history = w.find('[data-test="gifts-received"]');
    expect(history.text()).toContain("Reçu le 10/11/2026");
    expect(history.find('[data-test="gift-receive"]').exists()).toBe(false);
  });

  it("« Pas reçu » passe par la liste du cadeau, puis recharge", async () => {
    const w = await mountPanel();
    await w.find('[data-test="gift-decline"]').trigger("click");
    await flushPromises();
    expect(decline).toHaveBeenCalledWith(
      "wish",
      expect.objectContaining({ id: "g1" }),
    );
    expect(api.received).toHaveBeenCalledTimes(4);
  });
});
