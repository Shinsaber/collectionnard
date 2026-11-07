// Recommendations and blocks (I-4).
import type fr from "../fr/recommendations";

const en: typeof fr = {
  title: "Recommendations",
  subtitle: "Items other members recommended to you.",
  none: "No recommendation received yet.",
  from: "Recommended by {name}",
  new: "New",
  open: "Open the record",
  dismiss: "Dismiss",
  dismissed: "Recommendation dismissed",
  send: {
    action: "Recommend to a member…",
    title: "Recommend this item",
    to: "Member",
    toPlaceholder: "Username",
    message: "A word (optional)",
    submit: "Send",
    sent: "Recommendation sent to {name}",
    help: "Only members of the directory can receive a recommendation. They will see your name.",
  },
  settings: {
    accept: "Accept recommendations",
    help: "Other members may recommend items to you; you are notified.",
  },
  blocks: {
    title: "Blocked members",
    help: "A blocked member can no longer recommend items to you, nor comment on or react to your signed contents; their signed reviews, articles and comments are hidden from you. Their ratings still count in the averages, and an anonymous review stays visible: hiding it would name them.",
    none: "You have not blocked anyone.",
    add: "Block",
    addPlaceholder: "Username",
    unknown: "A member",
    since: "since {date}",
    unblock: "Unblock",
    unblocked: "Member unblocked",
  },
  block: {
    action: "Block",
    actionLabel: "Block the author",
    confirm:
      "Block this member? They will no longer be able to recommend items to you nor interact with your signed contents, and their contents will be hidden from you. You can unblock them from your profile.",
    confirmTitle: "Block a member",
    done: "Member blocked",
  },
};

export default en;
