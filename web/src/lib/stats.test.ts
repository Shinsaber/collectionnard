import { describe, expect, it } from "vitest";
import {
  distributionSeries,
  formatNumber,
  monthLabel,
  monthSeries,
  percent,
} from "./stats";
import type { Schema } from "./schema";

const schema: Schema = {
  type: "object",
  properties: {
    condition: {
      type: "string",
      enum: ["new", "good"],
      "x-enum-labels": { fr: { new: "Neuf", good: "Bon" }, en: { new: "New" } },
    },
    signed: { type: ["boolean", "null"] },
  },
};
const words = { yes: "Oui", no: "Non", missing: "Non renseigné" };

describe("stats", () => {
  it("met en forme les mois selon la langue", () => {
    expect(monthLabel("2026-10", "fr")).toBe("oct. 2026");
    expect(monthLabel("2026-10", "en")).toBe("Oct 2026");
    expect(monthLabel("n/a", "fr")).toBe("n/a");
    expect(
      monthSeries(
        [
          { month: "2026-01", count: 2 },
          { month: "2026-02", count: 0 },
        ],
        "fr",
      ),
    ).toEqual({ labels: ["janv. 2026", "févr. 2026"], values: [2, 0] });
  });

  it("traduit les valeurs d'enum et ajoute les valeurs manquantes", () => {
    const s = distributionSeries(
      {
        field: "condition",
        values: [
          { value: "new", count: 3 },
          { value: "good", count: 0 },
          { value: "mint", count: 1 },
        ],
        missing: 2,
      },
      schema,
      "fr",
      words,
    );
    expect(s).toEqual({
      labels: ["Neuf", "Bon", "mint", "Non renseigné"],
      values: [3, 0, 1, 2],
    });
  });

  it("affiche un booléen en oui / non", () => {
    const s = distributionSeries(
      {
        field: "signed",
        values: [
          { value: "true", count: 1 },
          { value: "false", count: 4 },
        ],
        missing: 0,
      },
      schema,
      "fr",
      words,
    );
    expect(s.labels).toEqual(["Oui", "Non"]);
  });

  it("calcule la part possédée", () => {
    expect(percent(30, 42)).toBe(71);
    expect(percent(0, 0)).toBe(0);
    expect(formatNumber(1234.567, "fr")).toBe("1\u202f234,57");
  });
});
