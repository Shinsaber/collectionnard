import { defineStore } from "pinia";
import { ref, shallowRef } from "vue";
import axios from "axios";
import { offlineApi } from "@/api/client";
import { clearOfflineData, idbGet, idbSet } from "@/lib/offlineDb";
import type { OfflineSnapshot } from "@/api/types";

/** Délai minimal entre deux synchronisations automatiques (ouverture, retour du réseau). */
const SYNC_INTERVAL = 5 * 60_000;

/**
 * Copie hors ligne des collections (lot E5b) : chargée depuis IndexedDB, rafraîchie par
 * GET /me/offline quand le serveur répond. `reachable` passe à faux dès qu'un appel échoue
 * faute de réseau (navigator.onLine ne suffit pas : Wi-Fi sans Internet en magasin).
 */
export const useOfflineStore = defineStore("offline", () => {
  // shallowRef : copie volumineuse, jamais modifiée en place (remplacée à chaque synchro).
  const snapshot = shallowRef<OfflineSnapshot | null>(null);
  const reachable = ref(typeof navigator === "undefined" || navigator.onLine);
  const syncing = ref(false);
  let lastSync = 0;
  let started = false;
  /** Étapes ajoutées après chaque synchro réussie (couvertures, lot E5d). */
  const afterSync: ((snap: OfflineSnapshot) => Promise<void>)[] = [];

  async function load() {
    snapshot.value =
      (await idbGet<OfflineSnapshot>("snapshot").catch(() => undefined)) ??
      null;
  }

  async function sync(force = false) {
    if (syncing.value || (!force && Date.now() - lastSync < SYNC_INTERVAL))
      return;
    syncing.value = true;
    try {
      const snap = await offlineApi.snapshot();
      reachable.value = true;
      lastSync = Date.now();
      snapshot.value = snap;
      await idbSet("snapshot", snap).catch(() => undefined);
      // Safari efface sinon les données d'un site peu visité au bout de quelques semaines.
      await navigator.storage?.persist?.().catch(() => false);
      for (const step of afterSync) await step(snap).catch(() => undefined);
    } catch (e) {
      if (axios.isAxiosError(e) && !e.response) reachable.value = false;
      else throw e;
    } finally {
      syncing.value = false;
    }
  }

  /** Une fois, à l'entrée dans l'application : copie locale, synchro, écoute du réseau. */
  function start() {
    if (started) return;
    started = true;
    void load().then(() => sync().catch(() => undefined));
    window.addEventListener("online", () => {
      reachable.value = true;
      sync().catch(() => undefined);
    });
    window.addEventListener("offline", () => (reachable.value = false));
  }

  /** Déconnexion : la copie appartient à l'utilisateur, elle ne reste pas sur l'appareil. */
  async function clear() {
    snapshot.value = null;
    lastSync = 0;
    await clearOfflineData();
  }

  return { snapshot, reachable, syncing, afterSync, load, sync, start, clear };
});
