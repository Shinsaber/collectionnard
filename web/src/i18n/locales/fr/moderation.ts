// Modération des avis (I-1).
export default {
  title: "Modération",
  help: "Contenus signalés par les membres. Aucune sanction n'est automatique : les récapitulatifs éclairent la décision.",
  queue: "À traiter",
  log: "Journal",
  empty: "Aucun contenu signalé.",
  gone: "Avis supprimé par son auteur depuis le signalement.",
  recap: "Masqué {recent} fois en 90 jours ({total} au total)",
  reporterDismissed:
    "aucun signalement classé sans suite | 1 signalement classé sans suite | {n} signalements classés sans suite",
  hide: "Masquer",
  restore: "Rétablir",
  dismiss: "Classer sans suite",
  reasonPrompt: "Motif du masquage, montré à l'auteur :",
  by: "par {name}",
  historyOf: "Historique de {name}",
  noHistory: "Rien n'a été modéré chez ce membre.",
  dismissedReports: "Ses signalements classés sans suite",
  commentBy: "Commentaire de {name}",
  inThreadOf: "Dans le fil de cet avis :",
  kinds: { review: "avis", comment: "commentaire" },
  actions: { hide: "Masqué", restore: "Rétabli", dismiss: "Classé sans suite" },
  confirm: {
    restore: "Rétablir cet avis ? Son auteur sera prévenu.",
    dismiss: "Classer sans suite les signalements de cet avis ?",
  },
  done: {
    hide: "Avis masqué, auteur prévenu",
    restore: "Avis rétabli, auteur prévenu",
    dismiss: "Signalements classés sans suite",
  },
};
