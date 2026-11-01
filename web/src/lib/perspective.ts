// Redressement d'une couverture photographiée de biais (lot M4, mobile.md) : détection des bords
// sur une petite copie de la photo, puis transformation perspective (homographie) du quadrilatère
// vers un rectangle. JavaScript pur sur des tableaux de pixels, sans OpenCV (8 Mo) : les fonctions
// travaillent sur un `Raster` (le contenu d'un ImageData, sans dépendre du DOM) et sont testables.

export interface Raster {
  width: number;
  height: number;
  /** RGBA, 4 octets par pixel, ligne par ligne. */
  data: Uint8ClampedArray<ArrayBuffer>;
}

export interface Point {
  x: number;
  y: number;
}

/** Coins dans l'ordre haut-gauche, haut-droit, bas-droit, bas-gauche. */
export type Quad = [Point, Point, Point, Point];

export interface Detection {
  quad: Quad;
  /** 0–1 : remplissage du quadrilatère par l'objet et proximité de l'enveloppe convexe. */
  confidence: number;
}

/** Côté de la copie sur laquelle les bords sont cherchés : assez pour des coins à 1 % près. */
export const DETECT_EDGE = 320;

/** Écart de couleur minimal (somme des écarts RVB, 0–765) entre l'objet et le fond. */
const MIN_DISTANCE = 45;

/** Quadrilatère = image entière (repli quand rien n'est détecté). */
export function fullQuad(width: number, height: number): Quad {
  return [
    { x: 0, y: 0 },
    { x: width, y: 0 },
    { x: width, y: height },
    { x: 0, y: height },
  ];
}

export function scaleQuad(quad: Quad, sx: number, sy: number): Quad {
  return quad.map((p) => ({ x: p.x * sx, y: p.y * sy })) as Quad;
}

/**
 * Coins après rotation de l'image de `turns` quarts de tour horaires (négatif : antihoraires) ;
 * `width` × `height` sont les dimensions de l'image avant rotation. L'ordre haut-gauche, haut-droit,
 * bas-droit, bas-gauche est conservé : l'ancien bas-gauche devient le haut-gauche après un quart
 * de tour horaire.
 */
export function rotateQuad(
  quad: Quad,
  width: number,
  height: number,
  turns: number,
): Quad {
  let q = quad;
  let w = width;
  let h = height;
  for (let i = 0; i < ((turns % 4) + 4) % 4; i++) {
    const rot = (p: Point): Point => ({ x: h - p.y, y: p.x });
    q = [rot(q[3]), rot(q[0]), rot(q[1]), rot(q[2])];
    [w, h] = [h, w];
  }
  return q;
}

/**
 * Cherche la couverture : l'objet le plus grand qui se distingue du fond (couleur médiane de la
 * bordure de l'image), puis les quatre coins de son enveloppe convexe. Null quand rien ne ressemble
 * à un quadrilatère plein (fond uniforme, couverture occupant tout le cadre, objet trop petit).
 */
export function detectQuad(img: Raster): Detection | null {
  const { width: w, height: h, data } = img;
  const n = w * h;
  if (w < 16 || h < 16) return null;

  // 1. Couleur du fond : médiane des pixels de la bordure.
  const ring = Math.max(2, Math.round(Math.min(w, h) * 0.04));
  const hist = [
    new Uint32Array(256),
    new Uint32Array(256),
    new Uint32Array(256),
  ];
  let border = 0;
  for (let y = 0; y < h; y++) {
    const edgeRow = y < ring || y >= h - ring;
    for (let x = 0; x < w; x++) {
      if (!edgeRow && x >= ring && x < w - ring) continue;
      const i = (y * w + x) * 4;
      for (let c = 0; c < 3; c++) {
        const hs = hist[c]!;
        const v = data[i + c]!;
        hs[v] = hs[v]! + 1;
      }
      border++;
    }
  }
  const bg = hist.map((hs) => median(hs, border));

  // 2. Écart au fond par pixel et seuil d'Otsu.
  const dist = new Uint16Array(n);
  const dh = new Uint32Array(766);
  for (let p = 0, i = 0; p < n; p++, i += 4) {
    const d =
      Math.abs(data[i]! - bg[0]!) +
      Math.abs(data[i + 1]! - bg[1]!) +
      Math.abs(data[i + 2]! - bg[2]!);
    dist[p] = d;
    dh[d] = dh[d]! + 1;
  }
  const thr = Math.max(otsu(dh, n), MIN_DISTANCE);

  // 3. Plus grande composante connexe du masque (4-connexité).
  const label = new Int32Array(n);
  const queue = new Int32Array(n);
  let best = 0;
  let bestSize = 0;
  let next = 1;
  for (let start = 0; start < n; start++) {
    if (dist[start]! <= thr || label[start] !== 0) continue;
    const id = next++;
    let head = 0;
    let tail = 0;
    queue[tail++] = start;
    label[start] = id;
    while (head < tail) {
      const p = queue[head++]!;
      const x = p % w;
      if (x > 0) visit(p - 1);
      if (x < w - 1) visit(p + 1);
      if (p >= w) visit(p - w);
      if (p + w < n) visit(p + w);
    }
    if (tail > bestSize) {
      bestSize = tail;
      best = id;
    }
    function visit(q: number) {
      if (label[q] === 0 && dist[q]! > thr) {
        label[q] = id;
        queue[tail++] = q;
      }
    }
  }
  if (bestSize < 0.08 * n || bestSize > 0.98 * n) return null;

  // 4. Contour : extrêmes de chaque ligne et de chaque colonne, puis enveloppe convexe.
  const points: Point[] = [];
  const minY = new Int32Array(w).fill(-1);
  const maxY = new Int32Array(w).fill(-1);
  for (let y = 0; y < h; y++) {
    let minX = -1;
    let maxX = -1;
    for (let x = 0; x < w; x++) {
      if (label[y * w + x] !== best) continue;
      if (minX < 0) minX = x;
      maxX = x;
      if (minY[x]! < 0) minY[x] = y;
      maxY[x] = y;
    }
    if (minX >= 0) {
      points.push({ x: minX, y: y + 0.5 }, { x: maxX + 1, y: y + 0.5 });
    }
  }
  for (let x = 0; x < w; x++) {
    if (minY[x]! >= 0) {
      points.push({ x: x + 0.5, y: minY[x]! }, { x: x + 0.5, y: maxY[x]! + 1 });
    }
  }
  const hull = convexHull(points);
  if (hull.length < 4) return null;

  // 5. Coins : extrêmes de x+y et x−y sur l'enveloppe.
  const by = (f: (p: Point) => number, max: boolean) =>
    hull.reduce((a, p) => ((max ? f(p) > f(a) : f(p) < f(a)) ? p : a));
  const quad: Quad = [
    by((p) => p.x + p.y, false),
    by((p) => p.x - p.y, true),
    by((p) => p.x + p.y, true),
    by((p) => p.x - p.y, false),
  ];

  // 6. Validation : quadrilatère convexe, aux côtés francs, presque toute l'enveloppe, bien rempli.
  if (!isConvex(quad)) return null;
  const minSide = Math.min(
    distance(quad[0], quad[1]),
    distance(quad[1], quad[2]),
    distance(quad[2], quad[3]),
    distance(quad[3], quad[0]),
  );
  if (minSide < 0.1 * Math.min(w, h)) return null;
  const quadArea = polygonArea(quad);
  const shape = quadArea / polygonArea(hull);
  const fill = bestSize / quadArea;
  if (shape < 0.85 || fill < 0.7) return null;
  return { quad, confidence: Math.min(1, shape, fill) };
}

function median(hist: Uint32Array, count: number): number {
  let acc = 0;
  for (let v = 0; v < hist.length; v++) {
    acc += hist[v]!;
    if (acc * 2 >= count) return v;
  }
  return hist.length - 1;
}

/** Seuil d'Otsu (variance interclasse maximale) sur un histogramme de `total` valeurs. */
export function otsu(hist: Uint32Array, total: number): number {
  let sum = 0;
  for (let v = 0; v < hist.length; v++) sum += v * hist[v]!;
  let wB = 0;
  let sumB = 0;
  let bestVar = -1;
  let thr = 0;
  for (let v = 0; v < hist.length; v++) {
    wB += hist[v]!;
    if (wB === 0) continue;
    const wF = total - wB;
    if (wF === 0) break;
    sumB += v * hist[v]!;
    const mB = sumB / wB;
    const mF = (sum - sumB) / wF;
    const between = wB * wF * (mB - mF) * (mB - mF);
    if (between > bestVar) {
      bestVar = between;
      thr = v;
    }
  }
  return thr;
}

/** Enveloppe convexe (chaîne monotone d'Andrew), sens horaire à l'écran (y vers le bas). */
export function convexHull(points: Point[]): Point[] {
  const pts = [...points].sort((a, b) => a.x - b.x || a.y - b.y);
  if (pts.length < 3) return pts;
  const cross = (o: Point, a: Point, b: Point) =>
    (a.x - o.x) * (b.y - o.y) - (a.y - o.y) * (b.x - o.x);
  const lower: Point[] = [];
  for (const p of pts) {
    while (
      lower.length >= 2 &&
      cross(lower[lower.length - 2]!, lower[lower.length - 1]!, p) <= 0
    )
      lower.pop();
    lower.push(p);
  }
  const upper: Point[] = [];
  for (let i = pts.length - 1; i >= 0; i--) {
    const p = pts[i]!;
    while (
      upper.length >= 2 &&
      cross(upper[upper.length - 2]!, upper[upper.length - 1]!, p) <= 0
    )
      upper.pop();
    upper.push(p);
  }
  lower.pop();
  upper.pop();
  return lower.concat(upper);
}

export function polygonArea(poly: Point[]): number {
  let s = 0;
  for (let i = 0; i < poly.length; i++) {
    const a = poly[i]!;
    const b = poly[(i + 1) % poly.length]!;
    s += a.x * b.y - b.x * a.y;
  }
  return Math.abs(s) / 2;
}

export function isConvex(quad: Quad): boolean {
  let sign = 0;
  for (let i = 0; i < 4; i++) {
    const a = quad[i]!;
    const b = quad[(i + 1) % 4]!;
    const c = quad[(i + 2) % 4]!;
    const cross = (b.x - a.x) * (c.y - b.y) - (b.y - a.y) * (c.x - b.x);
    if (cross === 0) return false;
    const s = cross > 0 ? 1 : -1;
    if (sign === 0) sign = s;
    else if (s !== sign) return false;
  }
  return true;
}

export function distance(a: Point, b: Point): number {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

/**
 * Taille du rectangle redressé : moyennes des côtés opposés du quadrilatère, ramenées sous
 * `maxEdge` sans agrandir.
 */
export function quadOutputSize(
  quad: Quad,
  maxEdge: number,
): { width: number; height: number } {
  const w = (distance(quad[0], quad[1]) + distance(quad[3], quad[2])) / 2;
  const h = (distance(quad[0], quad[3]) + distance(quad[1], quad[2])) / 2;
  const scale = Math.min(1, maxEdge / Math.max(w, h, 1));
  return {
    width: Math.max(1, Math.round(w * scale)),
    height: Math.max(1, Math.round(h * scale)),
  };
}

/**
 * Homographie 3×3 (ligne par ligne, h8 = 1) envoyant chaque `src[i]` sur `dst[i]` : huit
 * équations linéaires résolues par élimination de Gauss à pivot partiel.
 */
export function homography(src: Quad, dst: Quad): number[] {
  const a: number[][] = [];
  const b: number[] = [];
  for (let i = 0; i < 4; i++) {
    const { x, y } = src[i]!;
    const { x: u, y: v } = dst[i]!;
    a.push([x, y, 1, 0, 0, 0, -u * x, -u * y]);
    b.push(u);
    a.push([0, 0, 0, x, y, 1, -v * x, -v * y]);
    b.push(v);
  }
  return [...solve(a, b), 1];
}

function solve(a: number[][], b: number[]): number[] {
  const n = b.length;
  const m = a.map((row, i) => [...row, b[i]!]);
  for (let col = 0; col < n; col++) {
    let pivot = col;
    for (let r = col + 1; r < n; r++) {
      if (Math.abs(m[r]![col]!) > Math.abs(m[pivot]![col]!)) pivot = r;
    }
    if (Math.abs(m[pivot]![col]!) < 1e-12) {
      throw new Error("degenerate quadrilateral");
    }
    [m[col], m[pivot]] = [m[pivot]!, m[col]!];
    for (let r = 0; r < n; r++) {
      if (r === col) continue;
      const f = m[r]![col]! / m[col]![col]!;
      if (f === 0) continue;
      for (let c = col; c <= n; c++) m[r]![c]! -= f * m[col]![c]!;
    }
  }
  return m.map((row, i) => row[n]! / row[i]!);
}

export function applyHomography(h: number[], p: Point): Point {
  const d = h[6]! * p.x + h[7]! * p.y + h[8]!;
  return {
    x: (h[0]! * p.x + h[1]! * p.y + h[2]!) / d,
    y: (h[3]! * p.x + h[4]! * p.y + h[5]!) / d,
  };
}

/**
 * Redresse `quad` (dans `src`) en une image `width` × `height` : correspondance inverse pixel par
 * pixel, interpolation bilinéaire, blanc opaque hors de l'image source.
 */
export function warpPerspective(
  src: Raster,
  quad: Quad,
  width: number,
  height: number,
): Raster {
  const h = homography(fullQuad(width, height), quad);
  const [h0, h1, h2, h3, h4, h5, h6, h7, h8] = h as [
    number,
    number,
    number,
    number,
    number,
    number,
    number,
    number,
    number,
  ];
  const sw = src.width;
  const sh = src.height;
  const s = src.data;
  const out = new Uint8ClampedArray(width * height * 4);
  let o = 0;
  for (let v = 0; v < height; v++) {
    const vy = v + 0.5;
    for (let u = 0; u < width; u++, o += 4) {
      const ux = u + 0.5;
      const d = h6 * ux + h7 * vy + h8;
      const x = (h0 * ux + h1 * vy + h2) / d - 0.5;
      const y = (h3 * ux + h4 * vy + h5) / d - 0.5;
      if (x < -1 || y < -1 || x > sw || y > sh) {
        out[o] = out[o + 1] = out[o + 2] = out[o + 3] = 255;
        continue;
      }
      const x0 = Math.floor(x);
      const y0 = Math.floor(y);
      const fx = x - x0;
      const fy = y - y0;
      const xa = clamp(x0, sw);
      const xb = clamp(x0 + 1, sw);
      const ya = clamp(y0, sh) * sw;
      const yb = clamp(y0 + 1, sh) * sw;
      const i00 = (ya + xa) * 4;
      const i10 = (ya + xb) * 4;
      const i01 = (yb + xa) * 4;
      const i11 = (yb + xb) * 4;
      const w00 = (1 - fx) * (1 - fy);
      const w10 = fx * (1 - fy);
      const w01 = (1 - fx) * fy;
      const w11 = fx * fy;
      for (let c = 0; c < 3; c++) {
        out[o + c] =
          s[i00 + c]! * w00 +
          s[i10 + c]! * w10 +
          s[i01 + c]! * w01 +
          s[i11 + c]! * w11;
      }
      out[o + 3] = 255;
    }
  }
  return { width, height, data: out };
}

function clamp(i: number, size: number): number {
  return i < 0 ? 0 : i >= size ? size - 1 : i;
}
