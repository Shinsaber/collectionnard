<template>
  <div
    ref="el"
    class="w-full"
    role="img"
    :aria-label="summary"
    :style="{ height: `${height}px` }"
  ></div>
</template>

<script setup lang="ts">
// Graphique ECharts (barres, courbe ou anneau) ; couleurs des jetons « Étiquette ».
// Chargement à la carte : seuls les modules utiles entrent dans le paquet (vue paresseuse).
import { BarChart, LineChart, PieChart } from "echarts/charts";
import { GridComponent, TooltipComponent } from "echarts/components";
import * as echarts from "echarts/core";
import { SVGRenderer } from "echarts/renderers";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { useLang } from "@/composables/useLang";
import { useTheme } from "@/composables/useTheme";

echarts.use([
  BarChart,
  LineChart,
  PieChart,
  GridComponent,
  TooltipComponent,
  SVGRenderer,
]);

const props = withDefaults(
  defineProps<{
    kind: "bar" | "line" | "pie";
    labels: string[];
    values: number[];
    /** Barres horizontales (libellés longs : types, collections). */
    horizontal?: boolean;
    height?: number;
    /** Nom de la série, repris dans le résumé lu par les lecteurs d'écran. */
    title: string;
  }>(),
  { horizontal: false, height: 240 },
);

const { lang } = useLang();
const { theme } = useTheme();
const el = ref<HTMLDivElement | null>(null);
let chart: echarts.ECharts | null = null;
let observer: ResizeObserver | null = null;

/** Alternative textuelle : « Titre : A 3, B 1 » (les 12 premières valeurs). */
const summary = computed(() => {
  const parts = props.labels
    .slice(0, 12)
    .map((l, i) => `${l} ${fmt(props.values[i] ?? 0)}`);
  return `${props.title} : ${parts.join(", ")}`;
});

function fmt(n: number): string {
  return n.toLocaleString(lang.value, { maximumFractionDigits: 2 });
}

/** Jetons de couleur courants (`main.css`, redéfinis sous `html.dark`). */
function tokens() {
  const css = getComputedStyle(document.documentElement);
  const v = (name: string, fallback: string) =>
    css.getPropertyValue(name).trim() || fallback;
  return {
    ink: v("--c-ink", "#17202b"),
    muted: v("--c-muted", "#586270"),
    line: v("--c-line", "#e3e6e0"),
    surface: v("--c-surface", "#ffffff"),
    palette: [
      v("--c-primary", "#1d5c63"),
      v("--c-accent", "#e09a2c"),
      v("--c-tint-ink", "#123e43"),
      v("--c-muted", "#586270"),
      v("--c-dash", "#c9cec6"),
      v("--c-danger", "#a2342b"),
    ],
  };
}

function option(): echarts.EChartsCoreOption {
  const c = tokens();
  const text = { color: c.muted, fontFamily: "IBM Plex Mono", fontSize: 11 };
  const tooltip = {
    trigger: props.kind === "pie" ? "item" : "axis",
    valueFormatter: (v: unknown) => fmt(Number(v)),
    backgroundColor: c.surface,
    borderColor: c.line,
    textStyle: { color: c.ink },
  };
  if (props.kind === "pie") {
    return {
      color: c.palette,
      tooltip,
      series: [
        {
          type: "pie",
          name: props.title,
          radius: ["45%", "72%"],
          itemStyle: { borderColor: c.surface, borderWidth: 2 },
          label: { color: c.ink, fontSize: 12 },
          data: props.labels.map((name, i) => ({
            name,
            value: props.values[i] ?? 0,
          })),
        },
      ],
    };
  }
  const category = {
    type: "category",
    data: props.labels,
    axisLabel: { ...text, hideOverlap: true },
    axisLine: { lineStyle: { color: c.line } },
    axisTick: { show: false },
  };
  const value = {
    type: "value",
    minInterval: 1,
    axisLabel: text,
    splitLine: { lineStyle: { color: c.line } },
  };
  return {
    color: c.palette,
    tooltip,
    grid: { left: 8, right: 16, top: 16, bottom: 8 },
    xAxis: props.horizontal ? value : category,
    yAxis: props.horizontal ? { ...category, inverse: true } : value,
    series: [
      {
        type: props.kind,
        name: props.title,
        data: props.values,
        barMaxWidth: 28,
        itemStyle: {
          borderRadius: props.horizontal ? [0, 4, 4, 0] : [4, 4, 0, 0],
        },
        smooth: props.kind === "line",
        showSymbol: props.values.length <= 24,
        areaStyle: props.kind === "line" ? { opacity: 0.12 } : undefined,
      },
    ],
  };
}

function render() {
  chart?.setOption(option(), true);
}

onMounted(() => {
  if (!el.value) return;
  chart = echarts.init(el.value, null, { renderer: "svg" });
  render();
  observer = new ResizeObserver(() => chart?.resize());
  observer.observe(el.value);
});

onBeforeUnmount(() => {
  observer?.disconnect();
  chart?.dispose();
  chart = null;
});

watch(() => [props.kind, props.labels, props.values, lang.value], render, {
  deep: true,
});
// La classe `dark` est posée par useTheme : relire les jetons une fois le DOM à jour.
watch(theme, () => nextTick(render));
</script>
