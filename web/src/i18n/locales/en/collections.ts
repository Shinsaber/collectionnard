import type fr from "../fr/collections";

const en: typeof fr = {
  import: {
    button: "Import",
    title: "Import a collection",
    help: "Choose a file exported from a collection (JSON). If it matches one of your collections, that collection becomes exactly the content of the file: items missing from the file are removed from it. Otherwise a new collection is created. Items must exist in the catalogue. Start with a simulation.",
    file: "Collection export (.json)",
    confirm: "Import “{name}”?",
    confirmWithRemoval:
      "Import this file? | Import this file? {n} item missing from the file will be removed from the collection. | Import this file? {n} items missing from the file will be removed from the collection.",
    done: "Collection imported",
  },
  itemCount: "no items | {n} item | {n} items",
  loadFailed: "Loading failed",
  typesUnavailable: "Collection types unavailable",
  list: {
    title: "My collections",
    new: "New collection",
    allTypes: "All types",
    filterByType: "Filter by collection type",
    empty: "No collections yet. Create one to get started!",
    editTitle: "Edit collection",
    collectionType: "Collection type",
    choose: "Choose…",
    namePlaceholder: "My collection of…",
    updated: "Collection updated",
    created: "Collection created",
    deleted: "Collection deleted",
    confirmDelete:
      "Delete the collection “{name}”? Its items will not be deleted.",
  },
  detail: {
    backToList: "Back to collections",
    addItem: "Add an item",
    item: "Item",
    contextAttributes: "Contextual attributes",
    editAttributes: "Edit attributes",
    removeFromCollection: "Remove from collection",
    searchItem: "Search for an item",
    searchPlaceholder: "Search…",
    label: "Label",
    notFound: "Collection not found",
    unavailable: "Collection unavailable",
    searchFailed: "Search failed",
    itemAdded: "Item added",
    attributesUpdated: "Attributes updated",
    confirmRemove: "Remove “{label}” from the collection?",
    remove: "Remove",
    itemRemoved: "Item removed",
    export: "Export",
    exportFailed: "Export failed",
  },
};

export default en;
