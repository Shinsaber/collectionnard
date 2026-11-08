// Prêts (P7-3) : aides de présentation, sans appel réseau.
import type { CollectionItem, Loan } from "@/api/types";

/** Exemplaires du lot actuellement prêtés. */
export function lentCopies(lot: Pick<CollectionItem, "loans">): number {
  return (lot.loans ?? []).reduce((n, l) => n + l.quantity, 0);
}

/** Exemplaires du lot encore disponibles au prêt. */
export function availableCopies(
  lot: Pick<CollectionItem, "quantity" | "loans">,
): number {
  return Math.max(0, lot.quantity - lentCopies(lot));
}

/** Nom affiché de l'emprunteur : l'ami tel qu'il se présente, sinon le nom figé. */
export function borrowerName(loan: Loan): string {
  return (
    loan.borrower?.display_name || loan.borrower?.username || loan.borrower_name
  );
}

/** Date du jour au format de l'API (`AAAA-MM-JJ`, heure locale). */
export function today(now = new Date()): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${now.getFullYear()}-${p(now.getMonth() + 1)}-${p(now.getDate())}`;
}

/** Date `AAAA-MM-JJ` lisible dans la langue de l'interface, sans décalage de fuseau. */
export function formatDay(day: string, lang: string): string {
  const [y, m, d] = day.split("-").map(Number);
  return new Date(y!, (m ?? 1) - 1, d ?? 1).toLocaleDateString(lang);
}
