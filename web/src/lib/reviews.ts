import { orderedKeys, type Schema } from "@/lib/schema";
import type { ReviewVisibility, Visibility } from "@/api/types";

/** Réactions à un avis (I-1) : une intention ou une émotion, jamais un signalement. */
export const REACTION_KINDS = [
  "interesting",
  "tempting",
  "funny",
  "disapprove",
] as const;
export type ReactionKind = (typeof REACTION_KINDS)[number];

/** Motifs d'un signalement. */
export const REPORT_REASONS = [
  "abusive",
  "harassment",
  "spoiler",
  "spam",
  "off_topic",
  "other",
] as const;
export type ReportReason = (typeof REPORT_REASONS)[number];

/** Ordre du formulaire : du plus discret au plus exposé. */
export const REVIEW_VISIBILITIES: ReviewVisibility[] = [
  "private",
  "friends",
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

export type StarColorMode = "steps" | "gradient";

/** Couleur des étoiles : « steps » par paliers, « gradient » en dégradé continu. */
export const STAR_COLOR_MODE: StarColorMode = "steps";

/** Palier d'une note, arrondie comme les étoiles : 0–4 bas, 5–7 moyen, 8–10 haut. */
export function ratingTone(rating: number): "low" | "mid" | "high" {
  const r = Math.round(rating);
  if (r <= 4) return "low";
  return r <= 7 ? "mid" : "high";
}

/**
 * Couleur CSS des étoiles, à partir des jetons du thème (clair et sombre). En dégradé, la couleur
 * passe du rouge (3 et moins) à l'orange (6) puis au vert (9 et plus), mélangée en OKLCH.
 */
export function ratingColor(
  rating: number,
  mode: StarColorMode = STAR_COLOR_MODE,
): string {
  if (mode === "steps") return `var(--c-star-${ratingTone(rating)})`;
  const r = Math.min(9, Math.max(3, rating));
  const [from, to, start] =
    r <= 6 ? ["low", "mid", 3] : (["mid", "high", 6] as const);
  const part = Math.round(((r - Number(start)) / 3) * 100);
  return `color-mix(in oklch, var(--c-star-${to}) ${part}%, var(--c-star-${from}))`;
}
