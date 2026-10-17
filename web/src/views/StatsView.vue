<template>
  <div class="flex flex-col gap-6">
    <PageHeader :title="t('stats.title')">
      <template #subtitle>
        <p class="text-[15px] text-muted">{{ t("stats.help") }}</p>
      </template>
    </PageHeader>

    <AppSpinner v-if="loading" />
    <p
      v-else-if="stats && stats.total_collections === 0"
      class="rounded-2xl bg-surface p-5 text-muted shadow-card"
      data-test="stats-empty"
    >
      {{ t("stats.emptyAll") }}
    </p>
    <template v-else-if="stats">
      <div class="grid grid-cols-2 gap-3 md:grid-cols-3">
        <StatTile
          :label="t('stats.collections')"
          :value="formatNumber(stats.total_collections, lang)"
        />
        <StatTile
          :label="t('stats.entries')"
          :value="formatNumber(stats.total_entries, lang)"
        />
        <StatTile
          :label="t('stats.distinctItems')"
          :value="formatNumber(stats.distinct_items, lang)"
        />
      </div>

      <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <section
          class="flex flex-col gap-3 rounded-2xl bg-surface p-5 shadow-card"
        >
          <h2 class="text-base font-extrabold">
            {{ t("stats.byCollection") }}
          </h2>
          <ul class="flex flex-col gap-3" data-test="stats-collections">
            <li
              v-for="c in stats.collections"
              :key="c.id"
              class="flex flex-col gap-1"
            >
              <div class="flex items-baseline justify-between gap-3">
                <RouterLink
                  :to="`/collections/${c.id}`"
                  class="min-w-0 truncate font-semibold text-link"
                  >{{ c.name }}</RouterLink
                >
                <span class="shrink-0 font-mono text-xs text-muted"
                  >{{ collectionTypeTitle(c.collection_type_id) }} ·
                  {{ formatNumber(c.item_count, lang) }}</span
                >
              </div>
              <ElProgress
                :percentage="percent(c.item_count, largest)"
                :show-text="false"
                :stroke-width="6"
                color="var(--c-primary)"
                :aria-label="`${c.name} : ${c.item_count}`"
              />
            </li>
          </ul>
        </section>
        <ChartCard
          v-if="stats.items_by_type.length > 0"
          :title="t('stats.byType')"
          kind="pie"
          :series="byType"
        />
        <ChartCard
          v-if="stats.added.length > 0"
          class="lg:col-span-2"
          :title="t('stats.added')"
          kind="line"
          :series="monthSeries(stats.added, lang)"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
// Statistiques de toutes mes collections (P4-1) ; le détail est dans l'onglet de chaque collection.
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { statsApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { typeTitle } from "@/lib/schema";
import { formatNumber, monthSeries, percent, type Series } from "@/lib/stats";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import ChartCard from "@/components/stats/ChartCard.vue";
import StatTile from "@/components/stats/StatTile.vue";
import type { MeStats } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const metaTypes = useMetaTypes();

const stats = ref<MeStats | null>(null);
const loading = ref(true);

/** Les collections arrivent de la plus grande à la plus petite. */
const largest = computed(() => stats.value?.collections[0]?.item_count ?? 0);

const byType = computed<Series>(() => {
  const list = stats.value?.items_by_type ?? [];
  return {
    labels: list.map(
      (b) =>
        typeTitle(
          metaTypes.itemTypes.find((it) => it.id === b.type_id),
          lang.value,
        ) || b.type_name,
    ),
    values: list.map((b) => b.count),
  };
});

function collectionTypeTitle(id: string): string {
  return typeTitle(
    metaTypes.collectionTypes.find((ct) => ct.id === id),
    lang.value,
  );
}

onMounted(async () => {
  try {
    const [s] = await Promise.all([
      statsApi.mine(),
      metaTypes.loadItemTypes(),
      metaTypes.loadCollectionTypes(),
    ]);
    stats.value = s;
  } catch (e) {
    notifyError(e, t("stats.loadFailed"));
  } finally {
    loading.value = false;
  }
});
</script>
