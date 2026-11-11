import { mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { describe, expect, it } from "vitest";
import type { CollectionItem, GiftReservation } from "@/api/types";
import ReservationBlock from "./ReservationBlock.vue";

const res = (over: Partial<GiftReservation>) =>
  ({
    id: "r",
    quantity: 1,
    mine: false,
    status: "reserved",
    ...over,
  }) as GiftReservation;
const lot = (quantity: number, reservations: GiftReservation[]) =>
  ({ id: "l1", quantity, reservations }) as unknown as CollectionItem;

const mountBlock = (l: CollectionItem) =>
  mount(ReservationBlock, {
    props: { lot: l },
    global: { plugins: [ElementPlus] },
  });

describe("ReservationBlock", () => {
  it("nomme les offrants, moi compris, avec le compte", () => {
    const w = mountBlock(
      lot(3, [
        res({ giver: { username: "cam", display_name: "Camille" } }),
        res({ mine: true }),
      ]),
    );
    expect(w.find('[data-test="reservation-flag"]').text()).toBe(
      "Camille, vous · 2 sur 3",
    );
    expect(w.find('[data-test="reserve"]').text()).toBe("Ma réservation");
  });

  it("propose de réserver tant qu'il reste des exemplaires", async () => {
    const w = mountBlock(lot(1, []));
    expect(w.find('[data-test="reservation-flag"]').exists()).toBe(false);
    await w.find('[data-test="reserve"]').trigger("click");
    expect(w.emitted("reserve")).toBeTruthy();
  });

  it("un cadeau remis se signale aux autres amis, et ma remise ne se modifie plus", () => {
    const w = mountBlock(
      lot(2, [
        res({ giver: { username: "cam" }, status: "given" }),
        res({ mine: true, status: "given" }),
      ]),
    );
    expect(w.find('[data-test="reservation-flag"]').text()).toBe(
      "cam (offert), vous (offert) · 2 sur 2",
    );
    expect(w.find('[data-test="reserve"]').exists()).toBe(false);
    expect(w.find('[data-test="reserve-given"]').exists()).toBe(true);
  });

  it("tout réservé par d'autres : plus de bouton", () => {
    const w = mountBlock(lot(1, [res({ giver: { username: "cam" } })]));
    expect(w.find('[data-test="reserve"]').exists()).toBe(false);
    expect(w.text()).toContain("Tout est réservé");
  });
});
