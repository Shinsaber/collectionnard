<template>
  <div class="flex flex-col gap-4">
    <div v-for="cat in categories" :key="cat.name" class="flex flex-col gap-2">
      <span class="font-mono text-[11px] uppercase tracking-wider text-muted"
        >{{ cat.name }} · {{ urlsOf(cat.name).length }}</span
      >
      <div class="flex flex-wrap gap-3">
        <div
          v-for="(url, imgIdx) in urlsOf(cat.name)"
          :key="imgIdx"
          class="group relative"
        >
          <button
            type="button"
            class="block overflow-hidden rounded-lg shadow-cover"
            :class="{
              'ring-2 ring-primary ring-offset-2 ring-offset-surface':
                defaultUrl === url,
            }"
            :title="t('schema.gallery.setDefault')"
            :aria-label="
              t('schema.gallery.setDefaultN', {
                n: imgIdx + 1,
                category: cat.name,
              })
            "
            :aria-pressed="defaultUrl === url"
            @click="setDefault(url)"
          >
            <img
              :src="url"
              :alt="`${cat.name} ${imgIdx + 1}`"
              class="w-20 object-cover"
              :style="{ aspectRatio: cat.ratio || '1/1' }"
            />
          </button>
          <span
            v-if="defaultUrl === url"
            class="pointer-events-none absolute top-1.5 left-1.5 rounded-sm bg-primary px-1 font-mono text-[9px] uppercase text-primary-ink"
            >{{ t("items.gallery.default") }}</span
          >
          <button
            type="button"
            class="absolute -top-2 -right-2 flex size-6 items-center justify-center rounded-full bg-danger text-xs text-surface shadow-control transition-opacity lg:opacity-0 lg:group-hover:opacity-100 lg:focus:opacity-100"
            :aria-label="
              t('schema.gallery.removeN', { n: imgIdx + 1, category: cat.name })
            "
            @click="removeImage(cat.name, imgIdx)"
          >
            <i class="pi pi-times text-[10px]" aria-hidden="true"></i>
          </button>
        </div>

        <!-- Ajout depuis un fichier (ou la galerie du téléphone) : tuile au format de la catégorie. -->
        <input
          :ref="(el) => setInput(cat.name, 'file', el)"
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          class="hidden"
          :aria-label="t('schema.gallery.file', { category: cat.name })"
          @change="onFile(cat, $event, false)"
        />
        <button
          type="button"
          class="flex w-20 flex-col items-center justify-center gap-1 rounded-lg border-[1.5px] border-dashed border-dash text-xs font-semibold text-link disabled:opacity-60"
          :style="{ aspectRatio: cat.ratio || '1/1' }"
          :disabled="uploading[cat.name] ?? false"
          @click="inputs[`${cat.name}/file`]?.click()"
        >
          <i
            class="pi text-base"
            :class="uploading[cat.name] ? 'pi-spin pi-spinner' : 'pi-plus'"
            aria-hidden="true"
          ></i>
          {{ t("common.add") }}
        </button>

        <!-- Photo (lot M1, écrans tactiles) : appareil photo arrière, puis recadrage au ratio. -->
        <template v-if="coarsePointer">
          <input
            :ref="(el) => setInput(cat.name, 'camera', el)"
            type="file"
            accept="image/*"
            capture="environment"
            class="hidden"
            :aria-label="t('schema.gallery.photoLabel', { category: cat.name })"
            @change="onFile(cat, $event, true)"
          />
          <button
            type="button"
            class="flex w-20 flex-col items-center justify-center gap-1 rounded-lg border-[1.5px] border-dashed border-dash text-xs font-semibold text-link disabled:opacity-60"
            :style="{ aspectRatio: cat.ratio || '1/1' }"
            :disabled="uploading[cat.name] ?? false"
            data-test="gallery-photo"
            @click="inputs[`${cat.name}/camera`]?.click()"
          >
            <i class="pi pi-camera text-base" aria-hidden="true"></i>
            {{ t("schema.gallery.photo") }}
          </button>
        </template>
      </div>
    </div>

    <ImageCropDialog
      v-model:visible="cropVisible"
      :file="cropFile"
      :category="cropCategory"
      :reason="cropReason"
      :straighten="cropStraighten"
      @done="onCropped"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, reactive, ref } from "vue";
import { uploadApi } from "@/api/client";
import { useI18n } from "vue-i18n";
import { useNotify } from "@/composables/useNotify";
import { useLang } from "@/composables/useLang";
import {
  COARSE_POINTER_QUERY,
  useMediaQuery,
} from "@/composables/useMediaQuery";
import { checkImage, type ImageCategory, type ImageIssue } from "@/lib/schema";
import { imageSize, type Size } from "@/lib/images";
// cropperjs n'est chargé qu'au premier recadrage (hors du paquet de DynamicForm).
const ImageCropDialog = defineAsyncComponent(
  () => import("./ImageCropDialog.vue"),
);

// Widget `x-widget: image-gallery` : un tableau d'URLs par catégorie
// (`x-image-categories`) et une `default_image`. Les erreurs d'upload (type refusé,
// 413 au-delà de MAX_UPLOAD_SIZE…) sont notifiées. Une photo passe toujours par le
// recadrage ; un fichier hors format (catégorie `strict`) y est envoyé au lieu d'être refusé.
const props = defineProps<{
  categories: ImageCategory[];
  modelValue: Record<string, unknown> | null | undefined;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: Record<string, unknown>];
}>();

const { notifyError, notifyWarning } = useNotify();
const { lang } = useLang();
const { t } = useI18n();
const coarsePointer = useMediaQuery(COARSE_POINTER_QUERY);

const data = computed(() => props.modelValue ?? {});
const defaultUrl = computed(() => (data.value.default_image as string) ?? "");
const uploading = reactive<Record<string, boolean>>({});
const inputs = ref<Record<string, HTMLInputElement | null>>({});

function setInput(cat: string, kind: "file" | "camera", el: unknown) {
  inputs.value[`${cat}/${kind}`] = el as HTMLInputElement | null;
}

function urlsOf(cat: string): string[] {
  return (data.value[cat] as string[]) ?? [];
}

function setDefault(url: string) {
  emit("update:modelValue", { ...data.value, default_image: url });
}

function removeImage(cat: string, imgIdx: number) {
  const next = { ...data.value };
  const arr = [...urlsOf(cat)];
  const [removed] = arr.splice(imgIdx, 1);
  next[cat] = arr;
  if (next.default_image === removed) next.default_image = "";
  emit("update:modelValue", next);
}

// ── Recadrage ────────────────────────────────────────────────────────
const cropVisible = ref(false);
const cropFile = ref<File | null>(null);
const cropCategory = ref<ImageCategory | null>(null);
const cropReason = ref<string>();
/** Photo prise à l'instant : commencer par redresser la perspective (lot M4). */
const cropStraighten = ref(false);

function openCrop(
  cat: ImageCategory,
  file: File,
  reason?: string,
  straighten = false,
) {
  cropStraighten.value = straighten;
  cropFile.value = file;
  cropCategory.value = cat;
  cropReason.value = reason;
  cropVisible.value = true;
}

async function onCropped(file: File) {
  const cat = cropCategory.value;
  if (!cat) return;
  // Le cadre impose le ratio ; la taille minimale, elle, peut encore manquer.
  const size = await imageSize(file);
  const verdict = size ? verdictOf(cat, size) : undefined;
  if (verdict?.block) {
    notifyError(new Error(verdict.details), t("schema.gallery.refused"));
    return;
  }
  if (verdict?.details)
    notifyWarning(t("schema.gallery.warned", { details: verdict.details }));
  await send(cat, file);
}

// ── Envoi ────────────────────────────────────────────────────────────
async function onFile(cat: ImageCategory, event: Event, fromCamera: boolean) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  if (fromCamera) {
    openCrop(cat, file, undefined, true);
    return;
  }
  const size = await imageSize(file);
  const verdict = size ? verdictOf(cat, size) : undefined;
  if (verdict?.block) {
    openCrop(
      cat,
      file,
      t("schema.gallery.cropOffer", { details: verdict.details }),
    );
    return;
  }
  if (verdict?.details)
    notifyWarning(t("schema.gallery.warned", { details: verdict.details }));
  await send(cat, file);
}

async function send(cat: ImageCategory, file: File) {
  uploading[cat.name] = true;
  try {
    const resp = await uploadApi.upload(file);
    const next = {
      ...data.value,
      [cat.name]: [...urlsOf(cat.name), resp.url],
    };
    if (!next.default_image) next.default_image = resp.url;
    emit("update:modelValue", next);
  } catch (e) {
    notifyError(e, t("schema.gallery.uploadFailed"));
  } finally {
    uploading[cat.name] = false;
  }
}

/** Contrôles de la catégorie (ratio, taille minimale) : défauts décrits, et s'ils bloquent. */
function verdictOf(
  cat: ImageCategory,
  size: Size,
): { details: string; block: boolean } | undefined {
  const { issues, block } = checkImage(cat, size.width, size.height);
  if (!issues.length) return undefined;
  return { details: issues.map((i) => describe(i, size)).join(" ; "), block };
}

function describe(issue: ImageIssue, size: Size): string {
  const n = (v: number, digits = 0) =>
    v.toLocaleString(lang.value, { maximumFractionDigits: digits });
  return issue.kind === "ratio"
    ? t("schema.gallery.ratioOff", {
        actual: n(issue.actual, 2),
        expected: issue.expected,
        deviation: n(issue.deviation),
        tolerance: n(issue.tolerance),
      })
    : t("schema.gallery.tooSmall", {
        width: size.width,
        height: size.height,
        min: `${issue.minWidth || "–"} × ${issue.minHeight || "–"}`,
      });
}
</script>
