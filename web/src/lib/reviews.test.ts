import { describe, expect, it } from "vitest";
import {
  contextLabel,
  criteriaAverage,
  criteriaKeys,
  defaultReviewVisibility,
  ratingColor,
  ratingTone,
  starsFill,
} from "./reviews";

describe("reviews", () => {
  it("publie par défaut seulement si le profil est public", () => {
    expect(defaultReviewVisibility("public")).toBe("public");
    expect(defaultReviewVisibility("members")).toBe("anonymous");
    expect(defaultReviewVisibility("private")).toBe("anonymous");
    expect(defaultReviewVisibility(undefined)).toBe("anonymous");
  });

  it("suit l'ordre du schéma pour les critères", () => {
    expect(
      criteriaKeys({
        "x-order": ["story", "art"],
        properties: { art: {}, story: {} },
      }),
    ).toEqual(["story", "art"]);
    expect(criteriaKeys({ type: "object", properties: {} })).toEqual([]);
  });

  it("calcule la moyenne à une décimale, comme le serveur", () => {
    expect(criteriaAverage({ a: 8, b: 7 }, ["a", "b"])).toBe(7.5);
    expect(criteriaAverage({ a: 10, b: 10, c: 9 }, ["a", "b", "c"])).toBe(9.7);
    expect(criteriaAverage({}, [])).toBe(0);
  });

  it("remplit les étoiles par demi-étoile, dans l'échelle", () => {
    expect(starsFill(10)).toBe(100);
    expect(starsFill(7.5)).toBe(80);
    expect(starsFill(0)).toBe(0);
    expect(starsFill(12)).toBe(100);
  });

  it("prend le libellé de l'édition dans la langue courante", () => {
    const label = { fr: "Joué sur", en: "Played on" };
    expect(contextLabel(label, "en", "Édition")).toBe("Played on");
    expect(contextLabel({ fr: "Joué sur" }, "en", "Édition")).toBe("Joué sur");
    expect(contextLabel(undefined, "fr", "Édition")).toBe("Édition");
  });

  it("colore par paliers : 0–4, 5–7, 8–10, note arrondie", () => {
    expect([0, 4, 4.4].map(ratingTone)).toEqual(["low", "low", "low"]);
    expect([4.5, 5, 7, 7.4].map(ratingTone)).toEqual([
      "mid",
      "mid",
      "mid",
      "mid",
    ]);
    expect([7.5, 8, 10].map(ratingTone)).toEqual(["high", "high", "high"]);
    expect(ratingColor(8, "steps")).toBe("var(--c-star-high)");
  });

  it("colore en dégradé, du rouge au vert en passant par l'orange", () => {
    expect(ratingColor(0, "gradient")).toBe(
      "color-mix(in oklch, var(--c-star-mid) 0%, var(--c-star-low))",
    );
    expect(ratingColor(6, "gradient")).toContain("var(--c-star-mid) 100%");
    expect(ratingColor(7.5, "gradient")).toBe(
      "color-mix(in oklch, var(--c-star-high) 50%, var(--c-star-mid))",
    );
    expect(ratingColor(10, "gradient")).toContain("var(--c-star-high) 100%");
  });
});
