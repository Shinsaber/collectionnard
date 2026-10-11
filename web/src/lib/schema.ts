// Lecture des JSON Schemas des méta-types et de leurs extensions x-* pour
// l'affichage. Pendant Go de server/internal/label : les deux implémentations
// de `renderLabel` doivent rester alignées.

export type Schema = Record<string, unknown>;
export type Attributes = Record<string, unknown>;
export type Lang = "fr" | "en";

export const LANGS: Lang[] = ["fr", "en"];

/** Clés essayées, dans l'ordre, quand le schéma n'a pas de `x-label`. */
export const FALLBACK_LABEL_KEYS = [
  "title",
  "name",
  "label",
  "titre",
  "nom",
  "number",
  "numero",
];

const PLACEHOLDER = /\{([A-Za-z0-9_.-]+)\}/g;

function properties(schema?: Schema | null): Record<string, Schema> {
  return (schema?.properties ?? {}) as Record<string, Schema>;
}

/** Valeur d'affichage d'un attribut : objet i18n → langue demandée puis repli, tableau → liste, etc. */
export function displayValue(value: unknown, lang: Lang = "fr"): string {
  if (value == null) return "";
  if (Array.isArray(value)) {
    return value
      .map((v) => displayValue(v, lang))
      .filter((s) => s !== "")
      .join(", ");
  }
  if (typeof value === "object") {
    const obj = value as Record<string, unknown>;
    const own = obj[lang];
    if (typeof own === "string" && own !== "") return own;
    for (const l of LANGS) {
      const v = obj[l];
      if (typeof v === "string" && v !== "") return v;
    }
    return "";
  }
  return String(value);
}

/** Gabarit `x-label` du schéma, ou "" s'il n'en a pas. */
export function labelTemplate(schema?: Schema | null): string {
  const tpl = schema?.["x-label"];
  return typeof tpl === "string" ? tpl.trim() : "";
}

/**
 * Libellé d'une instance à partir du schéma de son méta-type et de ses attributs :
 * gabarit `x-label` (`"{number}. {title}"`) ou première clé de repli trouvée, sinon "".
 */
export function renderLabel(
  schema: Schema | null | undefined,
  attrs: Attributes | null | undefined,
  lang: Lang = "fr",
): string {
  if (!attrs || typeof attrs !== "object") return "";
  const tpl = labelTemplate(schema);
  if (tpl) {
    return tpl
      .replace(PLACEHOLDER, (_m, key: string) => displayValue(attrs[key], lang))
      .split(/\s+/)
      .filter(Boolean)
      .join(" ");
  }
  for (const key of FALLBACK_LABEL_KEYS) {
    if (key in attrs) {
      const s = displayValue(attrs[key], lang);
      if (s !== "") return s;
    }
  }
  return "";
}

/** Libellé d'un item (schéma de son ItemType, repli sur le début de l'id). */
export function itemLabel(
  item:
    | {
        id: string;
        attributes: Attributes;
        item_type?: { attributes_schema?: unknown } | null;
      }
    | null
    | undefined,
  lang: Lang = "fr",
): string {
  if (!item) return "?";
  const schema = item.item_type?.attributes_schema as Schema | undefined;
  return renderLabel(schema, item.attributes, lang) || item.id.slice(0, 8);
}

/** Libellé d'un champ via `x-labels`, repli sur la clé mise en forme. */
export function fieldLabel(
  schema: Schema | null | undefined,
  key: string,
  lang: Lang = "fr",
  fallback: "key" | "pretty" = "key",
): string {
  const labels = schema?.["x-labels"] as
    Record<string, Record<string, string>> | undefined;
  const own = labels?.[lang]?.[key];
  if (own) return own;
  for (const l of LANGS) {
    const v = labels?.[l]?.[key];
    if (v) return v;
  }
  if (fallback === "pretty") {
    return key.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
  }
  return key;
}

/** Clés d'attributs dans l'ordre `x-order`, puis les clés restantes (schéma puis données). */
export function orderedKeys(
  schema: Schema | null | undefined,
  attrs?: Attributes | null,
): string[] {
  const xOrder = (schema?.["x-order"] as string[] | undefined) ?? [];
  const known = new Set<string>();
  const out: string[] = [];
  const push = (k: string) => {
    if (!known.has(k)) {
      known.add(k);
      out.push(k);
    }
  };
  const propKeys = Object.keys(properties(schema));
  const attrKeys = attrs ? Object.keys(attrs) : [];
  const universe = new Set([...propKeys, ...attrKeys]);
  for (const k of xOrder) if (universe.has(k)) push(k);
  for (const k of propKeys) push(k);
  for (const k of attrKeys) push(k);
  return out;
}

/** Valeur `x-widget` d'une propriété, ou undefined. */
export function widgetOf(
  schema: Schema | null | undefined,
  key: string,
): string | undefined {
  const w = properties(schema)[key]?.["x-widget"];
  return typeof w === "string" ? w : undefined;
}

export function isImageGallery(
  schema: Schema | null | undefined,
  key: string,
): boolean {
  return widgetOf(schema, key) === "image-gallery";
}

/** Catégories d'une galerie (`x-image-categories`), format objet ou tableau historique. */
export function imageCategories(
  schema: Schema | null | undefined,
  key: string,
): { name: string; ratio: string }[] {
  const raw = properties(schema)[key]?.["x-image-categories"];
  if (!raw) return [];
  if (Array.isArray(raw)) {
    return (raw as string[]).map((name) => ({ name, ratio: "1/1" }));
  }
  if (typeof raw === "object") {
    return Object.entries(raw as Record<string, { ratio?: string }>).map(
      ([name, v]) => ({ name, ratio: v?.ratio ?? "1/1" }),
    );
  }
  return [];
}

/**
 * Première image d'un item : `default_image` de la première galerie du schéma,
 * sinon la première image de la première catégorie non vide.
 */
export function defaultImage(
  schema: Schema | null | undefined,
  attrs: Attributes | null | undefined,
): string {
  if (!attrs) return "";
  for (const key of Object.keys(properties(schema))) {
    if (!isImageGallery(schema, key)) continue;
    const gallery = attrs[key];
    if (!gallery || typeof gallery !== "object" || Array.isArray(gallery))
      continue;
    const g = gallery as Record<string, unknown>;
    if (typeof g.default_image === "string" && g.default_image)
      return g.default_image;
    for (const cat of imageCategories(schema, key)) {
      const list = g[cat.name];
      if (Array.isArray(list) && typeof list[0] === "string" && list[0])
        return list[0];
    }
  }
  return "";
}
