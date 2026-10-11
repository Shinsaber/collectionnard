// Instance vue-i18n (Composition API) : catalogues de l'interface et locales Element Plus.
// Dans un composant : `const { t } = useI18n()` ; ailleurs (composables, stores) : `i18n.global.t`.
// Ajouter une langue : `langs.ts`, puis une entrée dans `messages` et `elementLocales` ci-dessous.
import { createI18n } from "vue-i18n";
import type { Language } from "element-plus/es/locale";
import elEn from "element-plus/es/locale/lang/en";
import elFr from "element-plus/es/locale/lang/fr";
import en from "./locales/en";
import fr, { type Messages } from "./locales/fr";
import { DEFAULT_LANG, type Lang } from "./langs";

export * from "./langs";
export type { Messages };

const messages: Record<Lang, Messages> = { fr, en };

/** Libellés internes d'Element Plus (pagination, sélecteurs, boîtes de dialogue…). */
export const elementLocales: Record<Lang, Language> = { fr: elFr, en: elEn };

export const i18n = createI18n<[Messages], Lang, false>({
  legacy: false,
  locale: DEFAULT_LANG,
  fallbackLocale: DEFAULT_LANG,
  messages,
  // Une clé manquante s'affiche telle quelle (et avertit en dev) : pas de bruit en test.
  missingWarn: import.meta.env.DEV,
  fallbackWarn: false,
});
