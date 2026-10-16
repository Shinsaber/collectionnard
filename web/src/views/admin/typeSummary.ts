import { fieldLabel, orderedKeys, type Schema } from "@/lib/schema";
import type { Lang } from "@/i18n";

type Translate = (key: string, n: number) => string;

/** « 10 champs » et la liste de leurs libellés, pour les listes de méta-types. */
export function schemaSummary(
  schema: Record<string, unknown> | null | undefined,
  lang: Lang,
  t: Translate,
): { countLabel: string; summary: string } {
  const keys = orderedKeys(schema as Schema | undefined);
  return {
    countLabel: t("admin.shared.fieldCount", keys.length),
    summary: keys
      .map((k) => fieldLabel(schema as Schema | undefined, k, lang))
      .join(", "),
  };
}
