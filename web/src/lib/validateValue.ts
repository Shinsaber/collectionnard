// Vérification d'une valeur contre le JSON Schema de son champ, avant l'envoi : mêmes règles
// que le serveur (validator.ValidateAttributes) pour les mots-clés que portent les types —
// type, enum, pattern, longueurs, bornes, format (date, uri, email), éléments d'une liste,
// propriétés d'un objet (identifiants externes, sources…), clés non prévues. Le serveur reste
// la référence : un mot-clé inconnu ici est simplement ignoré.

export type Schema = Record<string, unknown>;

/** Problème trouvé : clé i18n `errors.check.<key>`, paramètres, chemin dans la valeur. */
export interface Issue {
  key: string;
  params?: Record<string, string | number>;
  /** Chemin dans un objet ou une liste (« bnf_ark », « 2 », « sources.0.source ») ; vide au premier niveau. */
  path: string;
}

const DATE = /^(\d{4})-(\d{2})-(\d{2})$/;
const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function isEmpty(v: unknown): boolean {
  return v === undefined || v === null || v === "";
}

function typeOf(v: unknown): string {
  if (Array.isArray(v)) return "array";
  if (v === null) return "null";
  if (typeof v === "number") return Number.isInteger(v) ? "integer" : "number";
  return typeof v;
}

function typeMatches(expected: string, v: unknown): boolean {
  const actual = typeOf(v);
  if (expected === "number") return actual === "number" || actual === "integer";
  return actual === expected;
}

function validDate(s: string): boolean {
  const m = DATE.exec(s);
  if (!m) return false;
  const d = new Date(Date.UTC(+m[1]!, +m[2]! - 1, +m[3]!));
  return d.getUTCMonth() === +m[2]! - 1 && d.getUTCDate() === +m[3]!;
}

function validURL(s: string): boolean {
  try {
    new URL(s);
    return true;
  } catch {
    return false;
  }
}

function regex(pattern: string): RegExp | null {
  try {
    return new RegExp(pattern, "u");
  } catch {
    return null; // motif invalide : laissé au serveur
  }
}

function join(path: string, key: string | number): string {
  return path ? `${path}.${key}` : String(key);
}

/**
 * Premier problème d'une valeur (déjà présente) au regard de son schéma, ou null. Une valeur
 * absente n'est pas vérifiée ici : `required` relève du formulaire.
 */
export function checkValue(
  schema: Schema | undefined,
  value: unknown,
  path = "",
): Issue | null {
  if (!schema || isEmpty(value)) return null;
  const type = schema.type as string | undefined;
  if (type && !typeMatches(type, value)) {
    return { key: "type", params: { type }, path };
  }
  const enumValues = schema.enum as unknown[] | undefined;
  if (enumValues && !enumValues.includes(value)) {
    return { key: "enum", path };
  }

  if (typeof value === "string") {
    const len = [...value].length;
    if (typeof schema.minLength === "number" && len < schema.minLength)
      return { key: "minLength", params: { n: schema.minLength }, path };
    if (typeof schema.maxLength === "number" && len > schema.maxLength)
      return { key: "maxLength", params: { n: schema.maxLength }, path };
    if (typeof schema.pattern === "string") {
      const re = regex(schema.pattern);
      if (re && !re.test(value)) {
        const hint =
          (schema.description as string | undefined) ?? schema.pattern;
        return { key: "pattern", params: { hint }, path };
      }
    }
    if (schema.format === "date" && !validDate(value))
      return { key: "date", path };
    if (schema.format === "uri" && !validURL(value))
      return { key: "uri", path };
    if (schema.format === "email" && !EMAIL.test(value))
      return { key: "email", path };
  }

  if (typeof value === "number") {
    if (typeof schema.minimum === "number" && value < schema.minimum)
      return { key: "minimum", params: { n: schema.minimum }, path };
    if (typeof schema.maximum === "number" && value > schema.maximum)
      return { key: "maximum", params: { n: schema.maximum }, path };
  }

  if (Array.isArray(value)) {
    if (typeof schema.minItems === "number" && value.length < schema.minItems)
      return { key: "minItems", params: { n: schema.minItems }, path };
    if (typeof schema.maxItems === "number" && value.length > schema.maxItems)
      return { key: "maxItems", params: { n: schema.maxItems }, path };
    if (schema.uniqueItems === true) {
      const seen = new Set(value.map((v) => JSON.stringify(v)));
      if (seen.size !== value.length) return { key: "uniqueItems", path };
    }
    const items = schema.items as Schema | undefined;
    for (let i = 0; i < value.length; i++) {
      const issue = checkValue(items, value[i], join(path, i + 1));
      if (issue) return issue;
    }
  }

  if (typeOf(value) === "object") {
    const obj = value as Record<string, unknown>;
    const props =
      (schema.properties as Record<string, Schema> | undefined) ?? {};
    for (const name of (schema.required as string[] | undefined) ?? []) {
      if (isEmpty(obj[name]))
        return { key: "requiredKey", params: { key: name }, path };
    }
    const names = schema.propertyNames as Schema | undefined;
    for (const [k, v] of Object.entries(obj)) {
      if (names?.pattern && !regex(names.pattern as string)?.test(k))
        return { key: "keyPattern", params: { key: k }, path };
      if (k in props) {
        const issue = checkValue(props[k], v, join(path, k));
        if (issue) return issue;
      } else if (schema.additionalProperties === false) {
        return { key: "unknownKey", params: { key: k }, path };
      } else if (
        schema.additionalProperties &&
        typeof schema.additionalProperties === "object"
      ) {
        const issue = checkValue(
          schema.additionalProperties as Schema,
          v,
          join(path, k),
        );
        if (issue) return issue;
      }
    }
  }
  return null;
}
