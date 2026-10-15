import { onBeforeUnmount, readonly, ref, type Ref } from "vue";

/** Points de rupture de la coquille (alignés sur Tailwind : md 768, lg 1024, xl 1280). */
export const MOBILE_QUERY = "(max-width: 767.98px)";
export const WIDE_QUERY = "(min-width: 1280px)";

/**
 * Suit une media query CSS. Réservé aux cas où le rendu change de composant (panneau ↔
 * feuille, pagination compacte) ; pour une simple mise en page, préférer les classes Tailwind.
 */
export function useMediaQuery(query: string): Readonly<Ref<boolean>> {
  const mql =
    typeof window !== "undefined" && window.matchMedia
      ? window.matchMedia(query)
      : null;
  const matches = ref(mql?.matches ?? false);
  if (mql) {
    const onChange = (e: MediaQueryListEvent) => {
      matches.value = e.matches;
    };
    mql.addEventListener("change", onChange);
    onBeforeUnmount(() => mql.removeEventListener("change", onChange));
  }
  return readonly(matches);
}
