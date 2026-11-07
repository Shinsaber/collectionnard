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
  },
  someone: "A moderator",
};

export default en;
