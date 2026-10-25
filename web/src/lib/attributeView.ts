// Affichage des attributs d'un item (bloc « Informations ») : tout se déduit du schéma.
// Liens (`x-widget: links`, `format: uri`) en libellés cliquables, objets en sous-liste
// clé : valeur, tableaux d'objets en liste, identifiants (`x-identifier`) en chasse fixe.
import type { Lang } from "@/i18n/langs";
import {
  fieldLabel,
  fieldValue,
  isImageGallery,
  isLinks,
  linksOf,
  orderedKeys,
  safeURL,
  widgetOf,
  type Link,
  type Schema,
} from "./schema";

export type AttrView =
  | { kind: "text"; text: string }
  | { kind: "links"; links: Link[] }
  | { kind: "object"; rows: AttrRow[] }
  | { kind: "list"; items: AttrView[] };

export interface AttrRow {
  key: string;
  label: string;
  view: AttrView;
  /** Identifiant (ISBN, QID…) : chasse fixe. */
  mono: boolean;
}

/** Au-delà, un objet imbriqué s'affiche en texte (JSON compact, qui passe à la ligne). */
const MAX_DEPTH = 2;

function isPlainObject(v: unknown): v is Record<string, unknown> {
  return !!v && typeof v === "object" && !Array.isArray(v);
}

function propOf(schema: Schema | undefined, key: string): Schema | undefined {
  return (schema?.properties as Record<string, Schema> | undefined)?.[key];
}

function viewOf(
  parent: Schema | undefined,
  key: string,
  value: unknown,
  lang: Lang,
  depth: number,
): AttrView {
  const prop = propOf(parent, key);
  if (isLinks(parent, key)) return { kind: "links", links: linksOf(value) };
  if (prop?.format === "uri") {
    const url = safeURL(value);
    if (url) return { kind: "links", links: [{ url }] };
  }
  // Texte multilingue, enum, scalaire : la valeur d'affichage habituelle.
  if (widgetOf(parent, key) === "i18n-text" || !isPlainObject(value)) {
    if (Array.isArray(value) && value.some(isPlainObject)) {
      const items = prop?.items as Schema | undefined;
      return {
        kind: "list",
        items: value.map((v) =>
          isPlainObject(v)
            ? objectView(items, v, lang, depth + 1)
            : { kind: "text", text: fieldValue(parent, key, v, lang) },
        ),
      };
    }
    return { kind: "text", text: fieldValue(parent, key, value, lang) };
  }
  return objectView(prop, value, lang, depth + 1);
}

function objectView(
  schema: Schema | undefined,
  value: Record<string, unknown>,
  lang: Lang,
  depth: number,
): AttrView {
  if (depth > MAX_DEPTH) return { kind: "text", text: JSON.stringify(value) };
  return { kind: "object", rows: attributeRows(schema, value, lang, depth) };
}

/** Lignes d'un objet d'attributs, dans l'ordre du schéma ; seules les clés renseignées. */
export function attributeRows(
  schema: Schema | undefined,
  attrs: Record<string, unknown>,
  lang: Lang,
  depth = 0,
): AttrRow[] {
  return orderedKeys(schema, attrs)
    .filter(
      (k) =>
        k in attrs &&
        attrs[k] !== null &&
        attrs[k] !== "" &&
        !isImageGallery(schema, k),
    )
    .map((key) => ({
      key,
      label: fieldLabel(schema, key, lang, "pretty"),
      view: viewOf(schema, key, attrs[key], lang, depth),
      mono: propOf(schema, key)?.["x-identifier"] === true,
    }))
    .filter((r) => !isEmpty(r.view));
}

function isEmpty(v: AttrView): boolean {
  switch (v.kind) {
    case "text":
      return v.text === "";
    case "links":
      return v.links.length === 0;
    case "object":
      return v.rows.length === 0;
    case "list":
      return v.items.length === 0;
  }
}
