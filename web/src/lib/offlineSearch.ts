// Recherche dans la copie hors ligne (lot E5c) : une ligne par item de mes collections, avec
// son libellé, son type et ses entrées (collection + attributs contextuels) ; filtre texte sans
// accents ni casse, qui reconnaît un ISBN saisi avec ou sans tirets.
import type { Lang } from "@/i18n/langs";
import { itemLabel, typeTitle } from "@/lib/schema";
import type {
  Collection,
  CollectionType,
  Item,
  OfflineEntry,
  OfflineSnapshot,
} from "@/api/types";

export interface OfflineRow {
  /** Item avec son `item_type` rattaché (libellé, schéma d'affichage). */
  item: Item;
  label: string;
  type: string;
  entries: {
    collection: Collection;
    collectionType?: CollectionType;
    entry: OfflineEntry;
  }[];
  /** Texte de recherche normalisé (libellé, type, valeurs des attributs). */
  haystack: string;
}

/** Minuscules sans accents ; tirets et espaces entre chiffres retirés (ISBN « 978-2-… »). */
export function normalize(text: string): string {
  return text
    .normalize("NFD")
    .replace(/\p{M}/gu, "")
    .toLowerCase()
    .replace(/(\d)[\s-]+(?=\d)/g, "$1");
}

/** Chaînes et nombres d'une valeur d'attribut, à toute profondeur (objets i18n compris), hors URLs. */
function leaves(value: unknown, out: string[]): string[] {
  if (typeof value === "string") {
    if (!/^(\/|https?:\/\/)/.test(value)) out.push(value);
  } else if (typeof value === "number") {
    out.push(String(value));
  } else if (value && typeof value === "object") {
    for (const v of Object.values(value)) leaves(v, out);
  }
  return out;
}

export function buildRows(snap: OfflineSnapshot, lang: Lang): OfflineRow[] {
  const itemTypes = new Map(snap.item_types.map((t) => [t.id, t]));
  const colTypes = new Map(snap.collection_types.map((t) => [t.id, t]));
  const collections = new Map(snap.collections.map((c) => [c.id, c]));
  const entriesByItem = new Map<string, OfflineRow["entries"]>();
  for (const entry of snap.entries) {
    const collection = collections.get(entry.collection_id);
    if (!collection) continue;
    const list = entriesByItem.get(entry.item_id) ?? [];
    list.push({
      collection,
      collectionType: colTypes.get(collection.collection_type_id),
      entry,
    });
    entriesByItem.set(entry.item_id, list);
  }
  const rows = snap.items.map((raw): OfflineRow => {
    const item = { ...raw, item_type: itemTypes.get(raw.item_type_id) };
    const label = itemLabel(item, lang);
    const type = typeTitle(item.item_type, lang) || item.item_type?.name || "";
    return {
      item,
      label,
      type,
      entries: entriesByItem.get(raw.id) ?? [],
      haystack: normalize(
        [label, type, ...leaves(raw.attributes, [])].join(" \u0001 "),
      ),
    };
  });
  return rows.sort((a, b) => a.label.localeCompare(b.label, lang));
}

/** Lignes dont le texte contient chaque mot de la requête, éventuellement d'une seule collection. */
export function searchRows(
  rows: OfflineRow[],
  query: string,
  collectionId?: string | null,
): OfflineRow[] {
  const words = normalize(query).split(/\s+/).filter(Boolean);
  return rows.filter(
    (r) =>
      (!collectionId ||
        r.entries.some((e) => e.collection.id === collectionId)) &&
      words.every((w) => r.haystack.includes(w)),
  );
}
