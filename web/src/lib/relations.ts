// Relations d'un item telles que les sert `GET /items/{id}/relations` (audit Q-001) : les deux
// extrémités sont incluses avec leurs attributs, mais ni le RelationType ni les ItemTypes (leurs
// schémas se répétaient à chaque ligne). Le front les retrouve dans le cache des méta-types.
import type { ItemRelation, ItemType, RelationType } from "@/api/types";

/** Index par identifiant. */
export function byId<T extends { id: string }>(list: T[]): Record<string, T> {
  return Object.fromEntries(list.map((t) => [t.id, t]));
}

/**
 * Complète une relation avec son type et les types de ses extrémités quand l'API ne les a
 * pas joints ; ce qui est déjà présent est conservé (réponse de `GET /relations/{id}`).
 */
export function withTypes(
  rel: ItemRelation,
  itemTypes: Record<string, ItemType>,
  relationTypes: Record<string, RelationType>,
): ItemRelation {
  const end = (item: ItemRelation["source_item"]) =>
    item && !item.item_type && itemTypes[item.item_type_id]
      ? { ...item, item_type: itemTypes[item.item_type_id] }
      : item;
  return {
    ...rel,
    relation_type: rel.relation_type ?? relationTypes[rel.relation_type_id],
    source_item: end(rel.source_item),
    target_item: end(rel.target_item),
  };
}
