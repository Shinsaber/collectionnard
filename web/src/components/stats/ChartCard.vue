<template>
  <section
    class="flex min-w-0 flex-col gap-3 rounded-2xl bg-surface p-5 shadow-card"
  >
    <h2 class="text-base font-extrabold">{{ title }}</h2>
    <StatsChart
      :kind="kind"
      :title="title"
      :labels="series.labels"
      :values="series.values"
      :horizontal="horizontal"
      :height="height"
    />
  </section>
</template>

<script setup lang="ts">
// Carte titrée autour d'un graphique (statistiques).
import { computed } from "vue";
import StatsChart from "./StatsChart.vue";
import type { Series } from "@/lib/stats";

const props = defineProps<{
  title: string;
  kind: "bar" | "line" | "pie";
  series: Series;
  horizontal?: boolean;
}>();

/** Barres horizontales : hauteur selon le nombre de barres (28 px chacune). */
const height = computed(() =>
  props.horizontal
    ? Math.min(560, Math.max(160, props.series.labels.length * 28 + 40))
    : 240,
);
</script>
