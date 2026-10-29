<template>
  <!--
    Recadrage d'une image avant son envoi (lot M1) : cadre au ratio de la catégorie, image
    déplaçable et zoomable (doigts ou molette), rotation par quarts de tour. Rendu par cropperjs 2
    (MIT, composants web) ; la sortie est encodée et réduite par lib/images.
    Étape préalable « Redresser » (lot M4, PerspectiveEditor) : d'office pour une photo, sur demande
    pour un fichier ; le recadrage porte alors sur l'image redressée.
  -->
  <ElDialog
    v-model="visible"
    :title="
      t(
        step === 'straighten'
          ? 'schema.gallery.straightenTitle'
          : 'schema.gallery.cropTitle',
        { category: category?.name ?? '' },
      )
    "
    width="40rem"
    append-to-body
    @opened="setup"
    @closed="teardown"
  >
    <p v-if="reason" class="mb-3 text-sm text-muted" data-test="crop-reason">
      {{ reason }}
    </p>
    <PerspectiveEditor
      v-if="step === 'straighten' && file"
      :file="file"
      @done="onStraightened"
      @skip="toCrop(file)"
    />
    <div v-show="step === 'crop'">
      <div
        ref="container"
        class="h-[60vh] max-h-128 overflow-hidden rounded-lg bg-ink shadow-cover"
        data-test="crop-container"
      ></div>
      <div class="mt-3 flex flex-wrap items-center gap-2">
        <ElButton
          :aria-label="t('schema.gallery.rotateLeft')"
          :disabled="!ready"
          @click="rotate(-90)"
        >
          <i class="pi pi-replay" aria-hidden="true"></i>
        </ElButton>
        <ElButton
          :aria-label="t('schema.gallery.rotateRight')"
          :disabled="!ready"
          @click="rotate(90)"
        >
          <i class="pi pi-refresh" aria-hidden="true"></i>
        </ElButton>
        <ElButton :disabled="!ready" @click="fit">
          <i class="pi pi-expand mr-2" aria-hidden="true"></i>
          {{ t("schema.gallery.fitImage") }}
        </ElButton>
        <ElButton
          :disabled="!ready"
          data-test="crop-straighten"
          @click="toStraighten"
        >
          <i class="pi pi-objects-column mr-2" aria-hidden="true"></i>
          {{ t("schema.gallery.straightenStep") }}
        </ElButton>
        <span class="ml-auto text-xs text-muted">
          {{
            aspect
              ? t("schema.gallery.cropHint", { ratio: category?.ratio })
              : t("schema.gallery.cropHintFree")
          }}
        </span>
      </div>
    </div>
    <template #footer>
      <ElButton text @click="visible = false">{{
        t("common.cancel")
      }}</ElButton>
      <template v-if="step === 'crop'">
        <ElButton :loading="busy" @click="asIs">
          {{ t("schema.gallery.asIs") }}
        </ElButton>
        <ElButton
          type="primary"
          :loading="busy"
          :disabled="!ready"
          @click="apply"
        >
          <i v-if="!busy" class="pi pi-check mr-2" aria-hidden="true"></i>
          {{ t("schema.gallery.applyCrop") }}
        </ElButton>
      </template>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import Cropper from "cropperjs";
import { useNotify } from "@/composables/useNotify";
import { parseRatio, type ImageCategory } from "@/lib/schema";
import {
  encodeImage,
  outputName,
  outputType,
  shrinkIfNeeded,
} from "@/lib/images";
import PerspectiveEditor from "./PerspectiveEditor.vue";

const props = defineProps<{
  /** Image à recadrer (fichier choisi ou photo prise). */
  file: File | null;
  /** Catégorie visée : son ratio contraint le cadre (libre sans ratio). */
  category: ImageCategory | null;
  /** Pourquoi le recadrage est proposé (image hors format…), affiché au-dessus. */
  reason?: string;
  /** Commencer par le redressement de la perspective (photo prise à l'instant). */
  straighten?: boolean;
}>();
const emit = defineEmits<{ done: [file: File] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { notifyError } = useNotify();

const container = ref<HTMLDivElement | null>(null);
const step = ref<"straighten" | "crop">("crop");
/** Image recadrée : l'originale, ou son redressement. */
const workFile = ref<File | null>(null);
const ready = ref(false);
const busy = ref(false);

/** Ratio numérique du cadre ; NaN = libre. */
const aspect = computed(() => {
  const r = props.category?.ratio ? parseRatio(props.category.ratio) : NaN;
  return Number.isFinite(r) && r > 0 ? r : NaN;
});

// Gabarit : pas de création de cadre par glissement sur le fond (le fond déplace l'image), un seul
// cadre redimensionnable par ses huit poignées, grille et croisée pour viser.
const TEMPLATE = `
<cropper-canvas background scale-step="0.1" style="height:100%">
  <cropper-image rotatable scalable translatable></cropper-image>
  <cropper-shade hidden></cropper-shade>
  <cropper-handle action="move" plain></cropper-handle>
  <cropper-selection movable resizable zoomable outlined>
    <cropper-grid role="grid" bordered covered></cropper-grid>
    <cropper-crosshair centered></cropper-crosshair>
    <cropper-handle action="move" theme-color="rgba(255, 255, 255, 0.35)"></cropper-handle>
    <cropper-handle action="n-resize"></cropper-handle>
    <cropper-handle action="e-resize"></cropper-handle>
    <cropper-handle action="s-resize"></cropper-handle>
    <cropper-handle action="w-resize"></cropper-handle>
    <cropper-handle action="ne-resize"></cropper-handle>
    <cropper-handle action="nw-resize"></cropper-handle>
    <cropper-handle action="se-resize"></cropper-handle>
    <cropper-handle action="sw-resize"></cropper-handle>
  </cropper-selection>
</cropper-canvas>`;

let cropper: Cropper | null = null;
let objectUrl = "";

function setup() {
  if (!props.file) return;
  if (props.straighten) {
    step.value = "straighten";
    return;
  }
  toCrop(props.file);
}

function toCrop(file: File) {
  workFile.value = file;
  step.value = "crop";
  void nextTick(mountCropper);
}

function onStraightened(file: File) {
  toCrop(file);
}

/** Retour au redressement : depuis la photo d'origine. */
function toStraighten() {
  unmountCropper();
  step.value = "straighten";
}

async function mountCropper() {
  const host = container.value;
  const file = workFile.value;
  if (!host || !file) return;
  unmountCropper();
  objectUrl = URL.createObjectURL(file);
  const img = document.createElement("img");
  img.src = objectUrl;
  img.alt = props.category?.name ?? "";
  cropper = new Cropper(img, { container: host, template: TEMPLATE });
  const image = cropper.getCropperImage();
  const selection = cropper.getCropperSelection();
  if (!image || !selection) return;
  selection.aspectRatio = aspect.value;
  try {
    await image.$ready();
    fit();
    ready.value = true;
  } catch (e) {
    notifyError(e, t("schema.gallery.cropFailed"));
  }
}

function unmountCropper() {
  ready.value = false;
  cropper = null;
  if (container.value) container.value.replaceChildren();
  if (objectUrl) URL.revokeObjectURL(objectUrl);
  objectUrl = "";
}

function teardown() {
  unmountCropper();
  workFile.value = null;
  step.value = "crop";
}

/** Image entière visible, cadre au ratio sur la plus grande zone de l'image. */
function fit() {
  const image = cropper?.getCropperImage();
  const canvas = cropper?.getCropperCanvas();
  const selection = cropper?.getCropperSelection();
  if (!image || !canvas || !selection) return;
  image.$center("contain");
  const c = canvas.getBoundingClientRect();
  const r = image.getBoundingClientRect();
  let x = r.left - c.left;
  let y = r.top - c.top;
  let w = r.width;
  let h = r.height;
  const ratio = aspect.value;
  if (ratio && w > 0 && h > 0) {
    if (w / h > ratio) {
      const nw = h * ratio;
      x += (w - nw) / 2;
      w = nw;
    } else {
      const nh = w / ratio;
      y += (h - nh) / 2;
      h = nh;
    }
  }
  selection.$change(x, y, w, h);
}

function rotate(deg: number) {
  cropper?.getCropperImage()?.$rotate(`${deg}deg`);
  fit();
}

/** Zone du cadre, en pixels de l'image recadrée, réduite et encodée. */
async function apply() {
  const selection = cropper?.getCropperSelection();
  const file = workFile.value;
  if (!selection || !file) return;
  busy.value = true;
  try {
    const canvas = await selection.$toCanvas();
    const type = outputType(file.type);
    const blob = await encodeImage(canvas, type);
    emit("done", new File([blob], outputName(file.name, type), { type }));
    visible.value = false;
  } catch (e) {
    notifyError(e, t("schema.gallery.cropFailed"));
  } finally {
    busy.value = false;
  }
}

/** Sans recadrage : l'image (redressée ou non) est seulement réduite si elle est très grande. */
async function asIs() {
  const file = workFile.value;
  if (!file) return;
  busy.value = true;
  try {
    emit("done", await shrinkIfNeeded(file));
    visible.value = false;
  } catch (e) {
    notifyError(e, t("schema.gallery.cropFailed"));
  } finally {
    busy.value = false;
  }
}
</script>
