// "notifications" catalog (English): in-app notifications (P4-3c).
import type fr from "../fr/notifications";

const en: typeof fr = {
  title: "Notifications",
  open: "Notifications ({n} unread)",
  none: "No notification.",
  markAll: "Mark all as read",
  toReview:
    "no proposal to review | 1 proposal to review | {n} proposals to review",
  kinds: {
    proposal_accepted: "{by} accepted “{title}”",
    proposal_rejected: "{by} rejected “{title}”",
    proposal_changes_requested: "{by} requests changes on “{title}”",
    proposal_comment: "{by} wrote on “{title}”",
  },
  someone: "A moderator",
};

export default en;
