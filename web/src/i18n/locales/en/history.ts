// "history" catalog (English): catalogue history (P4-3a).
import type fr from "../fr/history";

const en: typeof fr = {
  title: "History",
  help: "Every change of the record and of its relations, with its author.",
  created: "Added by {author} on {date}",
  updated: "Last changed by {author} on {date}",
  member: "a member",
  system: "the system",
  none: "No recorded change.",
  truncated: "Only the last 500 changes are listed.",
  showAll: "Show all ({n})",
  showLess: "Show less",
  version: "v{n}",
  item: "Record",
  relation: "Relation",
  kinds: {
    created: "Creation",
    deleted: "Deletion",
    updated: "Change",
  },
  origins: {
    direct: "Edit",
    import: "Import",
    external: "External source",
    proposal: "Accepted proposal",
    revert: "Back to version {ref}",
    system: "System",
  },
  otherItem: "linked item",
  empty: "—",
  revert: "Restore this version",
  revertConfirm:
    "Restore the attributes of version {n}? The restore itself is recorded in the history.",
  reverted: "Version {n} restored.",
};

export default en;
