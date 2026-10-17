// Types et constantes partagés par SchemaBuilder.vue (coquille) et SchemaFieldCard.vue (un champ).
import { parseImageCategory, type ImageCategory } from "@/lib/schema";

/** Options d'affichage du builder : chaque drapeau montre ou cache une contrainte. */
export interface SchemaBuilderFeatures {
  labels: boolean; // x-labels / x-enum-labels, une saisie par langue (LANGS)
  description: boolean;
  default: boolean;
  // string
  minLength: boolean;
  maxLength: boolean;
  pattern: boolean;
  format: boolean;
  enum: boolean;
  // number / integer
  minimum: boolean;
  maximum: boolean;
  exclusiveMinimum: boolean;
  exclusiveMaximum: boolean;
  multipleOf: boolean;
  // array
  array: boolean;
  minItems: boolean;
  maxItems: boolean;
  uniqueItems: boolean;
  // widgets
  i18nText: boolean;
  imageGallery: boolean;
  links: boolean;
}

export const defaultFeatures: SchemaBuilderFeatures = {
  labels: true,
  description: true,
  default: false,
  minLength: true,
  maxLength: true,
  pattern: true,
  format: true,
  enum: true,
  minimum: true,
  maximum: true,
  exclusiveMinimum: false,
  exclusiveMaximum: false,
  multipleOf: false,
  array: false,
  minItems: true,
  maxItems: true,
  uniqueItems: true,
  i18nText: true,
  imageGallery: true,
  links: true,
};

/** Un champ du schéma tel qu'édité dans le builder. */
export interface SchemaField {
  key: string;
  type: string;
  /** Libellé par code de langue (`x-labels`) ; les langues hors `LANGS` sont conservées. */
  labels: Record<string, string>;
  required: boolean;
  description?: string | null;
  defaultValue?: unknown;
  // string
  minLength?: number | null;
  maxLength?: number | null;
  pattern?: string | null;
  format?: string | null;
  enum?: string[] | null;
  /** Libellés des valeurs d'enum par langue (`x-enum-labels`) : `{fr: {tv: "Télévision"}}`. */
  enumLabels?: Record<string, Record<string, string>> | null;
  // number / integer
  minimum?: number | null;
  maximum?: number | null;
  exclusiveMinimum?: number | null;
  exclusiveMaximum?: number | null;
  multipleOf?: number | null;
  // array
  itemsType?: string | null;
  minItems?: number | null;
  maxItems?: number | null;
  uniqueItems?: boolean | null;
  // image-gallery
  imageCategories?: ImageCategory[] | null;
  /** Mots-clés de `items` hors `type` / `enum` (tableau d'objets…), restitués tels quels. */
  itemsExtra?: Record<string, unknown> | null;
  /** Mots-clés non gérés par le builder (futurs x-*, default sur un objet…), restitués tels quels. */
  extra?: Record<string, unknown>;
}

export function emptyField(): SchemaField {
  return {
    key: "",
    type: "string",
    labels: {},
    required: false,
    description: null,
    defaultValue: null,
    minLength: null,
    maxLength: null,
    pattern: null,
    minimum: null,
    maximum: null,
    exclusiveMinimum: null,
    exclusiveMaximum: null,
    multipleOf: null,
    format: null,
    enum: null,
    enumLabels: null,
    itemsType: null,
    minItems: null,
    maxItems: null,
    uniqueItems: null,
    imageCategories: null,
    extra: {},
  };
}

/** Option de sélecteur ; `labelKey` est une clé vue-i18n (traduite à l'affichage). */
export interface Option<T = string> {
  labelKey: string;
  value: T;
}

export const baseTypeOptions: Option[] = [
  { labelKey: "schema.types.string", value: "string" },
  { labelKey: "schema.types.integer", value: "integer" },
  { labelKey: "schema.types.number", value: "number" },
  { labelKey: "schema.types.boolean", value: "boolean" },
];
export const arrayTypeOption: Option = {
  labelKey: "schema.types.array",
  value: "array",
};
export const i18nTypeOption: Option = {
  labelKey: "schema.types.i18nText",
  value: "i18n-text",
};
export const imageGalleryTypeOption: Option = {
  labelKey: "schema.types.imageGallery",
  value: "image-gallery",
};
export const linksTypeOption: Option = {
  labelKey: "schema.types.links",
  value: "links",
};

export const formatOptions: Option[] = [
  { labelKey: "schema.formats.uri", value: "uri" },
  { labelKey: "schema.formats.date", value: "date" },
  { labelKey: "schema.formats.dateTime", value: "date-time" },
  { labelKey: "schema.formats.email", value: "email" },
];

export const boolDefaultOptions: Option<boolean>[] = [
  { labelKey: "common.yes", value: true },
  { labelKey: "common.no", value: false },
];

export const arrayItemTypeOptions: Option[] = [
  { labelKey: "schema.types.string", value: "string" },
  { labelKey: "schema.types.integer", value: "integer" },
  { labelKey: "schema.types.number", value: "number" },
  { labelKey: "schema.types.boolean", value: "boolean" },
];

/** Mots-clés d'une propriété pilotés par le builder ; tout le reste va dans `extra`. */
export const MANAGED_PROPERTY_KEYS = new Set([
  "type",
  "description",
  "default",
  "minLength",
  "maxLength",
  "pattern",
  "format",
  "enum",
  "minimum",
  "maximum",
  "exclusiveMinimum",
  "exclusiveMaximum",
  "multipleOf",
  "items",
  "minItems",
  "maxItems",
  "uniqueItems",
  "properties",
  "additionalProperties",
  "propertyNames",
  "x-widget",
  "x-image-categories",
  "x-enum-labels",
]);

/** Mots-clés racine pilotés par le builder ; le reste est conservé à l'identique. */
export const MANAGED_ROOT_KEYS = new Set([
  "type",
  "properties",
  "required",
  "additionalProperties",
  "x-order",
  "x-labels",
  "x-label",
  "x-title",
  "x-description",
]);

/** Parse « neuf, bon, mauvais » en tableau d'enum (null si vide). */
export function parseEnum(value: string): string[] | null {
  const items = value
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  return items.length > 0 ? items : null;
}

/** `x-image-categories` : format objet `{poster: {ratio, tolerance…}}` ou tableau historique de noms. */
export function parseImageCategories(raw: unknown): ImageCategory[] | null {
  if (!raw) return null;
  if (typeof raw === "object" && !Array.isArray(raw)) {
    return Object.entries(raw as Record<string, unknown>).map(([name, v]) =>
      parseImageCategory(name, v),
    );
  }
  if (Array.isArray(raw)) {
    return (raw as string[]).map((name) => ({ name, ratio: "1/1" }));
  }
  return null;
}

/** Code de langue accepté comme clé d'un objet `i18n-text` (`fr`, `en`, `pt-BR`…). */
export const LANG_CODE_PATTERN = "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$";

/** Objet `{clé: chaîne}` épuré (valeurs non textuelles ou vides retirées). */
export function stringMap(raw: unknown): Record<string, string> {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return {};
  return Object.fromEntries(
    Object.entries(raw as Record<string, unknown>).filter(
      (e): e is [string, string] => typeof e[1] === "string" && e[1] !== "",
    ),
  );
}

/** Colonne d'une table par langue : `{fr: {title: "Titre"}}` → `{fr: "Titre"}` pour `title`. */
export function column(
  table: Record<string, Record<string, string>>,
  key: string,
): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [l, row] of Object.entries(table ?? {})) {
    const v = row?.[key];
    if (typeof v === "string" && v !== "") out[l] = v;
  }
  return out;
}

/**
 * `x-enum-labels` à écrire : uniquement les valeurs encore présentes dans l'enum et les
 * libellés non vides ; `null` s'il ne reste rien.
 */
export function cleanEnumLabels(
  labels: Record<string, Record<string, string>> | null | undefined,
  values: string[],
): Record<string, Record<string, string>> | null {
  const out: Record<string, Record<string, string>> = {};
  for (const [l, row] of Object.entries(labels ?? {})) {
    const kept = Object.fromEntries(
      Object.entries(row ?? {}).filter(
        ([v, text]) => values.includes(v) && text,
      ),
    );
    if (Object.keys(kept).length > 0) out[l] = kept;
  }
  return Object.keys(out).length > 0 ? out : null;
}
