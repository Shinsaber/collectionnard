import { describe, expect, it } from "vitest";
import { LANGS, matchLang } from "./langs";
import en from "./locales/en";
import fr from "./locales/fr";

/** Liste « a.b.c » de toutes les clés feuilles d'un catalogue. */
function leaves(obj: object, prefix = ""): string[] {
  return Object.entries(obj).flatMap(([k, v]) =>
    v && typeof v === "object"
      ? leaves(v, `${prefix}${k}.`)
      : [`${prefix}${k}`],
  );
}

describe("catalogues i18n", () => {
  it("toutes les langues ont exactement les clés du français, sans texte vide", () => {
    const catalogs: Record<string, object> = { fr, en };
    expect(Object.keys(catalogs).sort()).toEqual([...LANGS].sort());
    const ref = leaves(fr).sort();
    for (const [lang, cat] of Object.entries(catalogs)) {
      expect(leaves(cat).sort(), lang).toEqual(ref);
      for (const key of ref) {
        const v = key
          .split(".")
          .reduce<unknown>((o, k) => (o as Record<string, unknown>)[k], cat);
        expect(v, `${lang}:${key}`).not.toBe("");
      }
    }
  });

  it("matchLang retient la première langue supportée", () => {
    expect(matchLang(["de-DE", "en-GB", "fr"])).toBe("en");
    expect(matchLang(["FR-ca"])).toBe("fr");
    expect(matchLang(["de"])).toBeUndefined();
  });
});

// Audit Q-007 : aucune clé orpheline. Une feuille du catalogue est « employée » si le code la
// cite littéralement (« ns.a.b »), ou par un préfixe dynamique (`ns.a.${x}`, "ns.a." + x), ou
// par un suffixe dynamique (`${ns}.title`).
const sources = import.meta.glob(
  [
    "/src/**/*.vue",
    "/src/**/*.ts",
    "!/src/**/*.test.ts",
    "!/src/i18n/locales/**",
  ],
  { query: "?raw", import: "default", eager: true },
) as Record<string, string>;

describe("clés i18n employées", () => {
  it("chaque clé du catalogue français est citée par le code", () => {
    const code = Object.values(sources).join("\n");
    const prefixes = [
      ...code.matchAll(/`([a-zA-Z]\w*(?:\.\w+)*\.)\$\{/g),
      ...code.matchAll(/"([a-zA-Z]\w*(?:\.\w+)*\.)"\s*\+/g),
    ].map((m) => m[1]!);
    const suffixes = [...code.matchAll(/\$\{[^}]+\}((?:\.\w+)+)`/g)].map(
      (m) => m[1]!,
    );
    const orphans = leaves(fr).filter(
      (key) =>
        !code.includes(key) &&
        !prefixes.some((p) => key.startsWith(p)) &&
        !suffixes.some((s) => key.endsWith(s)),
    );
    expect(orphans).toEqual([]);
  });
});
