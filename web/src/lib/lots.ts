// Lots d'une collection (P7-1) : un item peut y former plusieurs lots d'exemplaires identiques.
// Ajouter un item déjà présent laisse toujours le choix : augmenter un lot existant ou en créer un.
import { collectionApi } from "@/api/client";
import type { CollectionItem, ItemCollectionMembership } from "@/api/types";
import type { Lang } from "@/i18n/langs";
import {
  fieldLabel,
  fieldValue,
  isImageGallery,
  orderedKeys,
  type Schema,
} from "./schema";

/** Cible d'un ajout : un lot existant (son id) ou un nouveau lot. */
export type LotTarget = string | "new";

/** Lots de l'item dans la collection collectionId, parmi ses appartenances. */
export function lotsIn(
  memberships: ItemCollectionMembership[],
  collectionId: string,
): ItemCollectionMembership[] {
  return memberships.filter((m) => m.collection.id === collectionId);
}

/** Nombre de collections distinctes (un item peut y former plusieurs lots). */
export function collectionCount(
  memberships: ItemCollectionMembership[],
): number {
  return new Set(memberships.map((m) => m.collection.id)).size;
}

/** Résumé des attributs renseignés d'un lot (« État : Bon · Prix d'achat : 2 »), galeries exclues. */
export function lotSummary(
  schema: Schema | undefined,
  attributes: Record<string, unknown> | undefined,
  lang: Lang,
): string {
  const attrs = attributes ?? {};
  return orderedKeys(schema, attrs)
    .filter((k) => !isImageGallery(schema, k) && k in attrs)
    .map((k) => ({ k, v: fieldValue(schema, k, attrs[k], lang) }))
    .filter(({ v }) => v !== "")
    .map(({ k, v }) => `${fieldLabel(schema, k, lang)} : ${v}`)
    .join(" · ");
}

/**
 * Ajoute quantity exemplaires de itemId à la collection : au lot target (ses
 * attributs inchangés), ou dans un nouveau lot portant attributes.
 */
export function addCopies(
  collectionId: string,
  itemId: string,
  target: LotTarget,
  quantity: number,
  attributes: Record<string, unknown>,
  lots: ItemCollectionMembership[],
): Promise<CollectionItem> {
  const lot = lots.find((l) => l.lot_id === target);
  if (target === "new" || !lot) {
    return collectionApi.addItem(collectionId, {
      item_id: itemId,
      attributes,
      quantity,
    });
  }
  return collectionApi.updateLot(collectionId, lot.lot_id, {
    attributes: lot.attributes,
    quantity: lot.quantity + quantity,
  });
}

/** Actions sur un lot dans la liste d'une collection. */
/** Actions d'un lot ; `lend` et `loans` (P7-3) pour une collection dont le type ouvre les prêts. */
export type LotCommand =
  "edit" | "split" | "merge" | "lend" | "loans" | "remove";

/** Tri par défaut des lots : ajout le plus récent (défaut de l'API sans recherche). */
export const DEFAULT_LOT_SORT = "-created_at";

/** Recherche des lots d'une collection. */
export interface LotQuery {
  search: string;
  item_type_id: string;
  sort: string;
  multiple: boolean;
}

export function emptyLotQuery(): LotQuery {
  return {
    search: "",
    item_type_id: "",
    sort: DEFAULT_LOT_SORT,
    multiple: false,
  };
}

/** Paramètres de GET /collections/{id}/items : seuls les critères renseignés sont envoyés. */
export function lotQueryParams(q: LotQuery): Record<string, string> {
  const out: Record<string, string> = {};
  if (q.search.trim()) out.search = q.search.trim();
  if (q.item_type_id) out.item_type_id = q.item_type_id;
  // Toujours envoyé : sans `sort`, l'API applique le tri par défaut de la collection (P7-1b).
  if (q.sort) out.sort = q.sort;
  if (q.multiple) out.multiple = "true";
  return out;
}
