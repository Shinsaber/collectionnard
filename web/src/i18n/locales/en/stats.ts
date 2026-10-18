// "stats" catalog (English): personal statistics (P4-1).
import type fr from "../fr/stats";

const en: typeof fr = {
  title: "Statistics",
  help: "Overview of my collections.",
  tab: "Statistics",
  itemsTab: "Entries",
  loadFailed: "Statistics unavailable.",
  empty: "Nothing to count yet: add items to this collection.",
  emptyAll: "No collection yet.",
  entries: "Entries",
  collections: "Collections",
  distinctItems: "Distinct items",
  byType: "By item type",
  byCollection: "By collection",
  added: "Added per month",
  numberDetail: "average {avg} · min {min} · max {max}",
  filled: "no value | 1 value | {n} values",
  yes: "Yes",
  no: "No",
  missing: "Not set",
  completeness: {
    title: "Completeness",
    help: "Sets I own part of: volumes of an edition, episodes of a season, tracks of a disc…",
    count: "{owned} of {total}",
    complete: "Complete",
    showAll: "Show all ({n})",
    showLess: "Show less",
    truncated: "Only the first 200 sets are listed.",
  },
};

export default en;
