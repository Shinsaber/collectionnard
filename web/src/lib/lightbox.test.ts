import { describe, expect, it } from "vitest";
import {
  clampOffset,
  clampZoom,
  fitSize,
  rotatedSize,
  swipeDirection,
  zoomAround,
} from "./lightbox";

describe("fitSize / rotatedSize", () => {
  it("ajuste une grande image à la fenêtre sans déformer", () => {
    expect(
      fitSize({ width: 3000, height: 2000 }, { width: 600, height: 800 }, 0),
    ).toEqual({ width: 600, height: 400 });
  });

  it("n'agrandit jamais une petite image", () => {
    expect(
      fitSize({ width: 240, height: 320 }, { width: 600, height: 800 }, 0),
    ).toEqual({ width: 240, height: 320 });
  });

  it("tient compte de la rotation et de la marge", () => {
    // Tournée d'un quart, l'image 3000 × 2000 occupe 2000 × 3000 : la hauteur disponible
    // (800 − 2·20 = 760) borne son grand côté ; l'élément non tourné fait donc 760 × 506,67.
    const s = fitSize(
      { width: 3000, height: 2000 },
      { width: 600, height: 800 },
      90,
      20,
    );
    expect(s.width).toBeCloseTo(760, 5);
    expect(s.height).toBeCloseTo(506.6667, 3);
    expect(rotatedSize(s, 90)).toEqual({ width: s.height, height: s.width });
    expect(rotatedSize(s, 180)).toEqual(s);
    expect(rotatedSize(s, -90).height).toBeCloseTo(760, 5);
  });
});

describe("clampOffset", () => {
  const viewport = { width: 600, height: 800 };

  it("recentre une image qui tient dans la fenêtre", () => {
    expect(
      clampOffset({ x: 120, y: -50 }, { width: 600, height: 400 }, 1, viewport),
    ).toEqual({ x: 0, y: 0 });
  });

  it("borne le déplacement aux bords de l'image agrandie", () => {
    // Au zoom 2 : 1200 × 800 → 300 px de jeu horizontal, aucun vertical.
    const box = { width: 600, height: 400 };
    expect(clampOffset({ x: 999, y: 999 }, box, 2, viewport)).toEqual({
      x: 300,
      y: 0,
    });
    expect(clampOffset({ x: -150, y: -5 }, box, 2, viewport)).toEqual({
      x: -150,
      y: 0,
    });
  });
});

describe("zoomAround", () => {
  it("garde le point visé à sa place et borne le zoom", () => {
    const point = { x: 100, y: -40 };
    const before = { offset: { x: 20, y: 10 }, zoom: 1 };
    const detail = {
      x: (point.x - before.offset.x) / before.zoom,
      y: (point.y - before.offset.y) / before.zoom,
    };
    const after = zoomAround(before.offset, before.zoom, 2.5, point);
    expect(after.zoom).toBe(2.5);
    expect(after.offset.x + detail.x * after.zoom).toBeCloseTo(point.x, 9);
    expect(after.offset.y + detail.y * after.zoom).toBeCloseTo(point.y, 9);
    expect(zoomAround(before.offset, 1, 40, point).zoom).toBe(6);
    expect(zoomAround(before.offset, 1, 0.2, point).zoom).toBe(1);
    expect(clampZoom(3)).toBe(3);
  });
});

describe("swipeDirection", () => {
  it("ne réagit qu'à un balayage horizontal franc", () => {
    expect(swipeDirection(-80, 10)).toBe(-1);
    expect(swipeDirection(90, -20)).toBe(1);
    expect(swipeDirection(30, 0)).toBe(0);
    expect(swipeDirection(-80, 70)).toBe(0);
  });
});
