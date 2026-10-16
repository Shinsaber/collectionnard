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

        <!-- Ajout : tuile au format de la catégorie (appareil photo ou galerie sur mobile). -->
        <input
          :ref="(el) => setFileRef(cat.name, el)"
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          class="hidden"
          :aria-label="t('schema.gallery.file', { category: cat.name })"
          @change="handleUpload(cat.name, $event)"
        />
        <button
          type="button"
          class="flex w-20 flex-col items-center justify-center gap-1 rounded-lg border-[1.5px] border-dashed border-dash text-xs font-semibold text-link disabled:opacity-60"
          :style="{ aspectRatio: cat.ratio || '1/1' }"
          :disabled="uploading[cat.name] ?? false"
          @click="fileInputs[cat.name]?.click()"
        >
          <i
            class="pi text-base"
            :class="uploading[cat.name] ? 'pi-spin pi-spinner' : 'pi-plus'"
            aria-hidden="true"
          ></i>
          {{ t("common.add") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { uploadApi } from "@/api/client";
import { useI18n } from "vue-i18n";
import { useNotify } from "@/composables/useNotify";

// Widget `x-widget: image-gallery` : un tableau d'URLs par catégorie
// (`x-image-categories`) et une `default_image`. Les erreurs d'upload (type refusé,
// 413 au-delà de MAX_UPLOAD_SIZE…) sont notifiées.
const props = defineProps<{
  categories: { name: string; ratio: string }[];
  modelValue: Record<string, unknown> | null | undefined;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: Record<string, unknown>];
}>();

const { notifyError } = useNotify();
const { t } = useI18n();

const data = computed(() => props.modelValue ?? {});
const defaultUrl = computed(() => (data.value.default_image as string) ?? "");
const uploading = reactive<Record<string, boolean>>({});
const fileInputs = ref<Record<string, HTMLInputElement | null>>({});

function setFileRef(cat: string, el: unknown) {
  fileInputs.value[cat] = el as HTMLInputElement | null;
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

async function handleUpload(cat: string, event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  uploading[cat] = true;
  try {
    const resp = await uploadApi.upload(file);
    const next = { ...data.value, [cat]: [...urlsOf(cat), resp.url] };
    if (!next.default_image) next.default_image = resp.url;
    emit("update:modelValue", next);
  } catch (e) {
    notifyError(e, t("schema.gallery.uploadFailed"));
  } finally {
    uploading[cat] = false;
    input.value = "";
  }
}
</script>
