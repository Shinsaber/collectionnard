import { describe, expect, it } from "vitest";
import {
  defaultSelection,
  effectiveAttributes,
  groupPlan,
  parentRelation,
  selectedOverrides,
  writtenCount,
} from "./externalPlan";
import { dragonBallPlan } from "../../tests/fixtures/external-plan";

describe("externalPlan", () => {
  it("regroupe l'œuvre et ses auteurs, puis chaque édition avec éditeurs et tomes", () => {
    const g = groupPlan(dragonBallPlan);
    expect(g.main.map((n) => n.key)).toEqual([
      "wikidata:Q15051307",
      "wikidata:Q208582",
    ]);
    expect(g.editions).toHaveLength(1);
    expect(g.editions[0]!.edition.key).toBe("bnf:cb343384394");
    expect(g.editions[0]!.publishers.map((n) => n.key)).toEqual([
      "publisher:glénat",
    ]);
    expect(g.editions[0]!.volumes).toHaveLength(3);
  });

  it("sélectionne par défaut tout sauf les réimpressions", () => {
    const keys = defaultSelection(dragonBallPlan);
    expect(keys).toHaveLength(6);
    expect(keys).not.toContain("bnf:cb43462118n");
  });

  it("trouve la relation au parent dans les deux sens", () => {
    const person = dragonBallPlan.nodes[1]!;
    expect(parentRelation(dragonBallPlan, person)?.attributes).toEqual({
      roles: ["author", "artist"],
    });
    expect(parentRelation(dragonBallPlan, dragonBallPlan.nodes[0]!)).toBe(
      undefined,
    );
  });

  it("ne compte comme écrits que les nœuds nouveaux ou à compléter", () => {
    // Toriyama est à jour : sélectionné mais rien à écrire.
    expect(writtenCount(dragonBallPlan, defaultSelection(dragonBallPlan))).toBe(
      5,
    );
    expect(writtenCount(dragonBallPlan, [])).toBe(0);
  });

  it("applique les corrections et ne garde que celles des nœuds sélectionnés", () => {
    const toriyama = dragonBallPlan.nodes[1]!;
    const overrides = {
      [toriyama.key]: { last_name: "Toriyama", first_name: "A." },
    };
    expect(effectiveAttributes(toriyama, overrides)).toEqual({
      last_name: "Toriyama",
      first_name: "A.",
    });
    expect(effectiveAttributes(toriyama)).toBe(toriyama.attributes);
    // Toriyama est à jour, mais corrigé : il sera écrit.
    const keys = defaultSelection(dragonBallPlan);
    expect(writtenCount(dragonBallPlan, keys, overrides)).toBe(6);
    expect(selectedOverrides(overrides, keys)).toEqual(overrides);
    expect(selectedOverrides(overrides, ["wikidata:Q15051307"])).toEqual({});
  });
});
