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
  clampZoom,
  distance,
  fitSize,
  midpoint,
  rotatedSize,
  swipeDirection,
  zoomAround,
  type Offset,
  type Size,
} from "@/lib/lightbox";
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
/** Deux appuis en moins de 300 ms et à moins de 30 px : double-tap. */
const DOUBLE_TAP_MS = 300;
const DOUBLE_TAP_PX = 30;

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

// ── Gestes ───────────────────────────────────────────────────────────
interface Gesture {
  /** Décalage et zoom au début du geste. */
  offset: Offset;
  zoom: number;
  /** Un doigt : point de départ ; deux doigts : écart et centre de départ. */
  start: Offset;
  dist: number;
  moved: boolean;
  pinched: boolean;
}

const pointers = new Map<number, Offset>();
let gesture: Gesture | null = null;
let lastTap: { at: number; point: Offset } | null = null;
/** Un glissement vient de finir : le clic qui suit ne ferme pas. */
let justDragged = false;

/** Coordonnées d'un pointeur relatives au centre de la scène. */
function toStage(e: PointerEvent): Offset {
  const r = stage.value?.getBoundingClientRect();
  if (!r) return { x: 0, y: 0 };
  return {
    x: e.clientX - (r.left + r.width / 2),
    y: e.clientY - (r.top + r.height / 2),
  };
}

function beginGesture() {
  const pts = [...pointers.values()];
  if (!pts.length) return;
  const two = pts.length >= 2;
  gesture = {
    offset: offset.value,
    zoom: zoom.value,
    start: two ? midpoint(pts[0]!, pts[1]!) : pts[0]!,
    dist: two ? distance(pts[0]!, pts[1]!) : 0,
    moved: gesture?.moved ?? false,
    pinched: two || (gesture?.pinched ?? false),
  };
  gesturing.value = true;
}

function onDown(e: PointerEvent) {
  if (e.button !== 0 && e.pointerType === "mouse") return;
  stage.value?.setPointerCapture(e.pointerId);
  pointers.set(e.pointerId, toStage(e));
  swipeX.value = 0;
  beginGesture();
}

function onMove(e: PointerEvent) {
  if (!pointers.has(e.pointerId) || !gesture) return;
  const p = toStage(e);
  pointers.set(e.pointerId, p);
  const pts = [...pointers.values()];
  if (pts.length >= 2) {
    // Pincement : zoom autour du centre de départ, puis le centre suit les doigts.
    const d = distance(pts[0]!, pts[1]!);
    const c = midpoint(pts[0]!, pts[1]!);
    const r = zoomAround(
      gesture.offset,
      gesture.zoom,
      clampZoom((gesture.zoom * d) / Math.max(1, gesture.dist)),
      gesture.start,
    );
    zoom.value = r.zoom;
    offset.value = {
      x: r.offset.x + (c.x - gesture.start.x),
      y: r.offset.y + (c.y - gesture.start.y),
    };
    settle();
    gesture.moved = true;
    return;
  }
  const dx = p.x - gesture.start.x;
  const dy = p.y - gesture.start.y;
  if (Math.hypot(dx, dy) > 6) gesture.moved = true;
  if (zoom.value > 1) {
    offset.value = { x: gesture.offset.x + dx, y: gesture.offset.y + dy };
    settle();
  } else if (props.urls.length > 1 && Math.abs(dx) > Math.abs(dy)) {
    swipeX.value = dx;
  }
}

function onUp(e: PointerEvent) {
  if (!pointers.has(e.pointerId)) return;
  const p = toStage(e);
  pointers.delete(e.pointerId);
  const g = gesture;
  if (pointers.size > 0) {
    // Un doigt reste après un pincement : il reprend un déplacement sans à-coup.
    beginGesture();
    return;
  }
  gesture = null;
  gesturing.value = false;
  if (!g) return;
  justDragged = g.moved;
  if (!g.pinched && zoom.value <= 1 && swipeX.value) {
    const dir = swipeDirection(swipeX.value, p.y - g.start.y);
    swipeX.value = 0;
    if (dir) {
      step(-dir);
      return;
    }
  }
  if (!g.moved && !g.pinched) {
    const now = performance.now();
    if (
      lastTap &&
      now - lastTap.at < DOUBLE_TAP_MS &&
      distance(lastTap.point, p) < DOUBLE_TAP_PX
    ) {
      lastTap = null;
      toggleZoom(p);
    } else {
      lastTap = { at: now, point: p };
    }
  }
}

function onBackdropClick() {
  if (justDragged) {
    justDragged = false;
    return;
  }
  close();
}

function onWheel(e: WheelEvent) {
  const r = stage.value?.getBoundingClientRect();
  if (!r) return;
  const point = {
    x: e.clientX - (r.left + r.width / 2),
    y: e.clientY - (r.top + r.height / 2),
  };
  setZoom(zoom.value * (e.deltaY < 0 ? STEP : 1 / STEP), point);
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
