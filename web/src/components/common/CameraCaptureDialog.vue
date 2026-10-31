<template>
  <!--
    Prise de vue dans l'application (retour v0.4.0) : sur Android, `<input capture>` peut n'ouvrir
    que le gestionnaire de fichiers ; ici la caméra arrière s'affiche dans la page (useCamera) et
    « Prendre la photo » capture l'image (ImageCapture.takePhoto à pleine résolution quand le
    navigateur l'offre, sinon l'image vidéo courante). Repli : l'appareil photo du téléphone.
  -->
  <ElDialog
    v-model="visible"
    :title="t('schema.gallery.cameraTitle', { category })"
    width="30rem"
    append-to-body
    @opened="start"
    @closed="camera.stop"
  >
    <div
      class="relative h-[60vh] max-h-128 overflow-hidden rounded-lg bg-ink shadow-cover"
      data-test="camera-stage"
    >
      <video
        ref="video"
        class="h-full w-full object-contain"
        playsinline
        muted
        autoplay
        :aria-label="t('common.scan.preview')"
      ></video>
      <div
        v-if="camera.state.value === 'starting' || busy"
        class="absolute inset-0 grid place-items-center"
      >
        <AppSpinner />
      </div>
      <p
        v-else-if="camera.state.value === 'unavailable'"
        class="absolute inset-0 grid place-items-center p-6 text-center text-sm text-surface"
        role="status"
        data-test="camera-message"
      >
        {{ message }}
      </p>
    </div>
    <p class="mt-3 text-sm text-muted">{{ t("schema.gallery.cameraHint") }}</p>
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
      <span class="flex-1"></span>
      <!-- Appareil photo du téléphone : seul choix quand la caméra est inaccessible ici. -->
      <input
        ref="nativeInput"
        type="file"
        accept="image/*"
        capture="environment"
        class="hidden"
        :aria-label="t('schema.gallery.nativeCamera')"
        @change="fromNative"
      />
      <ElButton data-test="camera-native" @click="nativeInput?.click()">
        <i class="pi pi-mobile mr-2" aria-hidden="true"></i>
        {{ t("schema.gallery.nativeCamera") }}
      </ElButton>
    </div>
    <template #footer>
      <ElButton text @click="visible = false">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        size="large"
        :disabled="camera.state.value !== 'ready'"
        :loading="busy"
        data-test="camera-shutter"
        @click="shoot"
      >
        <i v-if="!busy" class="pi pi-camera mr-2" aria-hidden="true"></i>
        {{ t("schema.gallery.shutter") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useNotify } from "@/composables/useNotify";
import { useCamera } from "@/composables/useCamera";
import { frameOf } from "@/lib/images";
import AppSpinner from "./AppSpinner.vue";

defineProps<{
  /** Nom de la catégorie d'images visée (titre du dialogue). */
  category: string;
}>();
const emit = defineEmits<{ done: [file: File] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { notifyError } = useNotify();

const video = ref<HTMLVideoElement | null>(null);
const nativeInput = ref<HTMLInputElement | null>(null);
const camera = useCamera(video);
const message = ref("");
const busy = ref(false);

/** Résolution souhaitée de l'aperçu : assez pour une couverture quand `takePhoto` manque. */
const PREVIEW_WIDTH = 2048;
const PREVIEW_HEIGHT = 1536;

async function start() {
  const failure = await camera.start(PREVIEW_WIDTH, PREVIEW_HEIGHT);
  if (!failure) return;
  message.value = t(
    failure.reason === "denied"
      ? "schema.gallery.cameraDenied"
      : "schema.gallery.cameraNone",
  );
  if (failure.error)
    notifyError(failure.error, t("schema.gallery.captureFailed"));
}

async function toggleTorch() {
  try {
    await camera.toggleTorch();
  } catch (e) {
    notifyError(e, t("schema.gallery.captureFailed"));
  }
}

/** Photo à pleine résolution si le navigateur sait la prendre, sinon l'image vidéo. */
async function shoot() {
  const v = video.value;
  const track = camera.track();
  if (!v || !track) return;
  busy.value = true;
  try {
    const blob = (await takePhoto(track)) ?? (await frameOf(v));
    const type = blob.type || "image/jpeg";
    const ext = type === "image/png" ? "png" : "jpg";
    finish(new File([blob], `photo-${Date.now()}.${ext}`, { type }));
  } catch (e) {
    notifyError(e, t("schema.gallery.captureFailed"));
  } finally {
    busy.value = false;
  }
}

// `ImageCapture` (Chrome, Android) n'est pas dans les types du DOM ; null si absent ou en échec,
// l'image vidéo courante suffit alors.
interface ImageCaptureLike {
  takePhoto(): Promise<Blob>;
}

async function takePhoto(track: MediaStreamTrack): Promise<Blob | null> {
  const Ctor = (
    window as unknown as {
      ImageCapture?: new (t: MediaStreamTrack) => ImageCaptureLike;
    }
  ).ImageCapture;
  if (!Ctor) return null;
  try {
    return await new Ctor(track).takePhoto();
  } catch {
    return null;
  }
}

function fromNative(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (file) finish(file);
}

function finish(file: File) {
  emit("done", file);
  visible.value = false;
}
</script>
