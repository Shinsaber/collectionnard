// Lecture des JSON Schemas des méta-types et de leurs extensions x-* pour
// l'affichage. Pendant Go de server/internal/label : les deux implémentations
// de `renderLabel` doivent rester alignées.

import { DEFAULT_LANG, LANGS, type Lang } from "@/i18n/langs";

export { LANGS, type Lang };

export type Schema = Record<string, unknown>;
export type Attributes = Record<string, unknown>;

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
export function displayValue(
  value: unknown,
  lang: Lang = DEFAULT_LANG,
): string {
  if (value == null) return "";
  if (Array.isArray(value)) {
    return value
      .map((v) => displayValue(v, lang))
      .filter((s) => s !== "")
      .join(", ");
  }
  if (typeof value === "object") {
    return i18nText(value as Record<string, unknown>, lang);
  }
  return String(value);
}

/** Ordre de repli : langue demandée, langue par défaut, langues connues, puis les autres clés triées. */
function fallbackOrder(lang: Lang, keys: string[]): string[] {
  return [lang, DEFAULT_LANG, ...LANGS, ...[...keys].sort()];
}

/**
 * Texte d'un objet i18n (`{fr: "…", en: "…"}`) : la langue demandée, puis la langue
 * par défaut, puis les autres langues connues, puis toute autre clé (langue saisie mais
 * pas encore supportée par l'interface). "" si rien. Même ordre que `label.Stringify` (Go).
 */
export function i18nText(
  obj: Record<string, unknown> | null | undefined,
  lang: Lang = DEFAULT_LANG,
): string {
  if (!obj || typeof obj !== "object") return "";
  for (const l of fallbackOrder(lang, Object.keys(obj))) {
    const v = obj[l];
    if (typeof v === "string" && v !== "") return v;
  }
  return "";
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
  lang: Lang = DEFAULT_LANG,
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
  lang: Lang = DEFAULT_LANG,
): string {
  if (!item) return "?";
  const schema = item.item_type?.attributes_schema as Schema | undefined;
  return renderLabel(schema, item.attributes, lang) || item.id.slice(0, 8);
}

/** Libellé d'un champ via `x-labels`, repli sur la clé mise en forme. */
export function fieldLabel(
  schema: Schema | null | undefined,
  key: string,
  lang: Lang = DEFAULT_LANG,
  fallback: "key" | "pretty" = "key",
): string {
  const own = byLang(schema?.["x-labels"], key, lang);
  if (own) return own;
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

/** Un lien d'un champ `x-widget: links` : adresse et libellé facultatif. */
export interface Link {
  label?: string;
  url: string;
}

/** Schéma d'un champ « liens » (tableau de `{label, url}`), écrit par le SchemaBuilder. */
export function linksProperty(): Schema {
  return {
    type: "array",
    "x-widget": "links",
    items: {
      type: "object",
      required: ["url"],
      additionalProperties: false,
      properties: {
        label: { type: "string" },
        url: { type: "string", format: "uri" },
      },
    },
  };
}

export function isLinks(
  schema: Schema | null | undefined,
  key: string,
): boolean {
  return widgetOf(schema, key) === "links";
}

/** URL affichable comme lien : http(s) seulement (jamais `javascript:` ni `data:`). */
export function safeURL(url: unknown): string {
  if (typeof url !== "string") return "";
  try {
    const u = new URL(url);
    return u.protocol === "http:" || u.protocol === "https:" ? u.href : "";
  } catch {
    return "";
  }
}

/** Liens valides d'une valeur (entrées sans URL http(s) écartées). */
export function linksOf(value: unknown): Link[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((v) => {
    const o = v as Record<string, unknown> | null;
    const url = safeURL(o?.url);
    if (!url) return [];
    const label = typeof o?.label === "string" ? o.label.trim() : "";
    return [label ? { label, url } : { url }];
  });
}

/** Texte d'un lien : son libellé, sinon le nom du site (`www.nintendo.com`). */
export function linkText(link: Link): string {
  if (link.label) return link.label;
  try {
    return new URL(link.url).hostname;
  } catch {
    return link.url;
  }
}

/**
 * Catégorie d'une galerie (`x-image-categories`). Contrôles facultatifs à l'envoi :
 * `tolerance` (écart de ratio permis, en %), `min_width` / `min_height` (pixels) ;
 * `strict` refuse l'image en défaut, sinon l'envoi se fait avec un avertissement.
 */
export interface ImageCategory {
  name: string;
  ratio: string;
  tolerance?: number;
  strict?: boolean;
  min_width?: number;
  min_height?: number;
}

/** Lecture d'une entrée de `x-image-categories` (champs inconnus ou mal typés ignorés). */
export function parseImageCategory(name: string, raw: unknown): ImageCategory {
  const v = (raw && typeof raw === "object" ? raw : {}) as Record<
    string,
    unknown
  >;
  const cat: ImageCategory = {
    name,
    ratio: typeof v.ratio === "string" && v.ratio ? v.ratio : "1/1",
  };
  for (const k of ["tolerance", "min_width", "min_height"] as const) {
    if (typeof v[k] === "number" && v[k] >= 0) cat[k] = v[k];
  }
  if (v.strict === true) cat.strict = true;
  return cat;
}

/** Catégories d'une galerie (`x-image-categories`), format objet ou tableau historique. */
export function imageCategories(
  schema: Schema | null | undefined,
  key: string,
): ImageCategory[] {
  const raw = properties(schema)[key]?.["x-image-categories"];
  if (!raw) return [];
  if (Array.isArray(raw)) {
    return (raw as string[]).map((name) => ({ name, ratio: "1/1" }));
  }
  if (typeof raw === "object") {
    return Object.entries(raw as Record<string, unknown>).map(([name, v]) =>
      parseImageCategory(name, v),
    );
  }
  return [];
}

/** Ratio CSS (« 2/3 », « 16 / 9 », « 1.5 ») en nombre largeur / hauteur ; NaN si illisible. */
export function parseRatio(ratio: string): number {
  const [w, h = "1"] = ratio.split("/").map((s) => s.trim());
  const r = Number(w) / Number(h);
  return Number.isFinite(r) && r > 0 ? r : NaN;
}

export type ImageIssue =
  | {
      kind: "ratio";
      expected: string;
      actual: number;
      deviation: number;
      tolerance: number;
    }
  | { kind: "size"; minWidth: number; minHeight: number };

/**
 * Contrôle d'une image de `width` × `height` px pour sa catégorie : écart de ratio au-delà de
 * `tolerance` (pas de contrôle sans `tolerance`), taille sous `min_width` / `min_height`.
 * `block` : la catégorie est `strict` et l'image a au moins un défaut.
 */
export function checkImage(
  cat: ImageCategory,
  width: number,
  height: number,
): { issues: ImageIssue[]; block: boolean } {
  const issues: ImageIssue[] = [];
  const expected = parseRatio(cat.ratio);
  if (cat.tolerance !== undefined && !Number.isNaN(expected) && height > 0) {
    const actual = width / height;
    const deviation = Math.abs(actual / expected - 1) * 100;
    if (deviation > cat.tolerance) {
      issues.push({
        kind: "ratio",
        expected: cat.ratio,
        actual,
        deviation,
        tolerance: cat.tolerance,
      });
    }
  }
  const minWidth = cat.min_width ?? 0;
  const minHeight = cat.min_height ?? 0;
  if (width < minWidth || height < minHeight) {
    issues.push({ kind: "size", minWidth, minHeight });
  }
  return { issues, block: !!cat.strict && issues.length > 0 };
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

/** Lecture d'une table `{<lang>: {<clé>: "texte"}}` (`x-labels`, `x-enum-labels`), même repli que `i18nText`. */
function byLang(table: unknown, key: string, lang: Lang): string {
  if (!table || typeof table !== "object") return "";
  const t = table as Record<string, Record<string, unknown> | undefined>;
  for (const l of fallbackOrder(lang, Object.keys(t))) {
    const v = t[l]?.[key];
    if (typeof v === "string" && v !== "") return v;
  }
  return "";
}

/**
 * Libellé d'une valeur d'enum via `x-enum-labels` de la propriété
 * (`{fr: {tv: "Télévision"}, en: {tv: "TV"}}`), repli sur la valeur brute.
 */
export function enumLabel(
  schema: Schema | null | undefined,
  key: string,
  value: unknown,
  lang: Lang = DEFAULT_LANG,
): string {
  const raw = value == null ? "" : String(value);
  if (raw === "") return "";
  return byLang(properties(schema)[key]?.["x-enum-labels"], raw, lang) || raw;
}

/**
 * Valeur d'affichage d'un attribut connaissant son schéma : libellé d'enum
 * (`x-enum-labels`, y compris pour un tableau de valeurs), sinon `displayValue`.
 */
export function fieldValue(
  schema: Schema | null | undefined,
  key: string,
  value: unknown,
  lang: Lang = DEFAULT_LANG,
): string {
  if (isLinks(schema, key)) return linksOf(value).map(linkText).join(", ");
  if (properties(schema)[key]?.["x-enum-labels"]) {
    if (Array.isArray(value)) {
      return value
        .map((v) => enumLabel(schema, key, v, lang))
        .filter((s) => s !== "")
        .join(", ");
    }
    if (typeof value === "string") return enumLabel(schema, key, value, lang);
  }
  const prop = properties(schema)[key];
  // Décimaux et dates mis en forme selon la langue (affichage seulement : les libellés
  // x-label gardent displayValue, aligné sur Go). Les entiers restent bruts (années, numéros).
  if (prop?.type === "number" && typeof value === "number") {
    return value.toLocaleString(lang, {
      maximumFractionDigits: 2,
      useGrouping: false,
    });
  }
  if (typeof value === "string") {
    if (prop?.format === "date") return formatDate(value, lang, false);
    if (prop?.format === "date-time") return formatDate(value, lang, true);
  }
  return displayValue(value, lang);
}

/** Date ISO (`2021-03-10`, ou date-heure) en forme locale ; valeur brute si illisible. */
function formatDate(value: string, lang: Lang, withTime: boolean): string {
  const dateOnly = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  // Date seule : lue en UTC et affichée en UTC, pour ne jamais glisser d'un jour.
  const date = dateOnly
    ? new Date(
        Date.UTC(
          Number(dateOnly[1]),
          Number(dateOnly[2]) - 1,
          Number(dateOnly[3]),
        ),
      )
    : new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return withTime && !dateOnly
    ? date.toLocaleString(lang, { dateStyle: "medium", timeStyle: "short" })
    : date.toLocaleDateString(lang, {
        dateStyle: "medium",
        ...(dateOnly ? { timeZone: "UTC" } : {}),
      });
}

/** Méta-type (ItemType, RelationType, CollectionType) tel que lu par `typeTitle` / `typeDescription`. */
export interface MetaTypeLike {
  name: string;
  description?: string | null;
  attributes_schema?: unknown;
}

/**
 * Nom affiché d'un méta-type : extension racine `x-title` de son schéma
 * (`{fr: "Série", en: "Series"}`), repli sur `name` (identifiant technique).
 */
export function typeTitle(
  type: MetaTypeLike | null | undefined,
  lang: Lang = DEFAULT_LANG,
): string {
  if (!type) return "";
  const schema = type.attributes_schema as Schema | null | undefined;
  return (
    i18nText(schema?.["x-title"] as Record<string, unknown>, lang) || type.name
  );
}

/** Description affichée d'un méta-type : `x-description` du schéma, repli sur `description`. */
export function typeDescription(
  type: MetaTypeLike | null | undefined,
  lang: Lang = DEFAULT_LANG,
): string {
  if (!type) return "";
  const schema = type.attributes_schema as Schema | null | undefined;
  return (
    i18nText(schema?.["x-description"] as Record<string, unknown>, lang) ||
    (type.description ?? "")
  );
}
