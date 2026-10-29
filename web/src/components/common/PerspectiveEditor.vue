<template>
  <!--
    Redressement d'une couverture photographiée de biais (lot M4) : quatre coins posés sur la photo
    (détectés automatiquement quand le fond contraste, sinon l'image entière), déplaçables au doigt,
    à la souris ou au clavier ; « Redresser » applique l'homographie (lib/perspective) et rend un
    fichier prêt pour le recadrage.
  -->
  <div class="flex flex-col gap-3">
    <p class="text-sm text-muted">{{ t("schema.gallery.straightenHint") }}</p>
    <div
      class="relative h-[55vh] max-h-120 w-full overflow-hidden rounded-lg bg-ink shadow-cover"
      data-test="perspective-stage"
    >
      <img
        v-if="previewUrl"
        :src="previewUrl"
        alt=""
        class="absolute inset-0 h-full w-full object-contain"
      />
      <svg
        v-if="size && quad"
        ref="svg"
        :viewBox="`0 0 ${size.width} ${size.height}`"
        preserveAspectRatio="xMidYMid meet"
        class="absolute inset-0 h-full w-full touch-none select-none"
        @pointermove="onMove"
        @pointerup="onUp"
        @pointercancel="onUp"
      >
        <polygon
          :points="quad.map((p) => `${p.x},${p.y}`).join(' ')"
          class="fill-surface/15 stroke-surface"
          :stroke-width="stroke"
        />
        <circle
          v-for="(p, i) in quad"
          :key="i"
          :cx="p.x"
          :cy="p.y"
          :r="handleRadius"
          class="cursor-move fill-surface stroke-primary outline-none focus-visible:fill-tint"
          :stroke-width="stroke"
          role="button"
          tabindex="0"
          :aria-label="t('schema.gallery.corner', { n: i + 1 })"
          @pointerdown="onDown(i, $event)"
          @keydown="onKey(i, $event)"
        />
      </svg>
      <div v-if="busy" class="absolute inset-0 grid place-items-center">
        <AppSpinner />
      </div>
    </div>
    <p
      v-if="message"
      class="text-xs text-muted"
      role="status"
      data-test="perspective-message"
    >
      {{ message }}
    </p>
    <div class="flex flex-wrap items-center gap-2">
      <ElButton :disabled="busy || !bitmap" @click="detect">
        <i class="pi pi-objects-column mr-2" aria-hidden="true"></i>
        {{ t("schema.gallery.detectEdges") }}
      </ElButton>
      <ElButton :disabled="busy || !size" @click="reset">
        {{ t("schema.gallery.wholeImage") }}
      </ElButton>
      <span class="flex-1"></span>
      <ElButton :disabled="busy" @click="emit('skip')">
        {{ t("schema.gallery.skipStraighten") }}
      </ElButton>
      <ElButton
        type="primary"
        :loading="busy"
        :disabled="!size || !quad"
        data-test="perspective-apply"
        @click="apply"
      >
        <i v-if="!busy" class="pi pi-check mr-2" aria-hidden="true"></i>
        {{ t("schema.gallery.straighten") }}
      </ElButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useNotify } from "@/composables/useNotify";
import {
  MAX_EDGE,
  WARP_SOURCE_EDGE,
  canvasOf,
  encodeImage,
  outputName,
  outputType,
  rasterOf,
} from "@/lib/images";
import {
  DETECT_EDGE,
  detectQuad,
  fullQuad,
  quadOutputSize,
  scaleQuad,
  warpPerspective,
  type Point,
  type Quad,
} from "@/lib/perspective";
import AppSpinner from "./AppSpinner.vue";

const props = defineProps<{ file: File }>();
const emit = defineEmits<{ done: [file: File]; skip: [] }>();

const { t } = useI18n();
const { notifyError } = useNotify();

const svg = ref<SVGSVGElement | null>(null);
const previewUrl = ref("");
const size = ref<{ width: number; height: number } | null>(null);
const quad = ref<Quad | null>(null);
const busy = ref(false);
const message = ref("");
const bitmap = ref<ImageBitmap | null>(null);

/** Rayon des poignées et épaisseur des traits, en pixels de l'image (constants à l'écran). */
const handleRadius = computed(() =>
  size.value ? Math.max(size.value.width, size.value.height) / 40 : 1,
);
const stroke = computed(() => handleRadius.value / 6);

onMounted(load);
watch(() => props.file, load);
onBeforeUnmount(cleanup);

async function load() {
  cleanup();
  previewUrl.value = URL.createObjectURL(props.file);
  try {
    bitmap.value = await createImageBitmap(props.file);
  } catch (e) {
    notifyError(e, t("schema.gallery.straightenFailed"));
    return;
  }
  size.value = { width: bitmap.value.width, height: bitmap.value.height };
  await detect();
}

function cleanup() {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value);
  previewUrl.value = "";
  bitmap.value?.close();
  bitmap.value = null;
  size.value = null;
  quad.value = null;
  message.value = "";
}

/** Bords cherchés sur une petite copie ; sans résultat, l'image entière. */
async function detect() {
  const bmp = bitmap.value;
  if (!bmp) return;
  busy.value = true;
  try {
    await nextFrame();
    const small = rasterOf(bmp, DETECT_EDGE);
    const found = detectQuad(small);
    if (found) {
      quad.value = scaleQuad(
        found.quad,
        bmp.width / small.width,
        bmp.height / small.height,
      );
      message.value =
        found.confidence < 0.9 ? t("schema.gallery.roughEdges") : "";
    } else {
      reset();
      message.value = t("schema.gallery.noEdges");
    }
  } catch (e) {
    reset();
    notifyError(e, t("schema.gallery.straightenFailed"));
  } finally {
    busy.value = false;
  }
}

function reset() {
  if (size.value) quad.value = fullQuad(size.value.width, size.value.height);
  message.value = "";
}

/** Homographie du quadrilatère vers un rectangle, à la résolution de sortie, puis encodage. */
async function apply() {
  const bmp = bitmap.value;
  const q = quad.value;
  if (!bmp || !q) return;
  busy.value = true;
  try {
    await nextFrame();
    const src = rasterOf(bmp, WARP_SOURCE_EDGE);
    const scale = src.width / bmp.width;
    const scaled = scaleQuad(q, scale, scale);
    const out = quadOutputSize(scaled, MAX_EDGE);
    const warped = warpPerspective(src, scaled, out.width, out.height);
    const type = outputType(props.file.type);
    const blob = await encodeImage(canvasOf(warped), type);
    emit("done", new File([blob], outputName(props.file.name, type), { type }));
  } catch (e) {
    notifyError(e, t("schema.gallery.straightenFailed"));
  } finally {
    busy.value = false;
  }
}

/** Laisse le navigateur peindre l'indicateur avant un calcul qui bloque le fil principal. */
function nextFrame(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

// ── Déplacement des coins ────────────────────────────────────────────
let dragging = -1;

function onDown(i: number, e: PointerEvent) {
  dragging = i;
  (e.currentTarget as Element).setPointerCapture?.(e.pointerId);
  e.preventDefault();
}

function onMove(e: PointerEvent) {
  if (dragging < 0) return;
  moveCorner(dragging, toImage(e));
}

function onUp() {
  dragging = -1;
}

/** Flèches : un pour cent de l'image par appui (dix avec Maj). */
function onKey(i: number, e: KeyboardEvent) {
  const p = quad.value?.[i];
  if (!p || !size.value) return;
  const step =
    (Math.max(size.value.width, size.value.height) / 100) *
    (e.shiftKey ? 10 : 1);
  const delta: Record<string, Point> = {
    ArrowLeft: { x: -step, y: 0 },
    ArrowRight: { x: step, y: 0 },
    ArrowUp: { x: 0, y: -step },
    ArrowDown: { x: 0, y: step },
  };
  const d = delta[e.key];
  if (!d) return;
  e.preventDefault();
  moveCorner(i, { x: p.x + d.x, y: p.y + d.y });
}

function moveCorner(i: number, p: Point) {
  if (!quad.value || !size.value) return;
  const next = [...quad.value] as Quad;
  next[i] = {
    x: Math.min(size.value.width, Math.max(0, p.x)),
    y: Math.min(size.value.height, Math.max(0, p.y)),
  };
  quad.value = next;
}

/** Coordonnées écran → repère de l'image (viewBox du SVG). */
function toImage(e: PointerEvent): Point {
  const el = svg.value;
  const ctm = el?.getScreenCTM();
  if (!el || !ctm) return { x: 0, y: 0 };
  const pt = new DOMPoint(e.clientX, e.clientY).matrixTransform(ctm.inverse());
  return { x: pt.x, y: pt.y };
}
</script>
