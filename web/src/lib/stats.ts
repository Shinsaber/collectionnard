// Mise en forme des statistiques (P4-1) : libellés des graphiques déduits du schéma
// du CollectionType (x-labels, x-enum-labels), jamais d'une clé codée en dur.
import type { Lang } from "@/i18n/langs";
import type { Distribution, MonthCount } from "@/api/types";
import { enumLabel, type Schema } from "./schema";

export interface Series {
  labels: string[];
  values: number[];
}

/** « 2026-10 » → « oct. 2026 » (fr) / « Oct 2026 » (en). */
export function monthLabel(month: string, lang: Lang): string {
  const [y, m] = month.split("-").map(Number);
  if (!y || !m) return month;
  return new Date(Date.UTC(y, m - 1, 1)).toLocaleDateString(lang, {
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  });
}

export function monthSeries(months: MonthCount[], lang: Lang): Series {
  return {
    labels: months.map((m) => monthLabel(m.month, lang)),
    values: months.map((m) => m.count),
  };
}

function isBoolean(schema: Schema | null | undefined, key: string): boolean {
  const type = (schema?.properties as Record<string, Schema> | undefined)?.[key]
    ?.type;
  return Array.isArray(type) ? type.includes("boolean") : type === "boolean";
}

/**
 * Barres d'une répartition : libellés d'enum traduits (`x-enum-labels`), booléens en
 * oui / non, puis une barre « non renseigné » s'il manque des valeurs.
 */
export function distributionSeries(
  dist: Distribution,
  schema: Schema | null | undefined,
  lang: Lang,
  words: { yes: string; no: string; missing: string },
): Series {
  const bool = isBoolean(schema, dist.field);
  const labels = dist.values.map((v) => {
    if (bool && v.value === "true") return words.yes;
    if (bool && v.value === "false") return words.no;
    return enumLabel(schema, dist.field, v.value, lang);
  });
  const values = dist.values.map((v) => v.count);
  if (dist.missing > 0) {
    labels.push(words.missing);
    values.push(dist.missing);
  }
  return { labels, values };
}

/** Part possédée, en pourcentage entier (0 si total nul). */
export function percent(owned: number, total: number): number {
  return total > 0 ? Math.round((owned / total) * 100) : 0;
}

/** Nombre mis en forme selon la langue (deux décimales au plus). */
export function formatNumber(n: number, lang: Lang): string {
  return n.toLocaleString(lang, { maximumFractionDigits: 2 });
}
