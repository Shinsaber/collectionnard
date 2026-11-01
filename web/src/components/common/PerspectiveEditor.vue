<template>
  <!--
    Redressement d'une couverture photographiée de biais (lot M4) : quatre coins posés sur la photo
    (détectés automatiquement quand le fond contraste, sinon l'image entière), déplaçables au doigt,
    à la souris ou au clavier ; « Redresser » applique l'homographie (lib/perspective) et rend un
    fichier prêt pour le recadrage. Retours v0.4.0 : zone de prise élargie autour de chaque coin,
    loupe pendant le déplacement (le doigt cache le coin), rotation par quarts de tour.
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
          :points="points"
          class="fill-surface/15 stroke-surface"
          :stroke-width="stroke"
        />
        <!-- Poignée visible, et par-dessus un disque transparent bien plus large pour la saisir. -->
        <g v-for="(p, i) in quad" :key="i" class="group">
          <circle
            :cx="p.x"
            :cy="p.y"
            :r="handleRadius"
            class="pointer-events-none fill-surface stroke-primary group-focus-within:fill-tint"
            :stroke-width="stroke"
          />
          <circle
            :cx="p.x"
            :cy="p.y"
            :r="handleRadius * HIT_FACTOR"
            class="cursor-move fill-transparent outline-none"
            role="button"
            tabindex="0"
            :aria-label="t('schema.gallery.corner', { n: i + 1 })"
            data-test="perspective-handle"
            @pointerdown="onDown(i, $event)"
            @keydown="onKey(i, $event)"
          />
        </g>
      </svg>
      <!-- Loupe : le voisinage du coin déplacé, agrandi, dans l'angle opposé au doigt. -->
      <svg
        v-if="loupe && previewUrl && size"
        :viewBox="loupe.viewBox"
        class="pointer-events-none absolute top-2 size-32 rounded-lg bg-ink shadow-cover ring-2 ring-surface"
        :class="loupe.right ? 'right-2' : 'left-2'"
        aria-hidden="true"
        data-test="perspective-loupe"
      >
        <image
          :href="previewUrl"
          :width="size.width"
          :height="size.height"
          preserveAspectRatio="none"
        />
        <polygon
          :points="points"
          class="fill-none stroke-surface"
          :stroke-width="loupe.stroke"
        />
        <line
          :x1="loupe.x - loupe.arm"
          :y1="loupe.y"
          :x2="loupe.x + loupe.arm"
          :y2="loupe.y"
          class="stroke-primary"
          :stroke-width="loupe.stroke"
        />
        <line
          :x1="loupe.x"
          :y1="loupe.y - loupe.arm"
          :x2="loupe.x"
          :y2="loupe.y + loupe.arm"
          class="stroke-primary"
          :stroke-width="loupe.stroke"
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
      <ElButton
        :aria-label="t('schema.gallery.rotateLeft')"
        :disabled="busy || !bitmap"
        data-test="perspective-rotate-left"
        @click="rotate(-1)"
      >
        <i class="pi pi-replay" aria-hidden="true"></i>
      </ElButton>
      <ElButton
        :aria-label="t('schema.gallery.rotateRight')"
        :disabled="busy || !bitmap"
        data-test="perspective-rotate-right"
        @click="rotate(1)"
      >
        <i class="pi pi-refresh" aria-hidden="true"></i>
      </ElButton>
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
  rotateQuad,
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

/** Rayon du disque transparent qui saisit un coin, en multiples de la poignée visible. */
const HIT_FACTOR = 2.6;
/** Côté de la loupe à l'écran (px) et grossissement par rapport à l'aperçu. */
const LOUPE_PX = 128;
const LOUPE_ZOOM = 3;

/** Rayon des poignées et épaisseur des traits, en pixels de l'image (constants à l'écran). */
const handleRadius = computed(() =>
  size.value ? Math.max(size.value.width, size.value.height) / 36 : 1,
);
const stroke = computed(() => handleRadius.value / 6);
const points = computed(
  () => quad.value?.map((p) => `${p.x},${p.y}`).join(" ") ?? "",
);

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

/**
 * Quart de tour (horaire si `turns` = 1) : l'image est redessinée tournée (bitmap et aperçu
 * remplacés), les coins suivent sans nouvelle détection.
 */
async function rotate(turns: 1 | -1) {
  const bmp = bitmap.value;
  const q = quad.value;
  if (!bmp || !q) return;
  busy.value = true;
  try {
    await nextFrame();
    const canvas = document.createElement("canvas");
    canvas.width = bmp.height;
    canvas.height = bmp.width;
    const ctx = canvas.getContext("2d");
    if (!ctx) throw new Error("canvas unavailable");
    ctx.translate(canvas.width / 2, canvas.height / 2);
    ctx.rotate((turns * Math.PI) / 2);
    ctx.drawImage(bmp, -bmp.width / 2, -bmp.height / 2);
    const rotated = await createImageBitmap(canvas);
    // Aperçu : une copie réduite suffit à l'affichage et à la loupe.
    const preview = await encodeImage(
      canvas,
      outputType(props.file.type),
      WARP_SOURCE_EDGE,
    );
    bmp.close();
    bitmap.value = rotated;
    if (previewUrl.value) URL.revokeObjectURL(previewUrl.value);
    previewUrl.value = URL.createObjectURL(preview);
    quad.value = rotateQuad(q, bmp.width, bmp.height, turns);
    size.value = { width: rotated.width, height: rotated.height };
  } catch (e) {
    notifyError(e, t("schema.gallery.straightenFailed"));
  } finally {
    busy.value = false;
  }
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
const dragging = ref(-1);

function onDown(i: number, e: PointerEvent) {
  dragging.value = i;
  (e.currentTarget as Element).setPointerCapture?.(e.pointerId);
  e.preventDefault();
}

function onMove(e: PointerEvent) {
  if (dragging.value < 0) return;
  moveCorner(dragging.value, toImage(e));
}

function onUp() {
  dragging.value = -1;
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

// ── Loupe ────────────────────────────────────────────────────────────
/**
 * Fenêtre de la loupe (en pixels de l'image) autour du coin déplacé, et son placement : en haut à
 * gauche de la scène, sauf si le coin s'y trouve (alors en haut à droite). Recalculée à chaque
 * déplacement puisqu'elle dépend du coin ; la taille de la scène est relue à ce moment-là.
 */
const loupe = computed(() => {
  const i = dragging.value;
  const q = quad.value;
  const el = svg.value;
  const s = size.value;
  if (i < 0 || !q || !el || !s) return null;
  const p = q[i]!;
  const rect = el.getBoundingClientRect();
  const scale = Math.min(rect.width / s.width, rect.height / s.height);
  if (!scale || !Number.isFinite(scale)) return null;
  const side = LOUPE_PX / (scale * LOUPE_ZOOM);
  const screenX = (rect.width - s.width * scale) / 2 + p.x * scale;
  const screenY = (rect.height - s.height * scale) / 2 + p.y * scale;
  const margin = LOUPE_PX + 24;
  return {
    viewBox: `${p.x - side / 2} ${p.y - side / 2} ${side} ${side}`,
    x: p.x,
    y: p.y,
    arm: side / 8,
    stroke: side / 80,
    right: screenX < margin && screenY < margin,
  };
});
</script>
