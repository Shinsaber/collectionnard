// Lecture de l'aperçu d'un import externe (POST /external/bd-manga/preview) : regroupement
// pour l'affichage, sélection par défaut, relation qui rattache un nœud à son parent.
// Le vocabulaire (`role`) est celui du connecteur, pas des ItemTypes : aucun type n'est codé ici.
import type { ExternalNode, ExternalPlan, ExternalRelation } from "@/api/types";

export interface EditionGroup {
  edition: ExternalNode;
  publishers: ExternalNode[];
  volumes: ExternalNode[];
}

export interface PlanGroups {
  /** Œuvre, univers, personnes (dans l'ordre de l'aperçu). */
  main: ExternalNode[];
  editions: EditionGroup[];
}

const MAIN_ROLES = ["work", "universe", "person"] as const;

export function groupPlan(plan: ExternalPlan): PlanGroups {
  const main = MAIN_ROLES.flatMap((role) =>
    plan.nodes.filter((n) => n.role === role),
  );
  const editions = plan.nodes
    .filter((n) => n.role === "edition")
    .map((edition) => ({
      edition,
      publishers: plan.nodes.filter(
        (n) => n.role === "publisher" && n.parent === edition.key,
      ),
      volumes: plan.nodes.filter(
        (n) => n.role === "volume" && n.parent === edition.key,
      ),
    }));
  return { main, editions };
}

/** Clés sélectionnées par défaut (hors réimpressions et tomes sans numéro). */
export function defaultSelection(plan: ExternalPlan): string[] {
  return plan.nodes.filter((n) => n.selected).map((n) => n.key);
}

/** Relation qui relie un nœud à son parent (ex. rôles d'une personne sur l'œuvre). */
export function parentRelation(
  plan: ExternalPlan,
  node: ExternalNode,
): ExternalRelation | undefined {
  if (!node.parent) return undefined;
  return plan.relations.find(
    (r) =>
      (r.from === node.parent && r.to === node.key) ||
      (r.from === node.key && r.to === node.parent),
  );
}

/** Corrections faites dans l'aperçu : clé de nœud → attributs complets. */
export type Overrides = Record<string, Record<string, unknown>>;

/** Attributs qui seront écrits : la correction si elle existe, sinon la proposition. */
export function effectiveAttributes(
  node: ExternalNode,
  overrides: Overrides = {},
): Record<string, unknown> {
  return overrides[node.key] ?? node.attributes;
}

/** Corrections des seuls nœuds sélectionnés (le serveur refuse celles des autres). */
export function selectedOverrides(
  overrides: Overrides,
  selected: string[],
): Overrides {
  const keys = new Set(selected);
  return Object.fromEntries(
    Object.entries(overrides).filter(([k]) => keys.has(k)),
  );
}

/** Nombre de nœuds sélectionnés qui écriront quelque chose (nouveaux, à compléter ou corrigés). */
export function writtenCount(
  plan: ExternalPlan,
  selected: string[],
  overrides: Overrides = {},
): number {
  const keys = new Set(selected);
  return plan.nodes.filter(
    (n) => keys.has(n.key) && (n.status !== "unchanged" || n.key in overrides),
  ).length;
}
