// Historique du catalogue (P4-3a) : différences champ par champ entre deux instantanés.
import type { HistoryEntry } from "@/api/types";

export interface FieldChange {
  key: string;
  before: unknown;
  after: unknown;
}

const same = (a: unknown, b: unknown) =>
  JSON.stringify(a ?? null) === JSON.stringify(b ?? null);

/** Champs ajoutés, retirés ou modifiés entre deux instantanés d'attributs (ordre d'apparition). */
export function changedFields(
  before: Record<string, unknown> | null,
  after: Record<string, unknown> | null,
): FieldChange[] {
  const b = before ?? {};
  const a = after ?? {};
  const keys = [...new Set([...Object.keys(b), ...Object.keys(a)])];
  return keys
    .filter((k) => !same(b[k], a[k]))
    .map((k) => ({ key: k, before: b[k], after: a[k] }));
}

/** Nature d'une écriture : création, suppression ou modification. */
export function changeKind(e: HistoryEntry): "created" | "deleted" | "updated" {
  if (e.before === null) return "created";
  if (e.after === null) return "deleted";
  return "updated";
}

/** Dernière écriture de la fiche elle-même (pas d'une relation). */
export function latestItemVersion(entries: HistoryEntry[]): number {
  return entries.find((e) => e.kind === "item")?.version ?? 0;
}
