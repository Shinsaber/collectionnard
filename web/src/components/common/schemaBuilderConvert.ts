// Conversion JSON Schema ↔ liste de champs de l'éditeur visuel (SchemaBuilder.vue) : fonctions
// pures, sorties du composant (audit Q-008). Les mots-clés que l'éditeur ne pilote pas sont
// conservés (`extra`, `itemsExtra`) et restitués à l'identique.
import { LANGS } from "@/i18n/langs";
import { isIdentifier, linksProperty } from "@/lib/schema";
import {
  LANG_CODE_PATTERN,
  MANAGED_PROPERTY_KEYS,
  cleanEnumLabels,
  column,
  parseImageCategories,
  type SchemaField,
} from "./schemaBuilderTypes";

// ── Schema → fields ─────────────────────────────────────────────────
export function parseSchema(schema: Record<string, unknown>): SchemaField[] {
  const properties = (schema.properties ?? {}) as Record<
    string,
    Record<string, unknown>
  >;
  const required = (schema.required ?? []) as string[];
  const xLabels = (schema["x-labels"] ?? {}) as Record<
    string,
    Record<string, string>
  >;
  const xOrder = (schema["x-order"] ?? []) as string[];

  const keys =
    xOrder.length > 0
      ? [
          ...xOrder,
          ...Object.keys(properties).filter((k) => !xOrder.includes(k)),
        ]
      : Object.keys(properties);

  return keys
    .filter((key) => key in properties)
    .map((key) => {
      const prop = properties[key]!;
      let type = (prop.type as string) ?? "string";
      if (type === "object" && prop["x-widget"] === "i18n-text") {
        type = "i18n-text";
      }
      if (type === "object" && prop["x-widget"] === "image-gallery") {
        type = "image-gallery";
      }
      if (type === "array" && prop["x-widget"] === "links") {
        type = "links";
      }
      return {
        key,
        type,
        labels: column(xLabels, key),
        required: required.includes(key),
        description: (prop.description as string) ?? null,
        defaultValue: prop.default ?? null,
        minLength: (prop.minLength as number) ?? null,
        maxLength: (prop.maxLength as number) ?? null,
        pattern: (prop.pattern as string) ?? null,
        format: (prop.format as string) ?? null,
        // Tableau d'enum (ex. rôles) : les valeurs vivent dans `items.enum`, les libellés
        // (`x-enum-labels`) sur la propriété, comme pour un enum simple.
        enum:
          (prop.enum as string[]) ??
          (type === "array"
            ? ((prop.items as Record<string, unknown>)?.enum as string[])
            : undefined) ??
          null,
        enumLabels:
          (prop["x-enum-labels"] as Record<string, Record<string, string>>) ??
          null,
        minimum: (prop.minimum as number) ?? null,
        maximum: (prop.maximum as number) ?? null,
        exclusiveMinimum: (prop.exclusiveMinimum as number) ?? null,
        exclusiveMaximum: (prop.exclusiveMaximum as number) ?? null,
        multipleOf: (prop.multipleOf as number) ?? null,
        itemsType:
          ((prop.items as Record<string, unknown>)?.type as string) ?? null,
        minItems: (prop.minItems as number) ?? null,
        maxItems: (prop.maxItems as number) ?? null,
        uniqueItems: (prop.uniqueItems as boolean) ?? null,
        imageCategories:
          type === "image-gallery"
            ? parseImageCategories(prop["x-image-categories"])
            : null,
        private: prop["x-private"] === true,
        identifier: isIdentifier(prop),
        extra: preservedKeys(prop, type),
        itemsExtra:
          type === "array"
            ? Object.fromEntries(
                Object.entries(
                  (prop.items ?? {}) as Record<string, unknown>,
                ).filter(([k]) => k !== "type" && k !== "enum"),
              )
            : null,
      };
    });
}

// Mots-clés conservés tels quels : ceux que le builder ne pilote pas, plus la structure
// d'un objet générique (sans widget), qu'il n'édite pas (ex. `external_ids`).
function preservedKeys(
  prop: Record<string, unknown>,
  type: string,
): Record<string, unknown> {
  const keep = (k: string) =>
    !MANAGED_PROPERTY_KEYS.has(k) ||
    (type === "object" &&
      ["properties", "additionalProperties", "propertyNames"].includes(k));
  return Object.fromEntries(Object.entries(prop).filter(([k]) => keep(k)));
}

// ── Fields → schema ─────────────────────────────────────────────────
export function buildProperty(f: SchemaField): Record<string, unknown> {
  const prop: Record<string, unknown> = { ...(f.extra ?? {}), type: f.type };
  if (f.description) prop.description = f.description;
  if (f.private) prop["x-private"] = true;
  if (f.identifier) prop["x-identifier"] = true;

  if (f.type === "string") {
    if (f.minLength != null && f.minLength > 0) prop.minLength = f.minLength;
    if (f.maxLength != null && f.maxLength > 0) prop.maxLength = f.maxLength;
    if (f.pattern) prop.pattern = f.pattern;
    if (f.format) prop.format = f.format;
    if (f.enum && f.enum.length > 0) {
      prop.enum = f.enum;
      const enumLabels = cleanEnumLabels(f.enumLabels, f.enum);
      if (enumLabels) prop["x-enum-labels"] = enumLabels;
    }
    if (f.defaultValue != null) prop.default = f.defaultValue;
  }
  if (f.type === "integer" || f.type === "number") {
    if (f.minimum != null) prop.minimum = f.minimum;
    if (f.maximum != null) prop.maximum = f.maximum;
    if (f.exclusiveMinimum != null) prop.exclusiveMinimum = f.exclusiveMinimum;
    if (f.exclusiveMaximum != null) prop.exclusiveMaximum = f.exclusiveMaximum;
    if (f.multipleOf != null) prop.multipleOf = f.multipleOf;
    if (f.defaultValue != null) prop.default = f.defaultValue;
  }
  if (f.type === "boolean" && f.defaultValue != null) {
    prop.default = f.defaultValue;
  }
  if (f.type === "links") {
    // Tableau de { label, url } : structure fixe (lib/schema.ts), seules les bornes du tableau varient.
    Object.assign(prop, linksProperty());
    if (f.minItems != null && f.minItems > 0) prop.minItems = f.minItems;
    if (f.maxItems != null && f.maxItems > 0) prop.maxItems = f.maxItems;
  }
  if (f.type === "i18n-text") {
    prop.type = "object";
    prop["x-widget"] = "i18n-text";
    // Une chaîne par langue connue ; d'autres codes de langue restent acceptés
    // (ajout futur d'une langue sans migrer les schémas existants).
    prop.properties = Object.fromEntries(
      LANGS.map((l) => [l, { type: "string" }]),
    );
    prop.propertyNames = { pattern: LANG_CODE_PATTERN };
    prop.additionalProperties = { type: "string" };
  }
  if (f.type === "image-gallery") {
    const cats = (
      f.imageCategories ?? [{ name: "poster", ratio: "1/1" }]
    ).filter((c) => c.name);
    prop.type = "object";
    prop["x-widget"] = "image-gallery";
    // Contrôles à l'envoi : seulement les valeurs renseignées (tolerance, strict, min_width…).
    prop["x-image-categories"] = Object.fromEntries(
      cats.map(({ name, ratio, ...checks }) => [
        name,
        {
          ratio: ratio || "1/1",
          ...Object.fromEntries(
            Object.entries(checks).filter(
              ([k, v]) =>
                v !== undefined && v !== null && (k !== "strict" || v),
            ),
          ),
        },
      ]),
    );
    const catProperties: Record<string, unknown> = {
      default_image: { type: "string" },
    };
    for (const c of cats) {
      catProperties[c.name] = { type: "array", items: { type: "string" } };
    }
    prop.properties = catProperties;
    prop.additionalProperties = false;
  }
  if (f.type === "array") {
    // `items` complet (ex. tableau d'objets `sources`) : seuls `type` et `enum` sont édités.
    const items: Record<string, unknown> = {
      ...(f.itemsExtra ?? {}),
      type: f.itemsType ?? "string",
    };
    if (f.enum && f.enum.length > 0) {
      items.enum = f.enum;
      const enumLabels = cleanEnumLabels(f.enumLabels, f.enum);
      if (enumLabels) prop["x-enum-labels"] = enumLabels;
    }
    prop.items = items;
    if (f.minItems != null) prop.minItems = f.minItems;
    if (f.maxItems != null) prop.maxItems = f.maxItems;
    if (f.uniqueItems) prop.uniqueItems = true;
  }
  return prop;
}
