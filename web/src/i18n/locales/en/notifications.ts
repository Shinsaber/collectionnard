// "notifications" catalog (English): in-app notifications (P4-3c).
import type fr from "../fr/notifications";

const en: typeof fr = {
  title: "Notifications",
  open: "Notifications ({n} unread)",
  none: "No notification.",
  markAll: "Mark all as read",
  toReview:
    "no proposal to review | 1 proposal to review | {n} proposals to review",
  toModerate:
    "no content to moderate | 1 content to moderate | {n} contents to moderate",
  kinds: {
    proposal_accepted: "{by} accepted “{title}”",
    proposal_rejected: "{by} rejected “{title}”",
    proposal_changes_requested: "{by} requests changes on “{title}”",
    proposal_comment: "{by} wrote on “{title}”",
    review_hidden: "One of your reviews was hidden by the moderation: {reason}",
    review_restored: "One of your reviews was restored by the moderation",
    review_comment: "{by} commented on a review",
    comment_hidden:
      "One of your comments was hidden by the moderation: {reason}",
    comment_restored: "One of your comments was restored by the moderation",
    article_published: "Your article was published",
    article_rejected: "Your article was refused: {reason}",
    article_hidden:
      "One of your articles was hidden by the moderation: {reason}",
    article_restored: "One of your articles was restored by the moderation",
    article_comment: "{by} commented on an article",
    recommendation: "{by} recommends an item to you",
  },
  someone: "A moderator",
  reviewAuthor: "The author of the review",
  member: "A member",
};

export default en;
