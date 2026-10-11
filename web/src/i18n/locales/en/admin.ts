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
  importExport: {
    title: "Catalog import / export",
    exportTitle: "Export the catalog",
    exportHelp:
      "JSON file containing the meta-types (item, relation and collection types), the items, their relations and their tags. User accounts and collections are not included.",
    exportButton: "Download the JSON export",
    exportFailed: "Export failed",
    importTitle: "Import",
    importHelp:
      "Each element of the file is created if its identifier is unknown, updated otherwise; nothing is ever deleted. The import is all or nothing: on any error, no change is made. Start with a simulation.",
    file: "Export file (.json)",
    noFile: "No file chosen",
    simulate: "Simulate",
    apply: "Import",
    confirmTitle: "Import the catalog",
    confirmApply:
      "Import “{name}”? Existing elements with the same identifiers will be replaced.",
    importFailed: "Import failed",
    imported: "Catalog imported",
    reportSimulation: "Simulation result",
    reportApplied: "Import result",
    simulationOk: "Simulation without errors: you can import the file.",
    hasErrors:
      "no errors | {n} error: nothing was applied. | {n} errors: nothing was applied.",
    entity: "Element",
    created: "Created",
    updated: "Updated",
    unchanged: "Unchanged",
    removed: "Removed",
    position: "Position",
    id: "Identifier",
    message: "Message",
    wholeFile: "file",
    entities: {
      file: "File",
      item_types: "Item types",
      collection_types: "Collection types",
      relation_types: "Relation types",
      items: "Items",
      relations: "Relations",
      collection: "Collection",
    },
  },
};

export default en;
