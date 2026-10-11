<template>
  <div class="flex flex-col gap-3">
    <div v-for="cat in categories" :key="cat.name" class="flex flex-col gap-1">
      <span
        class="text-xs font-medium text-gray-500 dark:text-gray-400 capitalize"
        >{{ cat.name }}</span
      >
      <div class="flex flex-wrap gap-2">
        <div
          v-for="(url, imgIdx) in urlsOf(cat.name)"
          :key="imgIdx"
          class="relative group"
        >
          <button
            type="button"
            class="block rounded-sm border focus:outline-hidden focus:ring-2 focus:ring-blue-500"
            :class="{ 'ring-2 ring-blue-500': defaultUrl === url }"
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
              class="w-20 object-cover rounded-sm"
              :style="{ aspectRatio: cat.ratio || '1/1' }"
            />
          </button>
          <button
            type="button"
            class="absolute -top-1 -right-1 bg-red-500 text-white rounded-full w-5 h-5 text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 focus:opacity-100 transition-opacity"
            :aria-label="
              t('schema.gallery.removeN', { n: imgIdx + 1, category: cat.name })
            "
            @click="removeImage(cat.name, imgIdx)"
          >
            ×
          </button>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <input
          :ref="(el) => setFileRef(cat.name, el)"
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          class="hidden"
          :aria-label="t('schema.gallery.file', { category: cat.name })"
          @change="handleUpload(cat.name, $event)"
        />
        <ElButton
          size="small"
          plain
          :loading="uploading[cat.name] ?? false"
          @click="fileInputs[cat.name]?.click()"
        >
          <i class="pi pi-upload mr-2" aria-hidden="true"></i>
          {{ t("common.add") }}
        </ElButton>
      </div>
    </div>
    <div v-if="defaultUrl" class="text-xs text-gray-400 dark:text-gray-500">
      {{ t("schema.gallery.default") }}
      <span class="font-medium text-gray-600 dark:text-gray-300">{{
        defaultUrl
      }}</span>
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
