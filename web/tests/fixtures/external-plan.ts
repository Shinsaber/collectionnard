// Aperçu d'import externe pour les tests (Dragon Ball, édition Glénat 1993), aux identifiants
// des types des seeds (tests/fixtures/seed-schemas.json).
import type { ExternalPlan } from "@/api/types";

export const MANGA = "b0000000-0000-0000-0000-000000000012";
export const EDITION = "b0000000-0000-0000-0000-000000000014";
export const VOLUME = "b0000000-0000-0000-0000-000000000015";
export const PERSON = "b0000000-0000-0000-0000-000000000016";
export const PUBLISHER = "b0000000-0000-0000-0000-000000000017";
export const CREATED_BY = "c0000000-0000-0000-0000-000000000012";
export const CONTAINS_VOLUME = "c0000000-0000-0000-0000-000000000015";

const src = (source: string) => [{ source, retrieved_at: "2026-10-11" }];

export const dragonBallPlan: ExternalPlan = {
  retrieved_at: "2026-10-11",
  nodes: [
    {
      key: "wikidata:Q15051307",
      role: "work",
      item_type_id: MANGA,
      attributes: {
        title: { fr: "Dragon Ball" },
        start_year: 1984,
        sources: src("wikidata"),
      },
      status: "new",
      selected: true,
      source_url: "https://www.wikidata.org/wiki/Q15051307",
    },
    {
      key: "wikidata:Q208582",
      role: "person",
      item_type_id: PERSON,
      attributes: { first_name: "Akira", last_name: "Toriyama" },
      existing_item_id: "e0000000-0000-0000-0000-000000000105",
      status: "unchanged",
      selected: true,
      parent: "wikidata:Q15051307",
      source_url: "https://www.wikidata.org/wiki/Q208582",
    },
    {
      key: "bnf:cb343384394",
      role: "edition",
      item_type_id: EDITION,
      attributes: { title: { fr: "Dragon Ball" }, language: "fr" },
      status: "new",
      selected: true,
      parent: "wikidata:Q15051307",
      source_url: "https://catalogue.bnf.fr/ark:/12148/cb343384394",
    },
    {
      key: "publisher:glénat",
      role: "publisher",
      item_type_id: PUBLISHER,
      attributes: { name: "Glénat" },
      existing_item_id: "e0000000-0000-0000-0000-000000000107",
      status: "update",
      selected: true,
      parent: "bnf:cb343384394",
      source_url: "https://catalogue.bnf.fr/ark:/12148/cb343384394",
    },
    ...[
      ["bnf:cb356004006", 1, "Sangoku", true],
      ["bnf:cb356004068", 2, "Kamehameha", true],
      ["bnf:cb43462118n", 2, "Dragon Ball", false],
    ].map(([key, number, title, selected]) => ({
      key: key as string,
      role: "volume" as const,
      item_type_id: VOLUME,
      attributes: { number, title: { fr: title } },
      status: "new" as const,
      selected: selected as boolean,
      parent: "bnf:cb343384394",
      source_url: "https://catalogue.bnf.fr/",
      ...(selected ? {} : { note: "reprint" as const }),
    })),
  ],
  relations: [
    {
      relation_type_id: CREATED_BY,
      from: "wikidata:Q15051307",
      to: "wikidata:Q208582",
      attributes: { roles: ["author", "artist"] },
    },
    {
      relation_type_id: CONTAINS_VOLUME,
      from: "bnf:cb343384394",
      to: "bnf:cb356004006",
      attributes: { order: 1 },
    },
  ],
};
