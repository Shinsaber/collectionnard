// Registre des langues de l'application : seule liste à compléter pour en ajouter une
// (puis créer `locales/<code>/`, et brancher catalogue + locale Element Plus dans `index.ts`
// — le type `Record<Lang, …>` y signale tout oubli à la compilation).
// Volontairement sans dépendance : importé par `lib/schema.ts` (pur, testé seul).

export const LANGS = ["fr", "en"] as const;

export type Lang = (typeof LANGS)[number];

/** Langue de repli : interface par défaut et première valeur essayée dans un objet i18n. */
export const DEFAULT_LANG: Lang = "fr";

/** Nom de chaque langue dans sa propre langue (sélecteur de l'en-tête). */
export const LANG_NAMES: Record<Lang, string> = {
  fr: "Français",
  en: "English",
};

export function isLang(value: unknown): value is Lang {
  return (LANGS as readonly unknown[]).includes(value);
}

/**
 * Première langue supportée parmi des préférences BCP 47 (`navigator.languages`) :
 * `en-GB` → `en`. Renvoie `undefined` si aucune ne correspond.
 */
export function matchLang(prefs: readonly string[]): Lang | undefined {
  for (const p of prefs) {
    const base = p.toLowerCase().split("-")[0];
    if (isLang(base)) return base;
  }
  return undefined;
}
