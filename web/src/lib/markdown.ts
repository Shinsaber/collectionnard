import DOMPurify from "dompurify";
import MarkdownIt from "markdown-it";

// Markdown restreint des avis et commentaires (docs/phases/interactions.md, I-0) : ni HTML brut, ni
// image, ni titre, ni tableau. Le serveur stocke le source ; le rendu, assaini, se fait ici.
const md = new MarkdownIt("zero", {
  html: false,
  breaks: true,
  linkify: true,
}).enable([
  "blockquote",
  "code",
  "fence",
  "list",
  "autolink",
  "backticks",
  "emphasis",
  "entity",
  "escape",
  "link",
  "linkify",
  "newline",
  "strikethrough",
]);

const ALLOWED_TAGS = [
  "a",
  "blockquote",
  "br",
  "code",
  "em",
  "li",
  "ol",
  "p",
  "pre",
  "s",
  "strong",
  "ul",
];

/** Rend le source Markdown en HTML sûr : liens externes marqués comme contenu d'utilisateur. */
export function renderMarkdown(source: string): string {
  const html = md.render(source);
  const root = DOMPurify.sanitize(html, {
    ALLOWED_TAGS,
    ALLOWED_ATTR: ["href"],
    ALLOWED_URI_REGEXP: /^(?:https?|mailto):/i,
    RETURN_DOM_FRAGMENT: true,
  });
  root.querySelectorAll("a").forEach((a) => {
    a.setAttribute("rel", "nofollow ugc noopener");
    a.setAttribute("target", "_blank");
  });
  const out = document.createElement("div");
  out.append(root);
  return out.innerHTML;
}
