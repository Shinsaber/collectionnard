import { computed, ref } from "vue";

/** Valeur « vide » au sens d'un formulaire : absente, nulle, chaîne vide, liste ou objet vides. */
function isEmpty(value: unknown): boolean {
  if (value == null || value === "") return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value).length === 0;
  return false;
}

/** Forme canonique d'une valeur : clés triées, valeurs vides retirées (récursivement). */
function normalize(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(normalize);
  if (value && typeof value === "object") {
    const out: Record<string, unknown> = {};
    for (const key of Object.keys(value).sort()) {
      const v = normalize((value as Record<string, unknown>)[key]);
      if (!isEmpty(v)) out[key] = v;
    }
    return out;
  }
  return value;
}

/** Empreinte comparable d'un formulaire (un champ saisi puis vidé ne compte pas). */
export function formSnapshot(value: unknown): string {
  return JSON.stringify(normalize(value) ?? null);
}

/**
 * Détection des modifications d'un formulaire : `markClean()` à l'ouverture (ou après
 * enregistrement) mémorise l'état de référence ; `dirty` vaut vrai dès qu'il diffère.
 */
export function useDirty(source: () => unknown) {
  const baseline = ref(formSnapshot(source()));
  const dirty = computed(() => formSnapshot(source()) !== baseline.value);
  function markClean() {
    baseline.value = formSnapshot(source());
  }
  return { dirty, markClean };
}
