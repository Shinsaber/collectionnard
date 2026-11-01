<template>
  <!--
    Scanner de codes-barres (lot M2) : caméra arrière (composable useCamera), détection en continu
    (lib/barcode : natif ou ponyfill), lampe torche si la piste l'offre ; sans caméra ou accès
    refusé, lecture dans une photo. Émet `detected` et se ferme ; en mode `continuous`, reste
    ouvert et émet chaque code.
  -->
  <ElDialog
    v-model="visible"
    :title="t('common.scan.title')"
    width="30rem"
    append-to-body
    @opened="start"
    @closed="stop"
  >
    <div
      class="relative overflow-hidden rounded-lg bg-ink shadow-cover"
      style="aspect-ratio: 4 / 3"
    >
      <video
        ref="video"
        class="h-full w-full object-cover"
        playsinline
        muted
        autoplay
        :aria-label="t('common.scan.preview')"
      ></video>
      <!-- Viseur : simple repère, la détection porte sur toute l'image. -->
      <div
        v-if="camera.state.value === 'ready'"
        class="pointer-events-none absolute inset-x-[12%] inset-y-[32%] rounded-md border-2 border-surface/80"
        aria-hidden="true"
      ></div>
      <div
        v-if="camera.state.value === 'starting'"
        class="absolute inset-0 grid place-items-center"
      >
        <AppSpinner />
      </div>
      <p
        v-else-if="camera.state.value === 'unavailable'"
        class="absolute inset-0 grid place-items-center p-6 text-center text-sm text-surface"
        role="status"
        data-test="scan-message"
      >
        {{ message }}
      </p>
    </div>
    <p
      v-if="lastCode"
      class="mt-3 font-mono text-sm text-ink"
      aria-live="polite"
    >
      {{ t("common.scan.detected", { code: lastCode }) }}
    </p>
    <!-- Résultat du dernier code (page « Scanner ») sans quitter la caméra. -->
    <slot />
    <p class="mt-3 text-sm text-muted">{{ t("common.scan.hint") }}</p>
    <div class="mt-3 flex flex-wrap items-center gap-2">
      <ElButton
        v-if="camera.torchAvailable.value"
        :aria-pressed="camera.torchOn.value"
        @click="toggleTorch"
      >
        <i class="pi pi-lightbulb mr-2" aria-hidden="true"></i>
        {{
          camera.torchOn.value
            ? t("common.scan.torchOff")
            : t("common.scan.torchOn")
        }}
      </ElButton>
      <input
        ref="photoInput"
        type="file"
        accept="image/*"
        capture="environment"
        class="hidden"
        :aria-label="t('common.scan.fromPhoto')"
        @change="fromPhoto"
      />
      <ElButton :loading="busy" @click="photoInput?.click()">
        <i v-if="!busy" class="pi pi-camera mr-2" aria-hidden="true"></i>
        {{ t("common.scan.fromPhoto") }}
      </ElButton>
    </div>
    <template #footer>
      <ElButton text @click="visible = false">{{ t("common.close") }}</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useNotify } from "@/composables/useNotify";
import { useCamera } from "@/composables/useCamera";
import {
  createDetector,
  ScanConfirmer,
  type BarcodeReader,
} from "@/lib/barcode";
import AppSpinner from "./AppSpinner.vue";

const props = defineProps<{
  /** Reste ouvert et émet chaque nouveau code (page « Scanner », M3). */
  continuous?: boolean;
}>();
const emit = defineEmits<{ detected: [code: string, format: string] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { notifyError, notifyWarning } = useNotify();

const video = ref<HTMLVideoElement | null>(null);
const photoInput = ref<HTMLInputElement | null>(null);
const camera = useCamera(video);
const message = ref("");
const busy = ref(false);
const lastCode = ref("");

/** Intervalle minimal entre deux détections : le wasm n'a pas besoin de chaque image. */
const MIN_INTERVAL_MS = 120;

let detector: BarcodeReader | null = null;
let stopped = true;
let timer = 0;
let lastTick = 0;
let failures = 0;
const confirmer = new ScanConfirmer();

async function start() {
  stopped = false;
  failures = 0;
  lastCode.value = "";
  confirmer.reset();
  try {
    detector = await createDetector();
  } catch (e) {
    camera.state.value = "unavailable";
    message.value = t("common.scan.unsupported");
    notifyError(e, t("common.scan.failed"));
    return;
  }
  const failure = await camera.start(1280, 720);
  if (failure) {
    message.value = t(
      failure.reason === "denied"
        ? "common.scan.denied"
        : "common.scan.noCamera",
    );
    if (failure.error) notifyError(failure.error, t("common.scan.failed"));
    return;
  }
  schedule();
}

function schedule() {
  const v = video.value;
  if (!v || stopped) return;
  const withFrames = v as HTMLVideoElement & {
    requestVideoFrameCallback?: (cb: () => void) => number;
  };
  if (withFrames.requestVideoFrameCallback) {
    withFrames.requestVideoFrameCallback(() => void tick());
  } else {
    timer = window.setTimeout(() => void tick(), MIN_INTERVAL_MS);
  }
}

async function tick() {
  const v = video.value;
  if (!v || stopped || !detector) return;
  const now = performance.now();
  if (v.readyState >= 2 && now - lastTick >= MIN_INTERVAL_MS) {
    lastTick = now;
    try {
      const codes = await detector.detect(v);
      if (stopped) return;
      for (const c of codes) {
        if (confirmer.push(c.rawValue, c.format)) {
          onCode(c.rawValue.trim(), c.format);
          if (!props.continuous) return;
          break;
        }
      }
    } catch (e) {
      if (++failures > 5) {
        camera.stop();
        camera.state.value = "unavailable";
        message.value = t("common.scan.failed");
        notifyError(e, t("common.scan.failed"));
        return;
      }
    }
  }
  schedule();
}

function onCode(code: string, format: string) {
  lastCode.value = code;
  if (typeof navigator.vibrate === "function") navigator.vibrate(60);
  emit("detected", code, format);
  if (!props.continuous) visible.value = false;
}

async function toggleTorch() {
  try {
    await camera.toggleTorch();
  } catch (e) {
    notifyError(e, t("common.scan.failed"));
  }
}

// ── Lecture dans une photo (sans caméra, ou code difficile) ──────────
async function fromPhoto(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  busy.value = true;
  try {
    detector ??= await createDetector();
    const bitmap = await createImageBitmap(file);
    try {
      const codes = await detector.detect(bitmap);
      const hit = codes.find((c) => c.rawValue.trim());
      if (!hit) {
        notifyWarning(t("common.scan.nothingFound"));
        return;
      }
      onCode(hit.rawValue.trim(), hit.format);
    } finally {
      bitmap.close();
    }
  } catch (e) {
    notifyError(e, t("common.scan.failed"));
  } finally {
    busy.value = false;
  }
}

function stop() {
  stopped = true;
  window.clearTimeout(timer);
  camera.stop();
}
</script>
