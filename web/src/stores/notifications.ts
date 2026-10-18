import { defineStore } from "pinia";
import { ref } from "vue";
import { notificationApi, proposalApi } from "@/api/client";
import { useAuthStore } from "./auth";
import type { AppNotification } from "@/api/types";

/**
 * Notifications de l'utilisateur (P4-3c) et, pour un modérateur, le nombre de propositions à
 * relire. Partagé par les cloches (barre latérale, barre du haut) : une seule requête en vol.
 */
export const useNotificationsStore = defineStore("notifications", () => {
  const items = ref<AppNotification[]>([]);
  const unread = ref(0);
  const toReview = ref(0);
  let pending: Promise<void> | null = null;

  function refresh(): Promise<void> {
    pending ??= (async () => {
      const auth = useAuthStore();
      try {
        const [page, count] = await Promise.all([
          notificationApi.list({ page: 1, per_page: 10 }),
          auth.canEditCatalog ? proposalApi.reviewCount() : Promise.resolve(0),
        ]);
        items.value = page.data;
        unread.value = page.unread;
        toReview.value = count;
      } finally {
        pending = null;
      }
    })();
    return pending;
  }

  async function markRead(ids?: string[]) {
    await notificationApi.read(ids);
    const now = new Date().toISOString();
    const newlyRead = items.value.filter(
      (n) => !n.read_at && (!ids || ids.includes(n.id)),
    ).length;
    items.value = items.value.map((n) =>
      !ids || ids.includes(n.id) ? { ...n, read_at: n.read_at ?? now } : n,
    );
    unread.value = ids ? Math.max(0, unread.value - newlyRead) : 0;
  }

  function $reset() {
    items.value = [];
    unread.value = 0;
    toReview.value = 0;
  }

  return { items, unread, toReview, refresh, markRead, $reset };
});
