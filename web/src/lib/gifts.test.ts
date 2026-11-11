import { describe, expect, it } from "vitest";
import type { GiftReservation } from "@/api/types";
import {
  defaultExpiry,
  maxExpiry,
  myReservation,
  reservableCopies,
  reservedCopies,
} from "./gifts";

const r = (over: Partial<GiftReservation>) =>
  ({ quantity: 1, mine: false, ...over }) as GiftReservation;

describe("gifts", () => {
  it("compte les exemplaires réservés et ceux qui restent", () => {
    const mine = r({ quantity: 2, mine: true });
    const lot = { quantity: 4, reservations: [r({}), mine] };
    expect(reservedCopies(lot)).toBe(3);
    expect(reservableCopies(lot)).toBe(1);
    expect(reservableCopies(lot, mine)).toBe(3);
    expect(myReservation(lot)).toBe(mine);
    expect(reservableCopies({ quantity: 2 })).toBe(2);
  });
});

describe("échéance d'une réservation", () => {
  it("60 jours par défaut, un an au plus", () => {
    const now = new Date(2026, 10, 8);
    expect(defaultExpiry(now)).toBe("2027-01-07");
    expect(maxExpiry(now)).toBe("2027-11-08");
  });
});
