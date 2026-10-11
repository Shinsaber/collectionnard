import type fr from "../fr/admin";

const en: typeof fr = {
  shared: {
    newType: "New type",
    editType: "Edit type",
    technicalName: "Technical identifier (English)",
    technicalNameHelp:
      "Used by the API and the data (English, no accents). The displayed name in French and English is set in the attribute editor.",
    loadFailed: "Loading failed",
    created: "Type created",
    updated: "Type updated",
    deleted: "Type deleted",
    confirmDelete: "Delete the type “{name}”?",
  },
  itemTypes: {
    title: "Item types",
    schemaFields: "Schema fields",
    attributes: "Attributes",
    namePlaceholder: "e.g. Series, Episode, DVD…",
  },
  relationTypes: {
    title: "Relation types",
    sourcesTargets: "Sources → Targets",
    sourceTypes: "Source types",
    targetTypes: "Target types",
    choose: "Choose…",
    linkAttributes: "Link attributes",
    namePlaceholder: "e.g. contains_episode",
    itemTypesFailed: "Item types unavailable",
    created: "Relation type created",
    updated: "Relation type updated",
    deleted: "Relation type deleted",
    confirmDelete: "Delete the relation type “{name}”?",
  },
  collectionTypes: {
    title: "Collection types",
    contextFields: "Contextual fields",
    contextAttributes: "Contextual attributes",
    namePlaceholder: "e.g. Ownership, Watching, Wishlist…",
  },
};

export default en;
