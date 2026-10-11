import { computed, ref, watch } from "vue";
import {
  DEFAULT_LANG,
  LANGS,
  LANG_NAMES,
  elementLocales,
  i18n,
  isLang,
  matchLang,
  type Lang,
} from "@/i18n";

const STORAGE_KEY = "colec-lang";

/** Langue mémorisée, sinon celle du navigateur si supportée, sinon la langue par défaut. */
function getInitialLang(): Lang {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (isLang(stored)) return stored;
  } catch {
    /* stockage indisponible */
  }
  if (typeof navigator !== "undefined") {
    const prefs = navigator.languages?.length
      ? navigator.languages
      : [navigator.language];
    const found = matchLang(prefs.filter(Boolean));
    if (found) return found;
  }
  return DEFAULT_LANG;
}

// Langue de l'application (interface vue-i18n, Element Plus, x-labels, objets i18n) —
// partagée par toute l'app.
const lang = ref<Lang>(getInitialLang());

function apply(l: Lang) {
  i18n.global.locale.value = l;
  if (typeof document !== "undefined") document.documentElement.lang = l;
}
apply(lang.value);

watch(lang, (l) => {
  apply(l);
  try {
    localStorage.setItem(STORAGE_KEY, l);
  } catch {
    /* ignoré */
  }
});

const elementLocale = computed(() => elementLocales[lang.value]);

export function useLang() {
  function setLang(l: Lang) {
    lang.value = l;
  }
  /** Passe à la langue suivante du registre (bascule fr ↔ en tant qu'il n'y en a que deux). */
  function toggle() {
    const i = LANGS.indexOf(lang.value);
    lang.value = LANGS[(i + 1) % LANGS.length]!;
  }
  return {
    lang,
    setLang,
    toggle,
    langs: LANGS,
    langNames: LANG_NAMES,
    elementLocale,
  };
}
