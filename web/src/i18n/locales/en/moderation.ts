import type fr from "../fr/moderation";

const en: typeof fr = {
  title: "Moderation",
  help: "Contents reported by members. No sanction is automatic: the summaries inform the decision.",
  queue: "To handle",
  log: "Journal",
  empty: "No reported content.",
  gone: "Review deleted by its author since the report.",
  recap: "Hidden {recent} times in 90 days ({total} in all)",
  reporterDismissed:
    "no dismissed report | 1 dismissed report | {n} dismissed reports",
  hide: "Hide",
  restore: "Restore",
  dismiss: "Dismiss",
  reasonPrompt: "Reason of the hiding, shown to the author:",
  by: "by {name}",
  historyOf: "Record of {name}",
  noHistory: "Nothing of this member was moderated.",
  dismissedReports: "Their dismissed reports",
  commentBy: "Comment by {name}",
  inThreadOf: "In the thread of this review:",
  kinds: { review: "review", comment: "comment" },
  actions: { hide: "Hidden", restore: "Restored", dismiss: "Dismissed" },
  confirm: {
    restore: "Restore this review? Its author will be told.",
    dismiss: "Dismiss the reports of this review?",
  },
  done: {
    hide: "Review hidden, author told",
    restore: "Review restored, author told",
    dismiss: "Reports dismissed",
  },
};

export default en;
