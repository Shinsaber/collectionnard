<template>
  <div class="max-w-4xl">
    <PageHeader :title="t('offline.title')" :meta="state">
      <template #subtitle>
        <p class="text-[15px] text-muted">
          {{ t("offline.subtitle") }} ·
          {{
            snapshot
              ? t("offline.dataOf", { date: generatedAt })
              : t("offline.never")
          }}
        </p>
      </template>
      <template #actions>
        <ElButton :loading="offline.syncing" @click="syncNow">
          <i
            v-if="!offline.syncing"
            class="pi pi-refresh mr-2"
            aria-hidden="true"
          ></i>
          {{ offline.reachable ? t("offline.sync") : t("offline.retry") }}
        </ElButton>
      </template>
    </PageHeader>

    <p
      v-if="!snapshot"
      class="rounded-2xl bg-surface px-6 py-10 text-center text-muted shadow-card"
    >
      {{ t("offline.noCopy") }}
    </p>

    <template v-else>
      <ElInput
        v-model="query"
        size="large"
        clearable
        :placeholder="t('offline.search')"
        :aria-label="t('offline.searchLabel')"
        class="mb-3"
      >
        <template #prefix
          ><i class="pi pi-search" aria-hidden="true"></i
        ></template>
      </ElInput>

      <!-- Filtre par collection : pastilles qui défilent sur mobile -->
      <div class="mb-4 flex gap-2 overflow-x-auto pb-1" role="group">
        <button
          v-for="c in filters"
          :key="c.id ?? 'all'"
          type="button"
          class="shrink-0 rounded-full px-3 py-1 text-[13px] font-semibold"
          :class="
            collectionId === c.id
              ? 'bg-primary text-primary-ink'
              : 'bg-surface text-ink shadow-card'
          "
          :aria-pressed="collectionId === c.id"
          @click="collectionId = c.id"
        >
          {{ c.name }}
          <span class="ml-1 font-mono text-[11px] opacity-70">{{
            c.count
          }}</span>
        </button>
      </div>

      <p class="mb-2 px-1 font-mono text-[11px] uppercase text-muted">
        {{ t("offline.results", results.length) }}
      </p>
      <ul
        v-if="results.length"
        class="divide-y divide-line overflow-hidden rounded-2xl bg-surface shadow-card"
      >
        <OfflineItemRow
          v-for="row in shown"
          :key="row.item.id"
          :row="row"
          :linkable="offline.reachable"
        />
      </ul>
      <p v-else class="px-1 text-sm text-muted">{{ t("offline.empty") }}</p>
      <p
        v-if="results.length > shown.length"
        class="mt-3 px-1 text-sm text-muted"
      >
        {{ t("offline.more", { n: results.length - shown.length }) }}
      </p>
    </template>
  </div>
</template>

<script setup lang="ts">
// Consultation hors ligne (lot E5c) : la copie locale des collections (store « offline »),
// en lecture seule, avec recherche (titre, numéro, ISBN…) et filtre par collection.
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useOfflineStore } from "@/stores/offline";
import { buildRows, searchRows } from "@/lib/offlineSearch";
import PageHeader from "@/components/common/PageHeader.vue";
import OfflineItemRow from "@/components/offline/OfflineItemRow.vue";

/** Au-delà, la liste demande d'affiner la recherche (rendu fluide sur mobile). */
const MAX_SHOWN = 100;

const { t } = useI18n();
const { lang } = useLang();
const offline = useOfflineStore();
const { notifyError } = useNotify();
const query = ref("");
const collectionId = ref<string | null>(null);

const snapshot = computed(() => offline.snapshot);
const rows = computed(() =>
  snapshot.value ? buildRows(snapshot.value, lang.value) : [],
);
const results = computed(() =>
  searchRows(rows.value, query.value, collectionId.value),
);
const shown = computed(() => results.value.slice(0, MAX_SHOWN));

const filters = computed(() => [
  {
    id: null,
    name: t("offline.allCollections"),
    count: rows.value.length,
  },
  ...(snapshot.value?.collections ?? []).map((c) => ({
    id: c.id as string | null,
    name: c.name,
    count: rows.value.filter((r) =>
      r.entries.some((e) => e.collection.id === c.id),
    ).length,
  })),
]);

const generatedAt = computed(() =>
  snapshot.value
    ? new Date(snapshot.value.generated_at).toLocaleString(lang.value, {
        dateStyle: "medium",
        timeStyle: "short",
      })
    : "",
);
const state = computed(() =>
  offline.reachable ? t("offline.onlineNow") : t("offline.offlineNow"),
);

function syncNow() {
  offline.sync(true).catch((e) => notifyError(e));
}
</script>
