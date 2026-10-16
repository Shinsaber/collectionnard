// Stockage hors ligne (lot E5) : une base IndexedDB « colec-offline » à un magasin clé → valeur.
// Clés : « snapshot » (copie des collections, GET /me/offline) et « covers » (index des
// couvertures gardées, lot E5d). Les fichiers des couvertures vivent dans Cache Storage.
// Toutes les fonctions échouent sans bruit si IndexedDB est indisponible (navigation privée…) :
// le mode hors ligne n'est alors simplement pas proposé.

const DB_NAME = "colec-offline";
const STORE = "kv";
/** Cache Storage des vignettes de couvertures (lot E5d). */
export const COVERS_CACHE = "colec-covers";

let opening: Promise<IDBDatabase> | null = null;

function open(): Promise<IDBDatabase> {
  opening ??= new Promise<IDBDatabase>((resolve, reject) => {
    if (typeof indexedDB === "undefined") {
      reject(new Error("IndexedDB unavailable"));
      return;
    }
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => req.result.createObjectStore(STORE);
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  }).catch((e: unknown) => {
    opening = null; // nouvel essai au prochain appel
    throw e;
  });
  return opening;
}

function run<T>(
  mode: IDBTransactionMode,
  fn: (store: IDBObjectStore) => IDBRequest,
): Promise<T> {
  return open().then(
    (db) =>
      new Promise<T>((resolve, reject) => {
        const req = fn(db.transaction(STORE, mode).objectStore(STORE));
        req.onsuccess = () => resolve(req.result as T);
        req.onerror = () => reject(req.error);
      }),
  );
}

export function idbGet<T>(key: string): Promise<T | undefined> {
  return run<T | undefined>("readonly", (s) => s.get(key));
}

export function idbSet(key: string, value: unknown): Promise<void> {
  return run<void>("readwrite", (s) => s.put(value, key));
}

/** Efface toutes les données hors ligne (déconnexion, « Vider ») : base et couvertures. */
export async function clearOfflineData(): Promise<void> {
  await run<void>("readwrite", (s) => s.clear()).catch(() => undefined);
  if (typeof caches !== "undefined") {
    await caches.delete(COVERS_CACHE).catch(() => false);
  }
}
