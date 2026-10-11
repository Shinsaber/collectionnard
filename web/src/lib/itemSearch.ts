// Recherche avancée du catalogue (GET /items) : champs filtrables et triables dérivés du
// JSON Schema d'un ItemType, et état de recherche ↔ paramètres de requête. Les noms de
// paramètres sont ceux de l'API (`attr.<clé>[.<op>]`, `sort=attr.<clé>`…) et servent tels
// quels dans l'URL du catalogue : une recherche se partage par lien.
// Pendant Go : server/internal/itemquery (mêmes règles type → opérateurs).
import type { LocationQuery } from "vue-router";
import type { Schema } from "./schema";

export const ATTR_PREFIX = "attr.";

/** Contrôle de saisie d'un filtre, selon le type de la propriété. */
export type FilterControl =
  | "enum" // liste de valeurs (enum, tableau d'enum) : égalité, plusieurs = OU
  | "boolean" // oui / non
  | "numberRange" // integer / number : gte + lte
  | "dateRange" // string format date / date-time : gte + lte
  | "text" // string libre : contains
  | "i18nText"; // objet i18n-text : contains dans toutes les langues

export interface FilterField {
  key: string;
  control: FilterControl;
  /** Valeurs possibles (`enum`, ou `items.enum` d'un tableau). */
  options?: string[];
  integer?: boolean;
  minimum?: number;
  maximum?: number;
}

export interface SortField {
  key: string;
  numeric: boolean;
}

type Prop = Record<string, unknown>;

function props(schema: Schema | null | undefined): Record<string, Prop> {
  return (schema?.properties ?? {}) as Record<string, Prop>;
}

function orderedPropKeys(schema: Schema | null | undefined): string[] {
  const p = props(schema);
  const order = ((schema?.["x-order"] as string[] | undefined) ?? []).filter(
    (k) => k in p,
  );
  return [...order, ...Object.keys(p).filter((k) => !order.includes(k))];
}

/** Contrôle de filtre d'une propriété, ou `null` si elle n'est pas filtrable (galerie…). */
export function filterField(key: string, prop: Prop): FilterField | null {
  const type = prop.type;
  const en = Array.isArray(prop.enum)
    ? (prop.enum as unknown[]).map(String)
    : undefined;
  switch (type) {
    case "string":
      if (en) return { key, control: "enum", options: en };
      if (prop.format === "date" || prop.format === "date-time")
        return { key, control: "dateRange" };
      return { key, control: "text" };
    case "integer":
    case "number":
      if (en) return { key, control: "enum", options: en };
      return {
        key,
        control: "numberRange",
        integer: type === "integer",
        minimum: typeof prop.minimum === "number" ? prop.minimum : undefined,
        maximum: typeof prop.maximum === "number" ? prop.maximum : undefined,
      };
    case "boolean":
      return { key, control: "boolean" };
    case "array": {
      const items = (prop.items ?? {}) as Prop;
      if (Array.isArray(items.enum))
        return {
          key,
          control: "enum",
          options: (items.enum as unknown[]).map(String),
        };
      return null;
    }
    case "object":
      return prop["x-widget"] === "i18n-text"
        ? { key, control: "i18nText" }
        : null;
  }
  return null;
}

/** Champs filtrables d'un schéma, dans l'ordre `x-order`. */
export function filterFields(schema: Schema | null | undefined): FilterField[] {
  const p = props(schema);
  return orderedPropKeys(schema)
    .map((k) => filterField(k, p[k]!))
    .filter((f): f is FilterField => f !== null);
}

/** Champs triables d'un schéma (scalaires), dans l'ordre `x-order`. */
export function sortFields(schema: Schema | null | undefined): SortField[] {
  const p = props(schema);
  return orderedPropKeys(schema)
    .filter((k) =>
      ["string", "integer", "number", "boolean"].includes(p[k]!.type as string),
    )
    .map((k) => ({
      key: k,
      numeric: p[k]!.type === "integer" || p[k]!.type === "number",
    }));
}

/** Nom du paramètre d'un filtre : `attr.year`, `attr.year.gte`… */
export function attrParam(
  key: string,
  op?: "gte" | "lte" | "contains",
): string {
  return op ? `${ATTR_PREFIX}${key}.${op}` : `${ATTR_PREFIX}${key}`;
}

/** État de recherche du catalogue (= paramètres de GET /items, hors pagination). */
export interface ItemSearch {
  item_type_id?: string;
  search?: string;
  tag?: string;
  sort?: string;
  related_to?: string;
  relation_type_id?: string;
  page?: number;
  /** Filtres d'attributs : nom de paramètre (`attr.…`) → valeurs. */
  attrs: Record<string, string[]>;
}

export function emptySearch(): ItemSearch {
  return { attrs: {} };
}

const SCALAR_KEYS = [
  "item_type_id",
  "search",
  "tag",
  "sort",
  "related_to",
  "relation_type_id",
] as const;

/** Paramètres de requête (URL ou API) ; valeurs vides omises, filtres `attr.*` répétables. */
export function toQuery(s: ItemSearch): Record<string, string | string[]> {
  const out: Record<string, string | string[]> = {};
  for (const k of SCALAR_KEYS) {
    const v = s[k]?.trim();
    if (v) out[k] = v;
  }
  if (s.page && s.page > 1) out.page = String(s.page);
  for (const [name, values] of Object.entries(s.attrs)) {
    const kept = values.map((v) => v.trim()).filter(Boolean);
    if (kept.length === 1) out[name] = kept[0]!;
    else if (kept.length > 1) out[name] = kept;
  }
  return out;
}

/** État depuis la query du routeur (valeurs invalides ignorées : l'API les validera). */
export function fromQuery(q: LocationQuery): ItemSearch {
  const one = (v: unknown): string | undefined => {
    const first = Array.isArray(v) ? v[0] : v;
    return typeof first === "string" && first !== "" ? first : undefined;
  };
  const s: ItemSearch = { attrs: {} };
  for (const k of SCALAR_KEYS) s[k] = one(q[k]);
  const page = Number(one(q.page));
  if (Number.isInteger(page) && page > 1) s.page = page;
  for (const [name, v] of Object.entries(q)) {
    if (!name.startsWith(ATTR_PREFIX)) continue;
    const values = (Array.isArray(v) ? v : [v]).filter(
      (x): x is string => typeof x === "string" && x !== "",
    );
    if (values.length > 0) s.attrs[name] = values;
  }
  return s;
}

/** Nombre de filtres d'attributs actifs (badge du panneau). */
export function activeAttrCount(s: ItemSearch): number {
  return Object.values(s.attrs).filter((v) => v.some((x) => x.trim())).length;
}
