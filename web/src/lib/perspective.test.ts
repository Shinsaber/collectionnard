import { describe, expect, it } from "vitest";
import {
  applyHomography,
  convexHull,
  detectQuad,
  fullQuad,
  homography,
  isConvex,
  otsu,
  polygonArea,
  quadOutputSize,
  rotateQuad,
  scaleQuad,
  warpPerspective,
  type Point,
  type Quad,
  type Raster,
} from "./perspective";

type RGB = [number, number, number];

function raster(width: number, height: number, fill: RGB): Raster {
  const data = new Uint8ClampedArray(width * height * 4);
  for (let i = 0; i < data.length; i += 4) {
    data[i] = fill[0];
    data[i + 1] = fill[1];
    data[i + 2] = fill[2];
    data[i + 3] = 255;
  }
  return { width, height, data };
}

/** Point dans un polygone convexe (sens horaire à l'écran). */
function inside(poly: Point[], p: Point): boolean {
  let sign = 0;
  for (let i = 0; i < poly.length; i++) {
    const a = poly[i]!;
    const b = poly[(i + 1) % poly.length]!;
    const cross = (b.x - a.x) * (p.y - a.y) - (b.y - a.y) * (p.x - a.x);
    if (cross === 0) continue;
    const s = cross > 0 ? 1 : -1;
    if (sign === 0) sign = s;
    else if (s !== sign) return false;
  }
  return true;
}

function paint(img: Raster, poly: Point[], color: (p: Point) => RGB) {
  for (let y = 0; y < img.height; y++) {
    for (let x = 0; x < img.width; x++) {
      const p = { x: x + 0.5, y: y + 0.5 };
      if (!inside(poly, p)) continue;
      const i = (y * img.width + x) * 4;
      const [r, g, b] = color(p);
      img.data[i] = r;
      img.data[i + 1] = g;
      img.data[i + 2] = b;
    }
  }
}

const near = (a: Point, b: Point, tol: number) =>
  Math.abs(a.x - b.x) <= tol && Math.abs(a.y - b.y) <= tol;

describe("detectQuad", () => {
  it("trouve les coins d'une couverture sombre, de biais, sur une table claire", () => {
    const img = raster(240, 180, [225, 220, 210]);
    const truth: Quad = [
      { x: 50, y: 30 },
      { x: 190, y: 44 },
      { x: 180, y: 160 },
      { x: 42, y: 150 },
    ];
    paint(img, truth, () => [40, 60, 120]);
    const det = detectQuad(img);
    expect(det).not.toBeNull();
    for (let i = 0; i < 4; i++) {
      expect(near(det!.quad[i]!, truth[i]!, 2.5), `coin ${i}`).toBe(true);
    }
    expect(det!.confidence).toBeGreaterThan(0.9);
  });

  it("accepte une couverture claire aux motifs variés sur un fond sombre", () => {
    const img = raster(200, 200, [30, 30, 30]);
    const truth: Quad = [
      { x: 30, y: 20 },
      { x: 170, y: 30 },
      { x: 165, y: 185 },
      { x: 35, y: 175 },
    ];
    paint(img, truth, (p) => [
      200 + (p.x % 40 < 20 ? 40 : 0),
      150 + (p.y % 30 < 15 ? 60 : 0),
      120,
    ]);
    const det = detectQuad(img);
    expect(det).not.toBeNull();
    for (let i = 0; i < 4; i++) {
      expect(near(det!.quad[i]!, truth[i]!, 2.5), `coin ${i}`).toBe(true);
    }
  });

  it("ne détecte rien sur un fond uniforme, un objet minuscule ou une image trop petite", () => {
    expect(detectQuad(raster(120, 90, [200, 200, 200]))).toBeNull();
    const tiny = raster(120, 90, [200, 200, 200]);
    paint(tiny, fullQuad(8, 8), () => [0, 0, 0]);
    expect(detectQuad(tiny)).toBeNull();
    expect(detectQuad(raster(10, 10, [0, 0, 0]))).toBeNull();
  });

  it("rejette une forme qui n'est pas un quadrilatère plein (disque)", () => {
    const img = raster(200, 200, [240, 240, 240]);
    for (let y = 0; y < 200; y++) {
      for (let x = 0; x < 200; x++) {
        if (Math.hypot(x - 100, y - 100) > 70) continue;
        const i = (y * 200 + x) * 4;
        img.data[i] = img.data[i + 1] = img.data[i + 2] = 20;
      }
    }
    expect(detectQuad(img)).toBeNull();
  });
});

describe("géométrie", () => {
  it("otsu sépare deux populations", () => {
    const hist = new Uint32Array(766);
    hist[20] = 500;
    hist[400] = 300;
    const thr = otsu(hist, 800);
    expect(thr).toBeGreaterThanOrEqual(20);
    expect(thr).toBeLessThan(400);
  });

  it("enveloppe convexe, aire et convexité", () => {
    const hull = convexHull([
      { x: 0, y: 0 },
      { x: 4, y: 0 },
      { x: 2, y: 1 }, // intérieur
      { x: 4, y: 3 },
      { x: 0, y: 3 },
    ]);
    expect(hull).toHaveLength(4);
    expect(polygonArea(hull)).toBe(12);
    expect(isConvex(fullQuad(4, 3))).toBe(true);
    expect(
      isConvex([
        { x: 0, y: 0 },
        { x: 4, y: 0 },
        { x: 1, y: 1 }, // rentrant
        { x: 0, y: 3 },
      ]),
    ).toBe(false);
  });

  it("taille de sortie : moyenne des côtés, bornée sans agrandir", () => {
    const q: Quad = [
      { x: 0, y: 0 },
      { x: 100, y: 0 },
      { x: 120, y: 160 },
      { x: 0, y: 140 },
    ];
    // Haut 100, bas ≈ 121,7 → 111 ; gauche 140, droite ≈ 161,2 → 151.
    expect(quadOutputSize(q, 2000)).toEqual({ width: 111, height: 151 });
    expect(quadOutputSize(q, 75)).toEqual({ width: 55, height: 75 });
    expect(scaleQuad(fullQuad(2, 3), 10, 100)).toEqual(fullQuad(20, 300));
  });
});

describe("homography / warpPerspective", () => {
  it("envoie exactement les quatre coins et conserve l'identité", () => {
    const src: Quad = [
      { x: 10, y: 20 },
      { x: 110, y: 15 },
      { x: 120, y: 160 },
      { x: 5, y: 150 },
    ];
    const dst = fullQuad(100, 140);
    const h = homography(src, dst);
    for (let i = 0; i < 4; i++) {
      const p = applyHomography(h, src[i]!);
      expect(p.x).toBeCloseTo(dst[i]!.x, 6);
      expect(p.y).toBeCloseTo(dst[i]!.y, 6);
    }
    const id = homography(dst, dst);
    expect(applyHomography(id, { x: 33, y: 77 })).toEqual({ x: 33, y: 77 });
    expect(() => homography(fullQuad(0, 0), dst)).toThrow();
  });

  it("redresse un quadrilatère bicolore en un rectangle aux moitiés nettes", () => {
    const img = raster(200, 200, [255, 255, 255]);
    const quad: Quad = [
      { x: 40, y: 30 },
      { x: 160, y: 50 },
      { x: 150, y: 170 },
      { x: 30, y: 160 },
    ];
    // Moitié gauche rouge, moitié droite bleue (séparées par la médiane verticale du quadrilatère).
    const mid = (p: Point) => {
      const top = {
        x: (quad[0].x + quad[1].x) / 2,
        y: (quad[0].y + quad[1].y) / 2,
      };
      const bottom = {
        x: (quad[3].x + quad[2].x) / 2,
        y: (quad[3].y + quad[2].y) / 2,
      };
      return (
        (bottom.x - top.x) * (p.y - top.y) - (bottom.y - top.y) * (p.x - top.x)
      );
    };
    paint(img, quad, (p) => (mid(p) > 0 ? [255, 0, 0] : [0, 0, 255]));
    const out = warpPerspective(img, quad, 60, 60);
    const px = (x: number, y: number) => {
      const i = (y * 60 + x) * 4;
      return [out.data[i], out.data[i + 1], out.data[i + 2]];
    };
    expect(px(10, 30)).toEqual([255, 0, 0]);
    expect(px(50, 30)).toEqual([0, 0, 255]);
    expect(px(10, 5)).toEqual([255, 0, 0]);
    expect(px(50, 55)).toEqual([0, 0, 255]);
    expect(out.data[3]).toBe(255);
  });

  it("l'identité à la même taille rend l'image d'origine, et le hors-champ est blanc", () => {
    const img = raster(8, 6, [10, 20, 30]);
    img.data[(2 * 8 + 3) * 4] = 200;
    const same = warpPerspective(img, fullQuad(8, 6), 8, 6);
    expect(Array.from(same.data)).toEqual(Array.from(img.data));
    const outside = warpPerspective(
      img,
      [
        { x: -20, y: -20 },
        { x: -10, y: -20 },
        { x: -10, y: -10 },
        { x: -20, y: -10 },
      ],
      4,
      4,
    );
    expect(Array.from(outside.data.slice(0, 4))).toEqual([255, 255, 255, 255]);
  });
});

describe("rotateQuad", () => {
  it("un quart de tour horaire : l'image entière reste l'image entière, dans l'ordre", () => {
    expect(rotateQuad(fullQuad(100, 50), 100, 50, 1)).toEqual(
      fullQuad(50, 100),
    );
    expect(rotateQuad(fullQuad(100, 50), 100, 50, -1)).toEqual(
      fullQuad(50, 100),
    );
  });

  it("tourne les coins avec l'image et revient au départ après quatre quarts", () => {
    const q: Quad = [
      { x: 10, y: 5 },
      { x: 90, y: 8 },
      { x: 85, y: 45 },
      { x: 12, y: 40 },
    ];
    // Horaire : l'ancien bas-gauche (12, 40) devient le haut-gauche (50 − 40, 12).
    const once = rotateQuad(q, 100, 50, 1);
    expect(once[0]).toEqual({ x: 10, y: 12 });
    expect(once[1]).toEqual({ x: 45, y: 10 });
    expect(rotateQuad(once, 50, 100, 3)).toEqual(q);
    expect(rotateQuad(q, 100, 50, 4)).toEqual(q);
    expect(rotateQuad(q, 100, 50, -1)).toEqual(rotateQuad(q, 100, 50, 3));
  });
});
