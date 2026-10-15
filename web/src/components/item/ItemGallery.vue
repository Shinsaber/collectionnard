<template>
  <section v-if="gallery" class="flex flex-col gap-3.5">
    <h2 class="text-xl font-extrabold tracking-tight sm:text-[22px]">
      {{ fieldLabel(schema, gallery.key, lang) }}
    </h2>
    <div
      class="rounded-2xl bg-surface px-4.5 pb-4.5 shadow-card sm:px-6 sm:pb-6"
    >
      <ElTabs v-model="activeTab">
        <ElTabPane
          v-for="cat in gallery.categories"
          :key="cat.name"
          :name="cat.name"
        >
          <template #label>
            <span class="capitalize">{{ cat.name }}</span>
            <span class="ml-1 font-mono text-xs text-muted"
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
                class="w-28 rounded-lg object-cover shadow-cover sm:w-32"
                :style="{ aspectRatio: cat.ratio || '1/1' }"
                :class="{
                  'ring-2 ring-primary ring-offset-2 ring-offset-surface':
                    defaultUrl === url,
                }"
              />
              <span
                v-if="defaultUrl === url"
                class="absolute top-1.5 left-1.5 rounded-sm bg-primary px-1.5 py-0.5 font-mono text-[10px] uppercase text-primary-ink"
                >{{ t("items.gallery.default") }}</span
              >
            </div>
            <div
              v-if="urlsOf(cat.name).length === 0"
              class="flex h-24 w-full items-center justify-center rounded-xl border-[1.5px] border-dashed border-dash text-sm text-muted"
            >
              {{ t("items.gallery.empty") }}
            </div>
          </div>
        </ElTabPane>
      </ElTabs>
    </div>
  </section>
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
