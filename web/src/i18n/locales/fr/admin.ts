// Catalogue « admin » (français, langue de référence : les autres langues en suivent la forme).
export default {
  // Commun aux trois vues de gestion des méta-types.
  shared: {
    newType: "Nouveau type",
    editType: "Modifier le type",
    technicalName: "Identifiant technique (anglais)",
    technicalNameHelp:
      "Utilisé par l'API et les données (anglais, sans accents). Le nom affiché en français et en anglais se règle dans l'éditeur d'attributs.",
    loadFailed: "Chargement impossible",
    created: "Type créé",
    updated: "Type modifié",
    deleted: "Type supprimé",
    confirmDelete: "Supprimer le type « {name} » ?",
  },
  itemTypes: {
    title: "Types d'items",
    schemaFields: "Champs du schéma",
    attributes: "Attributs",
    namePlaceholder: "Ex. : Series, Episode, DVD…",
  },
  relationTypes: {
    title: "Types de relations",
    sourcesTargets: "Sources → Cibles",
    sourceTypes: "Types source",
    targetTypes: "Types cible",
    choose: "Choisir…",
    linkAttributes: "Attributs du lien",
    namePlaceholder: "Ex. : contains_episode",
    itemTypesFailed: "Types d'items indisponibles",
    created: "Type de relation créé",
    updated: "Type de relation modifié",
    deleted: "Type de relation supprimé",
    confirmDelete: "Supprimer le type de relation « {name} » ?",
  },
  collectionTypes: {
    title: "Types de collections",
    contextFields: "Champs contextuels",
    contextAttributes: "Attributs contextuels",
    namePlaceholder: "Ex. : Ownership, Watching, Wishlist…",
  },
};
