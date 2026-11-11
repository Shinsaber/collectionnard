import { describe, expect, it } from "vitest";
import type { Loan } from "@/api/types";
import { availableCopies, borrowerName, lentCopies, today } from "./loans";

const loan = (over: Partial<Loan>): Loan =>
  ({ quantity: 1, borrower_name: "Odile", ...over }) as Loan;

describe("loans", () => {
  it("compte les exemplaires prêtés et disponibles", () => {
    const lot = { quantity: 5, loans: [loan({ quantity: 2 }), loan({})] };
    expect(lentCopies(lot)).toBe(3);
    expect(availableCopies(lot)).toBe(2);
    expect(availableCopies({ quantity: 2 })).toBe(2);
  });

  it("nomme l'ami tel qu'il se présente, sinon le nom figé", () => {
    expect(
      borrowerName(
        loan({ borrower: { username: "cam", display_name: "Camille" } }),
      ),
    ).toBe("Camille");
    expect(borrowerName(loan({ borrower: { username: "cam" } }))).toBe("cam");
    expect(borrowerName(loan({}))).toBe("Odile");
  });

  it("formate la date du jour en heure locale", () => {
    expect(today(new Date(2026, 0, 5, 23, 30))).toBe("2026-01-05");
  });
});
