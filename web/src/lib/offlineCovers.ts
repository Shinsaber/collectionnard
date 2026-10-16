// Couvertures hors ligne (lot E5d) : vignettes (`?w=240`, ~10–20 Ko) des items de mes
// collections, gardées dans Cache Storage sous un plafond réglable. Un index (IndexedDB, clé
// « covers ») retient la taille et la dernière consultation de chaque vignette : au-delà du
// plafond, les moins récemment consultées sont retirées. Réglages propres à l'appareil
// (localStorage).
import { defaultImage, type Schema } from "@/lib/schema";
import { COVERS_CACHE, idbGet, idbSet } from "@/lib/offlineDb";
import type { OfflineSnapshot } from "@/api/types";

export const THUMB_WIDTH = 240;
export const COVER_LIMITS_MB = [20, 50, 100, 200] as const;
const SETTINGS_KEY = "colec-offline-covers";
const MB = 1024 * 1024;

export interface CoverSettings {
  enabled: boolean;
  limitMb: number;
}

/** url → taille (octets) et dernière consultation (ms). */
export type CoverIndex = Record<string, { size: number; used: number }>;

export function loadCoverSettings(): CoverSettings {
  try {
    const raw = JSON.parse(localStorage.getItem(SETTINGS_KEY) ?? "{}");
    return {
      enabled: raw.enabled !== false,
      limitMb: COVER_LIMITS_MB.includes(raw.limitMb) ? raw.limitMb : 50,
    };
  } catch {
    return { enabled: true, limitMb: 50 };
  }
}

export function saveCoverSettings(s: CoverSettings): void {
  try {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(s));
  } catch {
    /* stockage indisponible : réglages par défaut au prochain lancement */
  }
}

/** Vignette de l'image par défaut d'un item ; vide pour une image externe ou absente. */
export function thumbUrl(
  schema: Schema | undefined,
  attributes: Record<string, unknown>,
): string {
  const url = defaultImage(schema, attributes);
  return url.startsWith("/api/v1/uploads/") ? `${url}?w=${THUMB_WIDTH}` : "";
}

/** Vignettes voulues pour la copie : une par item qui a une image envoyée sur le serveur. */
export function wantedThumbs(snap: OfflineSnapshot): string[] {
  const schemas = new Map(
    snap.item_types.map((t) => [t.id, t.attributes_schema as Schema]),
  );
  const urls = snap.items
    .map((i) => thumbUrl(schemas.get(i.item_type_id), i.attributes))
    .filter(Boolean);
  return [...new Set(urls)];
}

export function coversSize(index: CoverIndex): number {
  return Object.values(index).reduce((s, c) => s + c.size, 0);
}

/**
 * Vignettes à retirer pour qu'un ajout de `incoming` octets tienne sous `limit` : les moins
 * récemment consultées d'abord. `null` si même en vidant tout l'ajout ne tiendrait pas.
 */
export function planEviction(
  index: CoverIndex,
  limit: number,
  incoming = 0,
): string[] | null {
  if (incoming > limit) return null;
  let total = coversSize(index) + incoming;
  const out: string[] = [];
  const byAge = Object.entries(index).sort((a, b) => a[1].used - b[1].used);
  for (const [url, c] of byAge) {
    if (total <= limit) break;
    out.push(url);
    total -= c.size;
  }
  return out;
}

export async function loadCoverIndex(): Promise<CoverIndex> {
  return (await idbGet<CoverIndex>("covers").catch(() => undefined)) ?? {};
}

async function evict(cache: Cache, index: CoverIndex, urls: string[]) {
  for (const url of urls) {
    await cache.delete(url);
    delete index[url];
  }
}

/**
 * Après une synchro : retire les vignettes des items sortis de mes collections, puis télécharge
 * les manquantes tant que le plafond le permet. Sans Cache Storage (navigateur ancien,
 * navigation privée), ne fait rien.
 */
export async function syncCovers(snap: OfflineSnapshot): Promise<void> {
  const settings = loadCoverSettings();
  if (typeof caches === "undefined") return;
  const cache = await caches.open(COVERS_CACHE);
  const index = await loadCoverIndex();
  const wanted = settings.enabled ? wantedThumbs(snap) : [];
  const wantedSet = new Set(wanted);
  await evict(
    cache,
    index,
    Object.keys(index).filter((u) => !wantedSet.has(u)),
  );
  const limit = settings.limitMb * MB;
  await evict(cache, index, planEviction(index, limit) ?? []);

  for (const url of wanted.filter((u) => !index[u])) {
    const resp = await fetch(url).catch(() => null);
    if (!resp?.ok) continue;
    const blob = await resp.blob();
    const evictions = planEviction(index, limit, blob.size);
    if (evictions === null || evictions.length) break; // plafond atteint : on garde l'existant
    await cache.put(url, new Response(blob, { headers: resp.headers }));
    index[url] = { size: blob.size, used: 0 };
  }
  await idbSet("covers", index).catch(() => undefined);
}

/** Applique un nouveau plafond (ou la désactivation) sans attendre la prochaine synchro. */
export async function enforceCoverSettings(): Promise<void> {
  if (typeof caches === "undefined") return;
  const settings = loadCoverSettings();
  const cache = await caches.open(COVERS_CACHE);
  const index = await loadCoverIndex();
  const out = settings.enabled
    ? (planEviction(index, settings.limitMb * MB) ?? Object.keys(index))
    : Object.keys(index);
  await evict(cache, index, out);
  await idbSet("covers", index).catch(() => undefined);
}

// Consultations en attente d'écriture (url → date), fusionnées dans l'index du moment : une
// synchro concurrente n'est jamais écrasée.
const pendingUse = new Map<string, number>();
let useTimer: ReturnType<typeof setTimeout> | undefined;

async function flushUse() {
  const index = await loadCoverIndex();
  for (const [url, used] of pendingUse) if (index[url]) index[url]!.used = used;
  pendingUse.clear();
  await idbSet("covers", index).catch(() => undefined);
}

/**
 * Vignette gardée sur l'appareil (URL `blob:` à révoquer), ou `null`. Note la consultation
 * pour l'éviction (écrite par lots, au plus une fois par seconde).
 */
export async function cachedCover(url: string): Promise<string | null> {
  if (!url || typeof caches === "undefined") return null;
  const hit = await caches.match(url, { cacheName: COVERS_CACHE });
  if (!hit) return null;
  pendingUse.set(url, Date.now());
  clearTimeout(useTimer);
  useTimer = setTimeout(() => void flushUse(), 1000);
  return URL.createObjectURL(await hit.blob());
}
