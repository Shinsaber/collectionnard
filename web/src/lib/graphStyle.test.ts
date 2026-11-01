import { describe, expect, it } from "vitest";
import {
  GRAPH_PALETTE,
  ROOT_BORDER,
  createTypeColors,
  escapeXml,
  graphStyle,
  makeTypeSvg,
  nodeWidth,
} from "./graphStyle";

describe("lib/graphStyle", () => {
  it("attribue une couleur stable par type, dans l'ordre de la palette, en boucle", () => {
    const { colors, colorOf } = createTypeColors();
    expect(colorOf("Series")).toBe(GRAPH_PALETTE[0]);
    expect(colorOf("Episode")).toBe(GRAPH_PALETTE[1]);
    expect(colorOf("Series")).toBe(GRAPH_PALETTE[0]);
    for (let i = 0; i < GRAPH_PALETTE.length; i++) colorOf(`T${i}`);
    expect(Object.keys(colors)).toHaveLength(GRAPH_PALETTE.length + 2);
    expect(colorOf("T6")).toBe(GRAPH_PALETTE[0]); // 9e type : retour au début
  });

  it("échappe le XML du badge et n'entoure que la racine", () => {
    expect(escapeXml('<a & "b">')).toBe("&lt;a &amp; &quot;b&quot;&gt;");
    const root = decodeURIComponent(makeTypeSvg("Série <TV>", "#1d5c63", true));
    expect(root).toContain("&lt;TV&gt;");
    expect(root).toContain(`stroke="${ROOT_BORDER}"`);
    expect(
      decodeURIComponent(makeTypeSvg("Episode", "#000", false)),
    ).not.toContain("stroke=");
    expect(nodeWidth("Episode")).toBe(7 * 7.5 + 24);
    expect(nodeWidth("A")).toBe(50);
  });

  it("adapte les couleurs du texte au thème", () => {
    const light = graphStyle(false);
    const dark = graphStyle(true);
    const nodeColor = (s: ReturnType<typeof graphStyle>) =>
      (s[0] as { style: { color: string } }).style.color;
    expect(nodeColor(light)).not.toBe(nodeColor(dark));
    expect(light.map((s) => (s as { selector: string }).selector)).toEqual([
      "node",
      "node[?isRoot]",
      "edge",
    ]);
  });
});
