import { describe, expect, it } from "vitest";
import { components } from "./elementPlus";

// Audit Q-002 : Element Plus n'est plus installé en bloc. Ce test garde la liste des
// composants enregistrés alignée sur les balises `<El…>` des gabarits.
const sources = import.meta.glob("/src/**/*.vue", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

describe("plugins/elementPlus", () => {
  it("enregistre chaque composant <El…> des gabarits, et rien de plus", () => {
    const used = new Set<string>();
    for (const src of Object.values(sources)) {
      for (const m of src.matchAll(/<(El[A-Z][A-Za-z]+)/g)) used.add(m[1]!);
    }
    const registered = new Set(Object.keys(components));
    expect([...used].filter((n) => !registered.has(n)).sort()).toEqual([]);
    expect([...registered].filter((n) => !used.has(n)).sort()).toEqual([]);
    expect(used.size).toBeGreaterThan(20);
  });

  it("la directive v-loading reste installée tant qu'un gabarit l'emploie", () => {
    const usesLoading = Object.values(sources).some((s) =>
      s.includes("v-loading"),
    );
    // installElementPlus installe ElLoading sans condition : ce test documente le besoin.
    expect(usesLoading).toBe(true);
  });
});
