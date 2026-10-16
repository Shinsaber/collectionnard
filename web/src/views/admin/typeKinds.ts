import { collectionTypeApi, itemTypeApi, relationTypeApi } from "@/api/client";
import type { CollectionType, ItemType, RelationType } from "@/api/types";

/** Les trois sortes de méta-types, éditées par la même page (TypeEditorView). */
export type TypeKind = "item" | "relation" | "collection";

export type AnyType = ItemType | RelationType | CollectionType;

/** Données saisies sur la page ; les types source / cible ne concernent que les relations. */
export interface TypeForm {
  name: string;
  description: string;
  source_item_type_ids: string[];
  target_item_type_ids: string[];
  attributes_schema: Record<string, unknown>;
}

interface TypeKindConfig {
  /** Chemin de la liste (le nouveau type est `${listPath}/new`, un type `${listPath}/:id`). */
  listPath: string;
  listTitleKey: string;
  namePlaceholderKey: string;
  createdKey: string;
  updatedKey: string;
  get: (id: string) => Promise<AnyType>;
  save: (form: TypeForm, id?: string) => Promise<AnyType>;
}

function base(f: TypeForm) {
  return {
    name: f.name,
    description: f.description,
    attributes_schema: f.attributes_schema,
  };
}

export const TYPE_KINDS: Record<TypeKind, TypeKindConfig> = {
  item: {
    listPath: "/admin/item-types",
    listTitleKey: "admin.itemTypes.title",
    namePlaceholderKey: "admin.itemTypes.namePlaceholder",
    createdKey: "admin.shared.created",
    updatedKey: "admin.shared.updated",
    get: (id) => itemTypeApi.get(id),
    save: (f, id) =>
      id ? itemTypeApi.update(id, base(f)) : itemTypeApi.create(base(f)),
  },
  relation: {
    listPath: "/admin/relation-types",
    listTitleKey: "admin.relationTypes.title",
    namePlaceholderKey: "admin.relationTypes.namePlaceholder",
    createdKey: "admin.relationTypes.created",
    updatedKey: "admin.relationTypes.updated",
    get: (id) => relationTypeApi.get(id),
    save: (f, id) => {
      const payload = {
        ...base(f),
        source_item_type_ids: f.source_item_type_ids,
        target_item_type_ids: f.target_item_type_ids,
      };
      return id
        ? relationTypeApi.update(id, payload)
        : relationTypeApi.create(payload);
    },
  },
  collection: {
    listPath: "/admin/collection-types",
    listTitleKey: "admin.collectionTypes.title",
    namePlaceholderKey: "admin.collectionTypes.namePlaceholder",
    createdKey: "admin.shared.created",
    updatedKey: "admin.shared.updated",
    get: (id) => collectionTypeApi.get(id),
    save: (f, id) =>
      id
        ? collectionTypeApi.update(id, base(f))
        : collectionTypeApi.create(base(f)),
  },
};

export function emptySchema(): Record<string, unknown> {
  return { type: "object", properties: {}, additionalProperties: false };
}

/** Formulaire initial d'un type existant, ou d'un nouveau type. */
export function formOf(type?: AnyType | null): TypeForm {
  const rel = type as RelationType | undefined;
  return {
    name: type?.name ?? "",
    description: type?.description ?? "",
    source_item_type_ids: rel?.source_item_type_ids ?? [],
    target_item_type_ids: rel?.target_item_type_ids ?? [],
    attributes_schema: type?.attributes_schema ?? emptySchema(),
  };
}
