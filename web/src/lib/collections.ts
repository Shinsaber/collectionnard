// Types d'item acceptés et tri par défaut d'une collection (P7-1b). Pendant Go :
// model.Collection.AcceptedItemTypes / SpecializedType et handler/collection.go (checkCollectionSettings).
import type { Collection } from "@/api/types";
import type { Lang } from "@/i18n/langs";
import { ATTR_PREFIX, sortFields } from "./itemSearch";
import { DEFAULT_LOT_SORT } from "./lots";
import { fieldLabel, type Schema } from "./schema";

/** Types d'item acceptés : la liste de la collection dans celle de son type ; vide = tous. */
export function acceptedTypeIds(
  c: Pick<Collection, "item_type_ids" | "collection_type">,
): string[] {
  const limit = c.collection_type?.item_type_ids ?? [];
  const own = c.item_type_ids ?? [];
  if (own.length === 0) return limit;
  if (limit.length === 0) return own;
  const both = own.filter((id) => limit.includes(id));
  return both.length ? both : limit; // liste vidée par un resserrement ultérieur du type
}

/** Seul type d'item d'une collection spécialisée (tris par ses attributs), sinon `undefined`. */
export function specializedTypeId(
  c: Pick<Collection, "item_type_ids" | "collection_type">,
): string | undefined {
  const ids = acceptedTypeIds(c);
  return ids.length === 1 ? ids[0] : undefined;
}

export interface SortOption {
  value: string;
  label: string;
}

/**
 * Tris des lots : communs à tous les items, plus les attributs triables de `schema` (type d'item choisi ou
 * collection spécialisée), ascendant et descendant.
 */
export function lotSortOptions(
  t: (key: string) => string,
  schema: Schema | undefined,
  lang: Lang,
): SortOption[] {
  const out: SortOption[] = [
    { value: DEFAULT_LOT_SORT, label: t("collections.toolbar.sortRecent") },
    { value: "created_at", label: t("collections.toolbar.sortOldest") },
    { value: "-quantity", label: t("collections.toolbar.sortQuantity") },
  ];
  for (const f of sortFields(schema)) {
    const label = fieldLabel(schema, f.key, lang);
    out.push(
      { value: `${ATTR_PREFIX}${f.key}`, label: `${label} ↑` },
      { value: `-${ATTR_PREFIX}${f.key}`, label: `${label} ↓` },
    );
  }
  return out;
}
