// Cadeaux réservés (P7-4) : aides de présentation, sans appel réseau.
import { daysFrom } from "./loans";

/** Durées d'une réservation (bornes du serveur) : 60 jours par défaut, un an au plus. */
export const GIFT_DEFAULT_DAYS = 60;
export const GIFT_MAX_DAYS = 365;

/** Échéance proposée à une nouvelle réservation. */
export const defaultExpiry = (now = new Date()) =>
  daysFrom(GIFT_DEFAULT_DAYS, now);
/** Échéance la plus lointaine acceptée. */
export const maxExpiry = (now = new Date()) => daysFrom(GIFT_MAX_DAYS, now);
import type { CollectionItem, GiftReservation } from "@/api/types";

/** Exemplaires du lot déjà réservés (offerts compris). */
export function reservedCopies(
  lot: Pick<CollectionItem, "reservations">,
): number {
  return (lot.reservations ?? []).reduce((n, r) => n + r.quantity, 0);
}

/** Ma réservation sur ce lot, s'il y en a une. */
export function myReservation(
  lot: Pick<CollectionItem, "reservations">,
): GiftReservation | undefined {
  return (lot.reservations ?? []).find((r) => r.mine);
}

/** Exemplaires encore à réserver ; avec `except`, ceux qu'il pourrait porter. */
export function reservableCopies(
  lot: Pick<CollectionItem, "quantity" | "reservations">,
  except?: GiftReservation,
): number {
  return Math.max(
    0,
    lot.quantity - reservedCopies(lot) + (except?.quantity ?? 0),
  );
}

/** Nom d'une personne tel qu'elle se présente. */
export function personName(p?: {
  username: string;
  display_name?: string;
}): string {
  return p?.display_name || p?.username || "";
}
