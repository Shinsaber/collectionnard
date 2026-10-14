import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { collectionTypeApi, itemTypeApi, relationTypeApi } from "@/api/client";
import type {
  CollectionType,
  ItemType,
  PaginatedResponse,
  PaginationParams,
  RelationType,
} from "@/api/types";

type ListFn<T> = (params?: PaginationParams) => Promise<PaginatedResponse<T>>;

/** Charge toutes les pages d'un listing (per_page max côté serveur : 100). */
async function fetchAll<T>(list: ListFn<T>): Promise<T[]> {
  const out: T[] = [];
  let page = 1;
  for (;;) {
    const res = await list({ page, per_page: 100 });
    out.push(...res.data);
    if (page >= (res.total_pages ?? 1) || res.data.length === 0) break;
    page++;
  }
  return out;
}

/**
 * Cache des méta-types (ItemTypes, RelationTypes, CollectionTypes) : ils changent
 * rarement et sont lus par de nombreuses vues. Les vues d'administration appellent
 * `invalidate()` après chaque écriture.
 */
export const useMetaTypes = defineStore("metaTypes", () => {
  const itemTypes = ref<ItemType[]>([]);
  const relationTypes = ref<RelationType[]>([]);
  const collectionTypes = ref<CollectionType[]>([]);

  const pending: Partial<
    Record<"items" | "relations" | "collections", Promise<void>>
  > = {};
  const loaded = { items: false, relations: false, collections: false };

  function load<T>(
    key: keyof typeof loaded,
    list: ListFn<T>,
    target: { value: T[] },
    force: boolean,
  ): Promise<void> {
    if (loaded[key] && !force) return Promise.resolve();
    if (!pending[key]) {
      pending[key] = fetchAll(list)
        .then((data) => {
          target.value = data;
          loaded[key] = true;
        })
        .finally(() => {
          delete pending[key];
        });
    }
    return pending[key]!;
  }

  const loadItemTypes = (force = false) =>
    load("items", itemTypeApi.list, itemTypes, force);
  const loadRelationTypes = (force = false) =>
    load("relations", relationTypeApi.list, relationTypes, force);
  const loadCollectionTypes = (force = false) =>
    load("collections", collectionTypeApi.list, collectionTypes, force);

  /** Schémas d'ItemType indexés par nom (les nœuds du graphe ne portent que le nom). */
  const itemTypeSchemasByName = computed<
    Record<string, Record<string, unknown>>
  >(() =>
    Object.fromEntries(
      itemTypes.value.map((t) => [
        t.name,
        t.attributes_schema as Record<string, unknown>,
      ]),
    ),
  );
  const relationTypeSchemasByName = computed<
    Record<string, Record<string, unknown>>
  >(() =>
    Object.fromEntries(
      relationTypes.value.map((t) => [
        t.name,
        t.attributes_schema as Record<string, unknown>,
      ]),
    ),
  );

  /** À appeler après toute création / modification / suppression d'un méta-type. */
  function invalidate() {
    loaded.items = loaded.relations = loaded.collections = false;
  }

  return {
    itemTypes,
    relationTypes,
    collectionTypes,
    itemTypeSchemasByName,
    relationTypeSchemasByName,
    loadItemTypes,
    loadRelationTypes,
    loadCollectionTypes,
    invalidate,
  };
});
