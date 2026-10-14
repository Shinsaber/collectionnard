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
