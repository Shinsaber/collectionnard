<template>
  <AppSpinner v-if="loading" />
  <p
    v-else-if="stats && stats.total_items === 0"
    class="rounded-2xl bg-surface p-5 text-muted shadow-card"
    data-test="stats-empty"
  >
    {{ t("stats.empty") }}
  </p>
  <div v-else-if="stats" class="flex flex-col gap-4" data-test="stats">
    <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
      <StatTile
        :label="t('stats.entries')"
        :value="formatNumber(stats.total_items, lang)"
      />
      <StatTile
        v-for="n in stats.numbers"
        :key="n.field"
        :label="fieldLabel(schema, n.field, lang, 'pretty')"
        :value="formatNumber(n.sum, lang)"
        :detail="numberDetail(n)"
        data-test="number-stat"
      />
    </div>

    <CompletenessList
      v-if="stats.completeness.length > 0"
      :entries="stats.completeness"
      :truncated="stats.completeness_truncated"
    />

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <ChartCard
        v-if="stats.items_by_type.length > 1"
        :title="t('stats.byType')"
        kind="bar"
        horizontal
        :series="byType"
      />
      <ChartCard
        v-for="d in stats.distributions"
        :key="d.field"
        :title="fieldLabel(schema, d.field, lang, 'pretty')"
        kind="bar"
        :series="distributionSeries(d, schema, lang, words)"
      />
      <ChartCard
        v-for="tl in datedTimelines"
        :key="tl.field"
        :title="fieldLabel(schema, tl.field, lang, 'pretty')"
        kind="line"
        :series="monthSeries(tl.months, lang)"
      />
      <ChartCard
        v-if="stats.added.length > 0"
        :title="t('stats.added')"
        kind="line"
        :series="monthSeries(stats.added, lang)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
// Statistiques d'une collection (P4-1) : tout se déduit du schéma de son CollectionType.
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { statsApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { fieldLabel, typeTitle, type Schema } from "@/lib/schema";
import {
  distributionSeries,
  formatNumber,
  monthSeries,
  type Series,
} from "@/lib/stats";
import AppSpinner from "@/components/common/AppSpinner.vue";
import ChartCard from "./ChartCard.vue";
import CompletenessList from "./CompletenessList.vue";
import StatTile from "./StatTile.vue";
import type { CollectionStats, NumberStat } from "@/api/types";

const props = defineProps<{
  collectionId: string;
  schema?: Schema;
  /** Change à chaque ajout / retrait : les statistiques sont relues. */
  version?: number;
}>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const metaTypes = useMetaTypes();

const stats = ref<CollectionStats | null>(null);
const loading = ref(true);

const words = computed(() => ({
  yes: t("stats.yes"),
  no: t("stats.no"),
  missing: t("stats.missing"),
}));

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

/** Une date jamais renseignée ne fait pas de courbe vide. */
const datedTimelines = computed(
  () => stats.value?.timelines.filter((tl) => tl.months.length > 0) ?? [],
);

function numberDetail(n: NumberStat): string {
  if (n.count === 0) return t("stats.filled", 0);
  const f = (x: number) => formatNumber(x, lang.value);
  return `${t("stats.filled", n.count)} · ${t("stats.numberDetail", {
    avg: f(n.avg),
    min: f(n.min),
    max: f(n.max),
  })}`;
}

async function load() {
  try {
    const [s] = await Promise.all([
      statsApi.collection(props.collectionId),
      metaTypes.loadItemTypes(),
    ]);
    stats.value = s;
  } catch (e) {
    notifyError(e, t("stats.loadFailed"));
  } finally {
    loading.value = false;
  }
}

onMounted(load);
watch(() => props.version, load);
</script>
