<template>
  <!--
    Visionneuse plein écran maison (retours v0.4.0) : pincer ou molette pour zoomer autour du
    point visé, double-tap pour alterner ajusté / agrandi, glisser pour se déplacer sans jamais
    sortir l'image de l'écran (lib/lightbox), balayer pour changer d'image, rotation par quarts de
    tour. Surface volontairement indépendante du thème : noir profond et commandes sombres, comme
    une table lumineuse, lisibles en mode clair comme en mode sombre.
  -->
  <Teleport to="body">
    <div
      v-if="visible"
      ref="root"
      class="fixed inset-0 z-3000 flex touch-none flex-col bg-black/90 text-white select-none"
      role="dialog"
      aria-modal="true"
      :aria-label="t('common.viewer.title')"
      tabindex="-1"
      data-test="lightbox"
      @keydown="onKey"
    >
      <div
        ref="stage"
        class="relative flex-1 overflow-hidden"
        @pointerdown="onDown"
        @pointermove="onMove"
        @pointerup="onUp"
        @pointercancel="onUp"
        @wheel.prevent="onWheel"
        @click.self="onBackdropClick"
      >
        <img
          v-if="current"
          :key="current"
          :src="current"
          :alt="counterLabel"
          class="absolute top-1/2 left-1/2 max-w-none"
          :class="{
            'transition-transform duration-200 ease-out': !gesturing,
            'opacity-0': loading,
          }"
          :style="imgStyle"
          draggable="false"
          data-test="lightbox-image"
          @load="onLoad"
          @error="loading = false"
        />
        <div
          v-if="loading"
          class="pointer-events-none absolute inset-0 grid place-items-center"
        >
          <AppSpinner />
        </div>
      </div>

      <!-- Haut : compteur et fermeture. -->
      <div
        class="pointer-events-none absolute inset-x-0 top-0 flex items-center justify-between gap-3 p-3"
      >
        <span
          v-if="urls.length > 1"
          class="rounded-full bg-black/60 px-3 py-1.5 font-mono text-xs text-white/85"
          aria-live="polite"
          data-test="lightbox-counter"
          >{{ counterLabel }}</span
        >
        <span v-else></span>
        <button
          type="button"
          class="pointer-events-auto flex size-11 items-center justify-center rounded-full bg-black/60 text-white/85 hover:bg-black/80"
          :aria-label="t('common.close')"
          data-test="lightbox-close"
          @click="close"
        >
          <i class="pi pi-times" aria-hidden="true"></i>
        </button>
      </div>

      <!-- Bas : zoom, ajustement, rotation, image précédente / suivante. -->
      <div
        class="pointer-events-none absolute inset-x-0 bottom-0 flex justify-center p-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))]"
      >
        <div
          class="pointer-events-auto flex items-center gap-1 rounded-full bg-black/60 p-1"
          role="toolbar"
          :aria-label="t('common.viewer.tools')"
        >
          <button
            v-for="tool in tools"
            :key="tool.key"
            type="button"
            class="flex size-11 items-center justify-center rounded-full text-white/85 hover:bg-black/80 disabled:opacity-40"
            :aria-label="tool.label"
            :title="tool.label"
            :disabled="tool.disabled"
            :data-test="`lightbox-${tool.key}`"
            @click="tool.run"
          >
            <i class="pi" :class="tool.icon" aria-hidden="true"></i>
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  clampOffset,
  fitSize,
  rotatedSize,
  zoomAround,
  type Offset,
  type Size,
} from "@/lib/lightbox";
import { useLightboxGestures } from "@/composables/useLightboxGestures";
import AppSpinner from "./AppSpinner.vue";

const props = withDefaults(
  defineProps<{
    /** Images à parcourir (URL d'origine, pas la vignette). */
    urls: string[];
    /** Image montrée à l'ouverture. */
    index?: number;
  }>(),
  { index: 0 },
);
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();

/** Marge autour de l'image ajustée, pour les barres. */
const PADDING = 16;
/** Zoom du double-tap et pas des boutons et de la molette. */
const TAP_ZOOM = 2.5;
const STEP = 1.4;

const root = ref<HTMLDivElement | null>(null);
const stage = ref<HTMLDivElement | null>(null);
const cursor = ref(0);
const loading = ref(true);
const natural = ref<Size | null>(null);
const viewport = ref<Size>({ width: 1, height: 1 });
const zoom = ref(1);
const rotation = ref(0);
const offset = ref<Offset>({ x: 0, y: 0 });
/** Décalage horizontal d'un balayage en cours (zoom 1), remis à zéro au relâchement. */
const swipeX = ref(0);
const gesturing = ref(false);

const current = computed(() => props.urls[cursor.value]);
const counterLabel = computed(() =>
  t("common.viewer.counter", { n: cursor.value + 1, total: props.urls.length }),
);

/** Taille de l'élément au zoom 1 (ajustée à la fenêtre et à la rotation). */
const fitted = computed(() =>
  natural.value
    ? fitSize(natural.value, viewport.value, rotation.value, PADDING)
    : null,
);
/** Empreinte tournée, pour borner le déplacement. */
const box = computed(() =>
  fitted.value ? rotatedSize(fitted.value, rotation.value) : null,
);

const imgStyle = computed(() => {
  const f = fitted.value;
  if (!f) return { width: "0px", height: "0px" };
  const x = offset.value.x + swipeX.value;
  const y = offset.value.y;
  return {
    width: `${f.width}px`,
    height: `${f.height}px`,
    transform: `translate(calc(-50% + ${x}px), calc(-50% + ${y}px)) rotate(${rotation.value}deg) scale(${zoom.value})`,
  };
});

const tools = computed(() => [
  {
    key: "zoom-out",
    icon: "pi-search-minus",
    label: t("common.viewer.zoomOut"),
    disabled: zoom.value <= 1,
    run: () => zoomBy(1 / STEP),
  },
  {
    key: "zoom-in",
    icon: "pi-search-plus",
    label: t("common.viewer.zoomIn"),
    disabled: zoom.value >= 6,
    run: () => zoomBy(STEP),
  },
  {
    key: "fit",
    icon: "pi-expand",
    label: t("common.viewer.fit"),
    disabled: zoom.value === 1 && !offset.value.x && !offset.value.y,
    run: fit,
  },
  {
    key: "rotate-left",
    icon: "pi-replay",
    label: t("common.viewer.rotateLeft"),
    disabled: false,
    run: () => rotate(-90),
  },
  {
    key: "rotate-right",
    icon: "pi-refresh",
    label: t("common.viewer.rotateRight"),
    disabled: false,
    run: () => rotate(90),
  },
  {
    key: "prev",
    icon: "pi-chevron-left",
    label: t("common.viewer.previous"),
    disabled: props.urls.length < 2,
    run: () => step(-1),
  },
  {
    key: "next",
    icon: "pi-chevron-right",
    label: t("common.viewer.next"),
    disabled: props.urls.length < 2,
    run: () => step(1),
  },
]);

// ── Ouverture / fermeture ─────────────────────────────────────────────
watch(visible, async (open) => {
  if (open) {
    cursor.value = Math.min(Math.max(0, props.index), props.urls.length - 1);
    resetView();
    document.documentElement.style.overflow = "hidden";
    window.addEventListener("resize", measure);
    await nextTick();
    measure();
    root.value?.focus();
  } else {
    document.documentElement.style.overflow = "";
    window.removeEventListener("resize", measure);
  }
});
onBeforeUnmount(() => {
  document.documentElement.style.overflow = "";
  window.removeEventListener("resize", measure);
});

function close() {
  visible.value = false;
}

function measure() {
  const r = stage.value?.getBoundingClientRect();
  if (r && r.width && r.height)
    viewport.value = { width: r.width, height: r.height };
  settle();
}

function onLoad(e: Event) {
  const img = e.target as HTMLImageElement;
  natural.value = { width: img.naturalWidth, height: img.naturalHeight };
  loading.value = false;
  measure();
}

function resetView() {
  loading.value = true;
  natural.value = null;
  zoom.value = 1;
  rotation.value = 0;
  offset.value = { x: 0, y: 0 };
  swipeX.value = 0;
}

/** Image voisine (en boucle) ; la vue repart ajustée. */
function step(delta: number) {
  const n = props.urls.length;
  if (n < 2) return;
  cursor.value = (cursor.value + delta + n) % n;
  resetView();
}

// ── Vue ──────────────────────────────────────────────────────────────
/** Remet le décalage dans les bornes après tout changement de zoom, de rotation ou de fenêtre. */
function settle() {
  if (box.value)
    offset.value = clampOffset(
      offset.value,
      box.value,
      zoom.value,
      viewport.value,
    );
}

function setZoom(next: number, point: Offset) {
  const r = zoomAround(offset.value, zoom.value, next, point);
  zoom.value = r.zoom;
  offset.value = r.offset;
  settle();
}

function zoomBy(factor: number) {
  setZoom(zoom.value * factor, { x: 0, y: 0 });
}

function fit() {
  zoom.value = 1;
  offset.value = { x: 0, y: 0 };
}

function rotate(deg: number) {
  rotation.value = (((rotation.value + deg) % 360) + 360) % 360;
  settle();
}

/** Double-tap : agrandi autour du point, ou retour à l'ajustement. */
function toggleZoom(point: Offset) {
  if (zoom.value > 1) fit();
  else setZoom(TAP_ZOOM, point);
}

// ── Gestes (composables/useLightboxGestures) ─────────────────────────
const { toStage, onDown, onMove, onUp, onBackdropClick } = useLightboxGestures({
  stage,
  zoom,
  offset,
  swipeX,
  gesturing,
  count: () => props.urls.length,
  settle,
  step,
  toggleZoom,
  close,
});

function onWheel(e: WheelEvent) {
  if (!stage.value) return;
  setZoom(zoom.value * (e.deltaY < 0 ? STEP : 1 / STEP), toStage(e));
}

function onKey(e: KeyboardEvent) {
  const actions: Record<string, () => void> = {
    Escape: close,
    ArrowLeft: () => step(-1),
    ArrowRight: () => step(1),
    "+": () => zoomBy(STEP),
    "-": () => zoomBy(1 / STEP),
    "0": fit,
  };
  const run = actions[e.key];
  if (!run) return;
  e.preventDefault();
  run();
}
</script>
