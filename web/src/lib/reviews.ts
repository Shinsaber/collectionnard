import { orderedKeys, type Schema } from "@/lib/schema";
import type { ReviewVisibility, Visibility } from "@/api/types";

/** Ordre du formulaire : du plus discret au plus exposé. */
export const REVIEW_VISIBILITIES: ReviewVisibility[] = [
  "private",
  "members",
  "anonymous",
  "public",
];

/** Visibilité d'un nouvel avis : publique si le profil l'est, sinon anonyme (phase 6, décision). */
export function defaultReviewVisibility(
  profile: Visibility | undefined,
): ReviewVisibility {
  return profile === "public" ? "public" : "anonymous";
}

/** Clés des critères d'un type d'avis, dans leur ordre d'affichage. */
export function criteriaKeys(schema: unknown): string[] {
  return orderedKeys(schema as Schema | undefined);
}

/** Libellé du choix de l'édition dans la langue courante, repli sur le français puis sur `fallback`. */
export function contextLabel(
  label: Record<string, string> | undefined,
  lang: string,
  fallback: string,
): string {
  return label?.[lang] || label?.fr || fallback;
}

/** Moyenne à une décimale, comme le serveur (`internal/review`). */
export function criteriaAverage(
  values: Record<string, unknown>,
  keys: string[],
): number {
  if (keys.length === 0) return 0;
  const sum = keys.reduce((total, k) => total + Number(values[k] ?? 0), 0);
  return Math.round((sum / keys.length) * 10) / 10;
}

/** Part remplie des 5 étoiles, en % : la demi-étoile vaut 1 point sur 10. */
export function starsFill(rating: number): number {
  return Math.round(Math.min(10, Math.max(0, rating))) * 10;
}

/** Couleur des étoiles selon la note. */
export function ratingTone(rating: number): "danger" | "accent" | "primary" {
  if (rating < 4) return "danger";
  return rating < 7 ? "accent" : "primary";
}
