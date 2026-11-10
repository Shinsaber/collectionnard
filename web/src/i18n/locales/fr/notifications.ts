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
    review_comment: "{by} a commenté un avis",
    comment_hidden:
      "Un de vos commentaires a été masqué par la modération : {reason}",
    comment_restored: "Un de vos commentaires a été rétabli par la modération",
    article_published: "Votre article a été publié",
    article_rejected: "Votre article a été refusé : {reason}",
    article_hidden:
      "Un de vos articles a été masqué par la modération : {reason}",
    article_restored: "Un de vos articles a été rétabli par la modération",
    article_comment: "{by} a commenté un article",
    recommendation: "{by} vous recommande un item",
    friend_request: "{by} vous propose d'être amis",
    friend_accepted: "{by} et vous êtes maintenant amis",
    loan_started: "{by} vous a prêté un item",
    loan_overdue: "Retour en retard : {borrower}",
    gift_expiring: "Votre réservation pour {by} expire le {date}",
    gift_left_list: "Un cadeau réservé pour {by} a quitté sa liste",
    loan_overdue_borrowed: "{by} attend le retour d'un prêt",
  },
  someone: "Un modérateur",
  reviewAuthor: "L'auteur de l'avis",
  member: "Un membre",
};
