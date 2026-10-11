// Export CSV : choix du séparateur selon la langue d'affichage.
import type { CsvDelimiter } from "@/api/types";

/**
 * Séparateur par défaut d'un export CSV : le point-virgule pour le français (un tableur
 * réglé en français prend la virgule comme séparateur décimal), la virgule sinon.
 */
export function defaultDelimiter(lang: string): CsvDelimiter {
  return lang === "fr" || lang.startsWith("fr-") ? "semicolon" : "comma";
}
