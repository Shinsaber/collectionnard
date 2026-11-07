import type fr from "../fr/articles";

const en: typeof fr = {
  tab: "News",
  share: "Share an article",
  none: "No article on this record yet.",
  published: "Article published",
  waiting: "Article sent: it will appear once approved by a moderator",
  confirmDelete: "Delete your article?",
  status: {
    pending: "Waiting for approval",
    rejected: "Refused",
    hidden: "Hidden by the moderation",
  },
  form: {
    title: "Title",
    url: "Link",
    message: "A word (optional)",
    send: "Send",
    moderated:
      "Your article will appear once approved by a moderator. The server never opens the link.",
  },
};

export default en;
