// Propositions (P4-3b) : charge au format de l'import, résolution des conflits champ par champ.
import type {
  ProposalConflict,
  ProposalPayload,
  ProposalStatus,
} from "@/api/types";

const same = (a: unknown, b: unknown) =>
  JSON.stringify(a ?? null) === JSON.stringify(b ?? null);

/** Charge d'une proposition pour un item (nouveau : identifiant généré). */
export function itemPayload(item: {
  id?: string;
  item_type_id: string;
  attributes: Record<string, unknown>;
  tags?: string[];
}): ProposalPayload {
  return {
    items: [
      {
        id: item.id ?? crypto.randomUUID(),
        item_type_id: item.item_type_id,
        attributes: item.attributes,
        tags: item.tags ?? [],
      },
    ],
    relations: [],
  };
}

export type Side = "current" | "proposed";

/**
 * Champs en conflit : changés depuis le départ **et** différents de la proposition.
 * Un champ changé seulement d'un côté n'est pas un conflit (on garde ce changement).
 */
export function conflictFields(c: ProposalConflict): string[] {
  const base = c.base ?? {};
  const cur = c.current ?? {};
  const prop = c.proposed ?? {};
  const keys = [
    ...new Set([
      ...Object.keys(base),
      ...Object.keys(cur),
      ...Object.keys(prop),
    ]),
  ];
  return keys.filter(
    (k) =>
      !same(cur[k], prop[k]) &&
      !same(base[k], cur[k]) &&
      !same(base[k], prop[k]),
  );
}

/**
 * Attributs fusionnés à trois volets : chaque champ prend le côté qui l'a changé depuis le
 * départ ; un champ changé des deux côtés prend le choix du modérateur (proposé par défaut).
 */
export function mergeConflict(
  c: ProposalConflict,
  choices: Record<string, Side> = {},
): Record<string, unknown> {
  const base = c.base ?? {};
  const cur = c.current ?? {};
  const prop = c.proposed ?? {};
  const out: Record<string, unknown> = {};
  const keys = new Set([
    ...Object.keys(base),
    ...Object.keys(cur),
    ...Object.keys(prop),
  ]);
  for (const k of keys) {
    const curChanged = !same(base[k], cur[k]);
    const propChanged = !same(base[k], prop[k]);
    let v: unknown;
    if (curChanged && propChanged && !same(cur[k], prop[k]))
      v = (choices[k] ?? "proposed") === "current" ? cur[k] : prop[k];
    else if (propChanged) v = prop[k];
    else v = cur[k];
    if (v !== undefined) out[k] = v;
  }
  return out;
}

/** Applique les attributs fusionnés aux entités en conflit d'une charge. */
export function resolvePayload(
  payload: ProposalPayload,
  conflicts: ProposalConflict[],
  choices: Record<string, Record<string, Side>>,
): ProposalPayload {
  const merged = new Map(
    conflicts.map((c) => [
      `${c.kind}:${c.id}`,
      mergeConflict(c, choices[c.id]),
    ]),
  );
  return {
    ...payload,
    items: (payload.items ?? []).map((it) =>
      merged.has(`item:${it.id}`)
        ? { ...it, attributes: merged.get(`item:${it.id}`)! }
        : it,
    ),
    relations: (payload.relations ?? []).map((r) =>
      merged.has(`relation:${r.id}`)
        ? { ...r, attributes: merged.get(`relation:${r.id}`)! }
        : r,
    ),
  };
}

/** Classes du badge d'état (utilitaires sémantiques). */
export function statusClass(s: ProposalStatus): string {
  switch (s) {
    case "pending":
      return "bg-accent text-accent-ink";
    case "accepted":
      return "bg-tint text-tint-ink";
    case "changes_requested":
      return "bg-primary text-primary-ink";
    default:
      return "bg-soft text-muted";
  }
}
