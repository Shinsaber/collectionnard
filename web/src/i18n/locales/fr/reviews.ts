// Avis sur un item (I-0, docs/phases/interactions.md).
export default {
  tab: "Avis",
  outOfTen: "{rating} sur 10",
  anonymousAuthor: "Anonyme",
  showSpoiler: "Afficher le spoiler",
  count: "aucune note | {n} note | {n} notes",
  belowThreshold:
    "Pas encore assez de notes pour une moyenne ({count} sur {min}).",
  detailedCount: "{n} avis détaillé | {n} avis détaillé | {n} avis détaillés",
  mine: "Mon avis",
  myReviews: "Mes avis",
  publicReviews: "Avis",
  noMyReviews: "Vous n'avez pas encore écrit d'avis.",
  noPublicReviews: "Aucun avis signé.",
  write: "Donner mon avis",
  none: "Pas encore d'avis d'autres membres.",
  noteOnly:
    "et aucune note sans texte | et {n} note sans texte | et {n} notes sans texte",
  myLangOnly: "Dans ma langue",
  contextOf: "{label} :",
  allContexts: "Toutes les éditions",
  saved: "Avis enregistré",
  deleted: "Avis supprimé",
  confirmDelete: "Supprimer votre avis ?",
  header: {
    open: "Voir les avis",
    average: "{rating}/10",
    notEnough: "{count}, moyenne à partir de {min}",
    none: "Aucune note",
    mine: "Ma note",
  },
  sort: {
    label: "Trier les avis",
    recent: "Les plus récents",
    rating: "Les mieux notés",
  },
  visibility: {
    private: "Note seule",
    privateHelp:
      "Votre note compte dans la moyenne, sans votre nom ; votre texte reste pour vous.",
    members: "Membres",
    membersHelp:
      "Avis signé pour les membres ; les visiteurs n'en voient que la note, dans la moyenne.",
    anonymous: "Anonyme",
    anonymousHelp: "Texte visible de tous, sans votre nom.",
    public: "Public",
    publicHelp: "Avis signé, visible de tous.",
  },
  form: {
    contextDefault: "Édition utilisée",
    contextNone: "Non précisée",
    contextHelp:
      "Facultatif. Elle n'est pas affichée si votre avis est anonyme.",
    contextAnonymous: "Avis anonyme : l'édition ne sera pas affichée.",
    mode: "Façon de noter",
    overall: "Note globale",
    byCriteria: "Par critère",
    rating: "Note sur 10",
    average: "Note de l'avis (moyenne) :",
    body: "Votre avis (facultatif)",
    bodyHelp:
      "Markdown : *italique*, **gras**, listes, > citation, [lien](https://…). Ni image ni HTML.",
    spoiler: "Contient un spoiler",
    lang: "Langue",
    visibility: "Qui voit cet avis ?",
  },
};
