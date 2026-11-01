// Styles et couleurs du graphe des relations (views/GraphView.vue) : fonctions pures, hors du
// composant, pour les tester et garder la vue sous les 400 lignes (audit Q-008).
import type cytoscape from "cytoscape";

/**
 * Couleurs des types (identité « Étiquette ») : tons sourds et distincts, tous lisibles sous un
 * texte blanc gras (contraste ≥ 4,5:1), sur fond clair comme sombre ; le pétrole de l'identité
 * en premier.
 */
export const GRAPH_PALETTE = [
  "#1d5c63", // pétrole
  "#a2342b", // brique
  "#2b5aa8", // bleu
  "#8a5300", // ambre brûlé
  "#6b3f7a", // prune
  "#4f6420", // olive
  "#8a3b5c", // framboise
  "#3b4552", // graphite
] as const;

/** Contour du nœud racine : l'ambre de l'identité. */
export const ROOT_BORDER = "#e09a2c";

/** Attribue une couleur stable à chaque type, dans l'ordre d'apparition (légende incluse). */
export function createTypeColors() {
  const colors: Record<string, string> = {};
  let next = 0;
  return {
    colors,
    colorOf(type: string): string {
      if (!colors[type]) {
        colors[type] = GRAPH_PALETTE[next % GRAPH_PALETTE.length]!;
        next++;
      }
      return colors[type]!;
    },
  };
}

export function escapeXml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

/** Badge SVG du type, en image de fond du nœud (contour ambre pour la racine). */
export function makeTypeSvg(
  type: string,
  color: string,
  isRoot: boolean,
): string {
  const charWidth = 7.5;
  const padding = 24;
  const w = Math.max(type.length * charWidth + padding, 50);
  const h = 30;
  const rx = 8;
  const borderWidth = isRoot ? 3 : 0;

  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}">
    <rect rx="${rx}" ry="${rx}" width="${w}" height="${h}" fill="${color}"
      ${borderWidth ? `stroke="${ROOT_BORDER}" stroke-width="${borderWidth}"` : ""} />
    <text x="${w / 2}" y="${h / 2}" text-anchor="middle" dominant-baseline="central"
      fill="white" font-family="system-ui, sans-serif" font-size="12"
      font-weight="bold">${escapeXml(type)}</text>
  </svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

/** Largeur d'un nœud d'après la longueur du libellé de son type. */
export function nodeWidth(typeLabel: string): number {
  return Math.max(typeLabel.length * 7.5 + 24, 50);
}

/** Styles Cytoscape aux couleurs du thème courant (réappliqués au changement de thème). */
export function graphStyle(isDark: boolean): cytoscape.StylesheetJson {
  const font = '"Schibsted Grotesk Variable", system-ui, sans-serif';
  const ink = isDark ? "#e8ecea" : "#17202b";
  const muted = isDark ? "#9aa7b0" : "#586270";
  const line = isDark ? "#3a4d58" : "#c9cec6";
  const surface = isDark ? "#17232c" : "#ffffff";
  return [
    {
      selector: "node",
      style: {
        shape: "roundrectangle",
        "background-image": "data(typeSvg)",
        "background-fit": "cover",
        "background-opacity": 0,
        "border-width": 0,
        width: "data(nodeWidth)",
        height: 30,
        // Titre sous le badge du type
        label: "data(title)",
        "text-valign": "bottom",
        "text-halign": "center",
        "text-margin-y": 6,
        "font-family": font,
        "font-size": "11px",
        "font-weight": 600,
        color: ink,
        "text-wrap": "wrap",
        "text-max-width": "140px",
      },
    },
    {
      selector: "node[?isRoot]",
      style: { "font-weight": 800, "font-size": "12px" },
    },
    {
      selector: "edge",
      style: {
        width: 1.5,
        "line-color": line,
        "target-arrow-color": line,
        "target-arrow-shape": "triangle",
        "arrow-scale": 1.1,
        "curve-style": "bezier",
        label: "data(label)",
        "font-family": font,
        "font-size": "9px",
        color: muted,
        "text-wrap": "wrap",
        "text-max-width": "120px",
        "text-rotation": "autorotate",
        "text-margin-y": -10,
        "text-background-color": surface,
        "text-background-opacity": 0.9,
        "text-background-padding": "3px",
        "text-background-shape": "roundrectangle",
      },
    },
  ];
}
