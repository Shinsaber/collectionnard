import { describe, expect, it } from "vitest";
import { renderMarkdown } from "./markdown";

describe("renderMarkdown", () => {
  it("rend la mise en forme permise", () => {
    expect(renderMarkdown("**gras** et *italique*")).toContain(
      "<strong>gras</strong> et <em>italique</em>",
    );
    expect(renderMarkdown("- un\n- deux")).toContain("<li>un</li>");
    expect(renderMarkdown("> cité")).toContain("<blockquote>");
    expect(renderMarkdown("a\nb")).toContain("<br>");
  });

  it("marque les liens comme contenu d'utilisateur", () => {
    const html = renderMarkdown(
      "[site](https://example.org) https://a.example",
    );
    expect(html).toContain('href="https://example.org"');
    expect(html).toContain('rel="nofollow ugc noopener"');
    expect(html).toContain('target="_blank"');
    expect(html).toContain('href="https://a.example"');
  });

  it("n'exécute ni HTML brut ni lien dangereux", () => {
    for (const source of [
      "<script>alert(1)</script>",
      "<img src=x onerror=alert(1)>",
      '<a href="javascript:alert(1)">x</a>',
    ]) {
      const div = document.createElement("div");
      div.innerHTML = renderMarkdown(source);
      expect(div.querySelector("script, img, a")).toBeNull();
    }
    expect(renderMarkdown("[x](javascript:alert(1))")).not.toContain("href=");
    expect(renderMarkdown("[x](data:text/html,bad)")).not.toContain("href=");
  });

  it("refuse les images et les titres", () => {
    expect(renderMarkdown("![a](https://example.org/a.png)")).not.toContain(
      "<img",
    );
    expect(renderMarkdown("# titre")).not.toContain("<h1");
  });
});
