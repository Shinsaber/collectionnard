// Types et constantes partagés par SchemaBuilder.vue (coquille) et SchemaFieldCard.vue (un champ).

/** Options d'affichage du builder : chaque drapeau montre ou cache une contrainte. */
export interface SchemaBuilderFeatures {
  labels: boolean; // x-labels FR/EN
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
};

/** Un champ du schéma tel qu'édité dans le builder. */
export interface SchemaField {
  key: string;
  type: string;
  labelFr: string;
  labelEn: string;
  required: boolean;
  description?: string | null;
  defaultValue?: unknown;
  // string
  minLength?: number | null;
  maxLength?: number | null;
  pattern?: string | null;
  format?: string | null;
  enum?: string[] | null;
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
  imageCategories?: { name: string; ratio: string }[] | null;
  /** Mots-clés non gérés par le builder (futurs x-*, default sur un objet…), restitués tels quels. */
  extra?: Record<string, unknown>;
}

export function emptyField(): SchemaField {
  return {
    key: "",
    type: "string",
    labelFr: "",
    labelEn: "",
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
    itemsType: null,
    minItems: null,
    maxItems: null,
    uniqueItems: null,
    imageCategories: null,
    extra: {},
  };
}

export interface Option<T = string> {
  label: string;
  value: T;
}

export const baseTypeOptions: Option[] = [
  { label: "Texte", value: "string" },
  { label: "Entier", value: "integer" },
  { label: "Nombre", value: "number" },
  { label: "Booléen", value: "boolean" },
];
export const arrayTypeOption: Option = { label: "Tableau", value: "array" };
export const i18nTypeOption: Option = {
  label: "Texte i18n",
  value: "i18n-text",
};
export const imageGalleryTypeOption: Option = {
  label: "Galerie d'images",
  value: "image-gallery",
};

export const formatOptions: Option[] = [
  { label: "URI / URL", value: "uri" },
  { label: "Date", value: "date" },
  { label: "Date-Time", value: "date-time" },
  { label: "Email", value: "email" },
];

export const boolDefaultOptions: Option<boolean>[] = [
  { label: "Vrai", value: true },
  { label: "Faux", value: false },
];

export const arrayItemTypeOptions: Option[] = [
  { label: "Texte", value: "string" },
  { label: "Entier", value: "integer" },
  { label: "Nombre", value: "number" },
  { label: "Booléen", value: "boolean" },
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
  "x-widget",
  "x-image-categories",
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
]);

/** Parse « neuf, bon, mauvais » en tableau d'enum (null si vide). */
export function parseEnum(value: string): string[] | null {
  const items = value
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  return items.length > 0 ? items : null;
}

/** `x-image-categories` : format objet `{poster: {ratio}}` ou tableau historique de noms. */
export function parseImageCategories(
  raw: unknown,
): { name: string; ratio: string }[] | null {
  if (!raw) return null;
  if (typeof raw === "object" && !Array.isArray(raw)) {
    return Object.entries(raw as Record<string, { ratio?: string }>).map(
      ([name, v]) => ({ name, ratio: v?.ratio ?? "1/1" }),
    );
  }
  if (Array.isArray(raw)) {
    return (raw as string[]).map((name) => ({ name, ratio: "1/1" }));
  }
  return null;
}
