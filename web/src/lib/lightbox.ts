// Géométrie de la visionneuse plein écran (ImageLightbox, retours v0.4.0) : taille d'ajustement,
// zoom autour d'un point, déplacement borné pour que l'image ne quitte jamais l'écran. Fonctions
// pures, sans DOM, testées.

export interface Size {
  width: number;
  height: number;
}

export interface Offset {
  x: number;
  y: number;
}

/** Zoom 1 = image ajustée à la fenêtre ; au-delà, jusqu'à six fois. */
export const MIN_ZOOM = 1;
export const MAX_ZOOM = 6;

/** Déplacement horizontal minimal (px) pour passer à l'image voisine. */
export const SWIPE_THRESHOLD = 60;

export function clampZoom(zoom: number): number {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom));
}

/** Boîte occupée par l'image une fois tournée : un quart de tour impair échange largeur et hauteur. */
export function rotatedSize(size: Size, rotation: number): Size {
  const quarter = Math.round(rotation / 90) % 2;
  return quarter ? { width: size.height, height: size.width } : size;
}

/**
 * Taille de l'élément image (non tourné) au zoom 1 : son empreinte tournée tient dans la fenêtre
 * moins la marge, sans jamais agrandir une petite image.
 */
export function fitSize(
  natural: Size,
  viewport: Size,
  rotation: number,
  padding = 0,
): Size {
  const box = rotatedSize(natural, rotation);
  const availW = Math.max(1, viewport.width - 2 * padding);
  const availH = Math.max(1, viewport.height - 2 * padding);
  const scale = Math.min(
    1,
    availW / Math.max(1, box.width),
    availH / Math.max(1, box.height),
  );
  return { width: natural.width * scale, height: natural.height * scale };
}

/**
 * Décalage borné : quand l'image dépasse la fenêtre, ses bords ne rentrent jamais à l'intérieur
 * (pas de vide) ; quand elle y tient, elle reste centrée.
 */
export function clampOffset(
  offset: Offset,
  box: Size,
  zoom: number,
  viewport: Size,
): Offset {
  const maxX = Math.max(0, (box.width * zoom - viewport.width) / 2);
  const maxY = Math.max(0, (box.height * zoom - viewport.height) / 2);
  // `|| 0` : jamais de -0 (comparaisons et styles plus simples).
  return {
    x: Math.min(maxX, Math.max(-maxX, offset.x)) || 0,
    y: Math.min(maxY, Math.max(-maxY, offset.y)) || 0,
  };
}

/**
 * Zoom autour d'un point de l'écran (coordonnées relatives au centre de la fenêtre) : le détail
 * sous le doigt ou le curseur reste à sa place. Le décalage rendu reste à borner.
 */
export function zoomAround(
  offset: Offset,
  zoom: number,
  nextZoom: number,
  point: Offset,
): { zoom: number; offset: Offset } {
  const z = clampZoom(nextZoom);
  const k = z / zoom;
  return {
    zoom: z,
    offset: {
      x: point.x - (point.x - offset.x) * k,
      y: point.y - (point.y - offset.y) * k,
    },
  };
}

/** Balayage horizontal franc : -1 (vers la gauche : image suivante), 1 (précédente), 0 sinon. */
export function swipeDirection(dx: number, dy: number): -1 | 0 | 1 {
  if (Math.abs(dx) < SWIPE_THRESHOLD || Math.abs(dx) < Math.abs(dy) * 1.5)
    return 0;
  return dx < 0 ? -1 : 1;
}

export function distance(a: Offset, b: Offset): number {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

export function midpoint(a: Offset, b: Offset): Offset {
  return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
}
