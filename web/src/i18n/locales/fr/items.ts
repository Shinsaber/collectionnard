// Catalogue « items » (français, langue de référence : les autres langues en suivent la forme).
export default {
  // Vue liste (ItemsView)
  list: {
    title: "Catalogue",
    newItem: "Nouvel item",
    allTypes: "Tous les types",
    filterByType: "Filtrer par type",
    searchPlaceholder: "Rechercher…",
    tag: "Tag",
    tagPlaceholder: "Tag…",
    colLabel: "Libellé",
    colTags: "Tags",
    graph: "Graphe",
    viewGraph: "Voir le graphe",
    typesUnavailable: "Types indisponibles",
    loadFailed: "Chargement impossible",
    confirmDelete: "Supprimer « {label} » ?",
    deleted: "Item supprimé",
  },
  // Vue détail (ItemDetailView)
  detail: {
    addToCollection: "Ajouter à une collection",
    graph: "Graphe",
    attributes: "Attributs",
    defaultImageAlt: "Image par défaut : {title}",
    relations: "Relations",
    createRelation: "Créer une relation",
    noRelations: "Aucune relation trouvée.",
    inYourCollections: "Dans vos collections",
    confirmDeleteRelation: "Supprimer la relation avec « {label} » ?",
    relationDeleted: "Relation supprimée",
    relationsUnavailable: "Relations indisponibles",
    collectionsUnavailable: "Collections indisponibles",
    notFound: "Item introuvable",
  },
  // Galerie d'images (ItemGallery) — les noms de catégories restent des clés de schéma
  gallery: {
    default: "défaut",
    empty: "Aucune image dans cette catégorie.",
    imageAlt: "{category} {n}",
  },
  // Arbre des relations (RelationTree)
  relations: {
    order: "Ordre",
    subRelations: "Sous-relations",
    expand: "Déplier les sous-relations",
    collapse: "Replier les sous-relations",
    editRelation: "Modifier la relation",
    deleteRelation: "Supprimer la relation",
    noSubRelations: "Aucune sous-relation sortante.",
    subRelationsUnavailable: "Sous-relations indisponibles",
  },
  // Création / modification d'un item (ItemEditDialog)
  editDialog: {
    editTitle: "Modifier l'item",
    createTitle: "Nouvel item",
    chooseType: "Choisir un type…",
    typeLabel: "Type :",
    tags: "Tags (séparés par des virgules)",
    tagsPlaceholder: "drama, thriller, …",
    updated: "Item modifié",
    created: "Item créé",
  },
  // Ajout à une collection (AddToCollectionDialog)
  addToCollection: {
    title: "Ajouter à une collection",
    empty:
      "Aucune collection disponible (toutes contiennent déjà cet item, ou vous n'avez pas de collection).",
    collection: "Collection",
    chooseCollection: "Choisir une collection…",
    contextAttributes: "Attributs contextuels",
    added: "Item ajouté à la collection",
    collectionsUnavailable: "Collections indisponibles",
  },
  // Création d'une relation (RelationCreateDialog)
  relationCreate: {
    title: "Créer une relation",
    relationType: "Type de relation",
    chooseType: "Choisir un type…",
    direction: "Direction",
    outgoing: "→ sortant (cet item est la source)",
    incoming: "← entrant (cet item est la cible)",
    linkedItem: "Item lié",
    selectItem: "Sélectionner un item… (tapez pour rechercher)",
    attributes: "Attributs",
    typesUnavailable: "Types de relation indisponibles",
    searchFailed: "Recherche impossible",
    created: "Relation créée",
  },
  // Modification d'une relation (RelationEditDialog)
  relationEdit: {
    title: "Modifier la relation",
    attributes: "Attributs",
    noFields: "Ce type de relation n'a pas d'attributs modifiables.",
    updated: "Relation modifiée",
  },
};
