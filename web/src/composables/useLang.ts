import { ref, watch } from "vue";
import type { Lang } from "@/lib/schema";

const STORAGE_KEY = "colec-lang";

function getInitialLang(): Lang {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "fr" || stored === "en") return stored;
  } catch {
    /* stockage indisponible */
  }
  return "fr";
}

// Langue d'affichage des libellés (x-labels, objets i18n) — partagée par toute l'app.
const lang = ref<Lang>(getInitialLang());

watch(lang, (l) => {
  try {
    localStorage.setItem(STORAGE_KEY, l);
  } catch {
    /* ignoré */
  }
});

export function useLang() {
  function toggle() {
    lang.value = lang.value === "fr" ? "en" : "fr";
  }
  return { lang, toggle };
}
