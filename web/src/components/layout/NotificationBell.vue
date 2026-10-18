<template>
  <ElPopover
    trigger="click"
    placement="bottom-end"
    :width="340"
    @show="store.refresh().catch(notifyError)"
  >
    <template #reference>
      <button
        type="button"
        class="relative flex size-11 items-center justify-center rounded-full text-ink hover:bg-soft"
        :aria-label="t('notifications.open', { n: badge })"
        data-test="notification-bell"
      >
        <i class="pi pi-bell text-lg" aria-hidden="true"></i>
        <span
          v-if="badge > 0"
          class="absolute top-1 right-1 min-w-4.5 rounded-full bg-accent px-1 font-mono text-[10px] font-bold text-accent-ink"
          aria-hidden="true"
          >{{ badge > 99 ? "99+" : badge }}</span
        >
      </button>
    </template>

    <div class="flex flex-col gap-2" data-test="notification-panel">
      <div class="flex items-center justify-between gap-2">
        <strong>{{ t("notifications.title") }}</strong>
        <ElButton
          v-if="store.unread > 0"
          text
          size="small"
          @click="store.markRead().catch(notifyError)"
          >{{ t("notifications.markAll") }}</ElButton
        >
      </div>
      <RouterLink
        v-if="store.toReview > 0"
        :to="{ path: '/proposals', query: { scope: 'review' } }"
        class="rounded-lg bg-accent px-3 py-2 text-sm font-semibold text-accent-ink"
        data-test="to-review"
        >{{ t("notifications.toReview", store.toReview) }}</RouterLink
      >
      <p v-if="store.items.length === 0" class="text-sm text-muted">
        {{ t("notifications.none") }}
      </p>
      <ul v-else class="flex flex-col">
        <li v-for="n in store.items" :key="n.id">
          <RouterLink
            :to="
              n.payload.proposal_id
                ? `/proposals/${n.payload.proposal_id}`
                : '/proposals'
            "
            class="flex flex-col rounded-lg px-2 py-1.5 text-sm hover:bg-soft"
            :class="n.read_at ? 'text-muted' : 'font-semibold text-ink'"
            @click="read(n)"
          >
            <span>{{ text(n) }}</span>
            <span class="font-mono text-[11px] text-muted">{{
              new Date(n.created_at).toLocaleString(lang, {
                dateStyle: "short",
                timeStyle: "short",
              })
            }}</span>
          </RouterLink>
        </li>
      </ul>
    </div>
  </ElPopover>
</template>

<script setup lang="ts">
// Cloche des notifications (P4-3c) : non lues, et propositions à relire pour un modérateur.
// Relue à chaque changement de page ; un clic sur une notification la marque comme lue.
import { computed, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useNotificationsStore } from "@/stores/notifications";
import type { AppNotification } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const route = useRoute();
const store = useNotificationsStore();

const badge = computed(() => store.unread + store.toReview);

function text(n: AppNotification): string {
  return t(`notifications.kinds.${n.kind}`, {
    by: n.payload.by ?? t("notifications.someone"),
    title: n.payload.title ?? "",
  });
}

function read(n: AppNotification) {
  if (!n.read_at) store.markRead([n.id]).catch(notifyError);
}

/** Erreur réseau silencieuse ici : la cloche n'est pas le contenu de la page. */
const refresh = () => store.refresh().catch(() => undefined);
onMounted(refresh);
watch(() => route.path, refresh);
</script>
