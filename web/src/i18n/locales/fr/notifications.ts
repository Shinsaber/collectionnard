// Catalogue « notifications » (français) : notifications dans l'application (P4-3c).
export default {
  title: "Notifications",
  open: "Notifications ({n} non lues)",
  none: "Aucune notification.",
  markAll: "Tout marquer comme lu",
  toReview:
    "aucune proposition à relire | 1 proposition à relire | {n} propositions à relire",
  toModerate:
    "aucun contenu à modérer | 1 contenu à modérer | {n} contenus à modérer",
  kinds: {
    proposal_accepted: "{by} a accepté « {title} »",
    proposal_rejected: "{by} a refusé « {title} »",
    proposal_changes_requested:
      "{by} demande des modifications sur « {title} »",
    proposal_comment: "{by} a écrit sur « {title} »",
    review_hidden: "Un de vos avis a été masqué par la modération : {reason}",
    review_restored: "Un de vos avis a été rétabli par la modération",
  },
  someone: "Un modérateur",
};
