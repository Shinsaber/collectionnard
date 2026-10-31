// Préparation des images envoyées à la galerie (lot M1) : dimensions, réduction des photos
// (un téléphone produit 12 Mpx et plusieurs Mo : inutile pour une couverture, et au-delà de
// MAX_UPLOAD_SIZE), encodage d'un canvas recadré. Tout est fait dans le navigateur ; le serveur
// garde ses contrôles (type, taille).

import type { Raster } from "./perspective";

/** Plus grand côté conservé, en pixels : une couverture n'a pas besoin de plus. */
export const MAX_EDGE = 2000;

/** Côté maximal de la source d'un redressement : un peu plus que la sortie, pour l'interpolation. */
export const WARP_SOURCE_EDGE = 2500;

/** Qualité JPEG des images réduites ou recadrées. */
export const JPEG_QUALITY = 0.9;

/** Types acceptés par le serveur (`POST /uploads`) ; tout autre format décodable est réencodé avant l'envoi. */
export const UPLOAD_TYPES: ReadonlySet<string> = new Set([
  "image/jpeg",
  "image/png",
  "image/gif",
  "image/webp",
]);

export interface Size {
  width: number;
  height: number;
}

/** Dimensions de l'image ; null si le navigateur ne sait pas la décoder (le serveur contrôle le type). */
export async function imageSize(file: Blob): Promise<Size | null> {
  if (typeof createImageBitmap !== "function") return null;
  try {
    const bmp = await createImageBitmap(file);
    const size = { width: bmp.width, height: bmp.height };
    bmp.close();
    return size;
  } catch {
    return null; // format non décodable ici : pas de contrôle, l'envoi décide
  }
}

/** Dimensions ramenées dans un carré de `max` px, sans jamais agrandir ; au moins 1 × 1. */
export function fitWithin(width: number, height: number, max: number): Size {
  const scale = Math.min(1, max / Math.max(width, height, 1));
  return {
    width: Math.max(1, Math.round(width * scale)),
    height: Math.max(1, Math.round(height * scale)),
  };
}

export type OutputType = "image/png" | "image/jpeg";

/** Type de sortie d'une image retravaillée : le PNG garde sa transparence, tout le reste devient JPEG. */
export function outputType(type: string): OutputType {
  return type === "image/png" ? "image/png" : "image/jpeg";
}

/** Nom de fichier avec l'extension du type de sortie (`photo.HEIC` → `photo.jpg`). */
export function outputName(name: string, type: OutputType): string {
  const base = name.replace(/\.[^.]+$/, "") || "image";
  return `${base}.${type === "image/png" ? "png" : "jpg"}`;
}

function canvasToBlob(
  canvas: HTMLCanvasElement,
  type: OutputType,
  quality: number,
): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) =>
        blob ? resolve(blob) : reject(new Error("image encoding failed")),
      type,
      quality,
    );
  });
}

/**
 * Encode `source` (canvas recadré ou bitmap) dans `type`, réduit à `max` px de côté si besoin.
 * Un JPEG n'a pas de transparence : fond blanc sous les zones vides (hors de l'image).
 */
export async function encodeImage(
  source: HTMLCanvasElement | ImageBitmap,
  type: OutputType,
  max = MAX_EDGE,
): Promise<Blob> {
  const { width, height } = fitWithin(source.width, source.height, max);
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("canvas unavailable");
  if (type === "image/jpeg") {
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, width, height);
  }
  ctx.drawImage(source, 0, 0, width, height);
  return canvasToBlob(canvas, type, JPEG_QUALITY);
}

/**
 * Photo gardée telle quelle : réduite à `max` px de côté si elle le dépasse (et réencodée), ou
 * réencodée si son format n'est pas accepté par le serveur (HEIC, AVIF, BMP… décodables ici) ;
 * sinon rendue inchangée. Un fichier non décodable est rendu tel quel : le serveur tranchera.
 */
export async function shrinkIfNeeded(
  file: File,
  max = MAX_EDGE,
): Promise<File> {
  if (typeof createImageBitmap !== "function") return file;
  let bmp: ImageBitmap;
  try {
    bmp = await createImageBitmap(file);
  } catch {
    return file;
  }
  try {
    if (Math.max(bmp.width, bmp.height) <= max && UPLOAD_TYPES.has(file.type))
      return file;
    const type = outputType(file.type);
    const blob = await encodeImage(bmp, type, max);
    return new File([blob], outputName(file.name, type), { type });
  } finally {
    bmp.close();
  }
}

/** Image courante d'un flux vidéo (prise de vue sans `ImageCapture`), en JPEG, sans réduction. */
export function frameOf(video: HTMLVideoElement): Promise<Blob> {
  const canvas = document.createElement("canvas");
  canvas.width = video.videoWidth;
  canvas.height = video.videoHeight;
  if (!canvas.width || !canvas.height) throw new Error("no video frame");
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("canvas unavailable");
  ctx.drawImage(video, 0, 0);
  return canvasToBlob(canvas, "image/jpeg", JPEG_QUALITY);
}

// ── Pixels (redressement, lot M4) ──────────────────────────────────────

/** Pixels d'un bitmap, réduit à `max` px de côté (copie dessinée dans un canvas). */
export function rasterOf(bmp: ImageBitmap, max: number): Raster {
  const { width, height } = fitWithin(bmp.width, bmp.height, max);
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("canvas unavailable");
  ctx.drawImage(bmp, 0, 0, width, height);
  return { width, height, data: ctx.getImageData(0, 0, width, height).data };
}

/** Canvas portant les pixels d'un raster (pour l'encoder avec `encodeImage`). */
export function canvasOf(raster: Raster): HTMLCanvasElement {
  const canvas = document.createElement("canvas");
  canvas.width = raster.width;
  canvas.height = raster.height;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("canvas unavailable");
  ctx.putImageData(
    new ImageData(raster.data, raster.width, raster.height),
    0,
    0,
  );
  return canvas;
}
