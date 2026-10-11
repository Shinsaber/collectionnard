<template>
  <div
    v-if="gallery"
    class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-6 mb-6"
  >
    <h2 class="text-lg font-semibold text-gray-700 dark:text-gray-200 mb-4">
      {{ fieldLabel(schema, gallery.key, lang) }}
    </h2>
    <ElTabs v-model="activeTab">
      <ElTabPane
        v-for="cat in gallery.categories"
        :key="cat.name"
        :name="cat.name"
      >
        <template #label>
          <span class="capitalize">{{ cat.name }}</span>
          <span class="ml-1 text-xs text-gray-400 dark:text-gray-500"
            >({{ urlsOf(cat.name).length }})</span
          >
        </template>
        <div class="flex flex-wrap gap-3 pt-3">
          <div
            v-for="(url, imgIdx) in urlsOf(cat.name)"
            :key="imgIdx"
            class="relative"
          >
            <img
              :src="url"
              :alt="
                t('items.gallery.imageAlt', {
                  category: cat.name,
                  n: imgIdx + 1,
                })
              "
              class="w-32 object-cover rounded-lg border"
              :style="{ aspectRatio: cat.ratio || '1/1' }"
              :class="{ 'ring-2 ring-blue-500': defaultUrl === url }"
            />
            <span
              v-if="defaultUrl === url"
              class="absolute top-1 left-1 bg-blue-500 text-white text-[10px] px-1.5 py-0.5 rounded-sm"
              >{{ t("items.gallery.default") }}</span
            >
          </div>
          <div
            v-if="urlsOf(cat.name).length === 0"
            class="text-sm text-gray-400 dark:text-gray-500 italic py-4"
          >
            {{ t("items.gallery.empty") }}
          </div>
        </div>
      </ElTabPane>
    </ElTabs>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import {
  fieldLabel,
  imageCategories,
  isImageGallery,
  type Schema,
} from "@/lib/schema";

// Galerie d'images d'un item : premier champ `x-widget: image-gallery` du schéma,
// un onglet par catégorie (`x-image-categories`).
const props = defineProps<{
  attributes: Record<string, unknown>;
  schema: Schema | null | undefined;
}>();

const { t } = useI18n();
const { lang } = useLang();

const gallery = computed(() => {
  for (const key of Object.keys(props.attributes)) {
    if (isImageGallery(props.schema, key)) {
      return { key, categories: imageCategories(props.schema, key) };
    }
  }
  return null;
});

const activeTab = ref(gallery.value?.categories[0]?.name ?? "");
watch(gallery, (g) => {
  if (g && !g.categories.some((c) => c.name === activeTab.value)) {
    activeTab.value = g.categories[0]?.name ?? "";
  }
});

const data = computed(
  () =>
    (gallery.value
      ? (props.attributes[gallery.value.key] as Record<string, unknown>)
      : null) ?? {},
);

function urlsOf(cat: string): string[] {
  return (data.value[cat] as string[]) ?? [];
}

const defaultUrl = computed(() => (data.value.default_image as string) ?? "");
</script>
