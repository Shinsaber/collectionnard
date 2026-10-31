<template>
  <!-- Vignettes d'une galerie (photos d'un exemplaire, lot M5) : les premières, puis « +n » ;
       chacune ouvre la visionneuse sur toute la galerie. -->
  <div
    v-if="urls.length"
    class="flex flex-wrap items-center gap-1.5"
    data-test="gallery-thumbs"
  >
    <button
      v-for="(url, i) in shown"
      :key="url"
      type="button"
      class="block overflow-hidden rounded-md shadow-cover"
      :aria-label="t('common.enlarge', { name: `${label} ${i + 1}` })"
      @click.stop="open(i)"
    >
      <img
        :src="thumb(url)"
        :alt="`${label} ${i + 1}`"
        class="object-cover"
        :style="{
          height: `${size}px`,
          width: `${Math.round(size * 0.75)}px`,
        }"
        loading="lazy"
      />
    </button>
    <button
      v-if="urls.length > shown.length"
      type="button"
      class="font-mono text-xs text-muted"
      :aria-label="
        t('common.enlarge', { name: `${label} ${shown.length + 1}` })
      "
      @click.stop="open(shown.length)"
    >
      +{{ urls.length - shown.length }}
    </button>
    <ImageLightbox
      v-model:visible="viewerOpen"
      :urls="urls"
      :index="viewerIndex"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import ImageLightbox from "./ImageLightbox.vue";

const props = withDefaults(
  defineProps<{
    urls: string[];
    /** Libellé de la galerie (texte alternatif : « Photos de l'exemplaire 1 »). */
    label: string;
    /** Hauteur des vignettes en pixels. */
    size?: number;
    /** Nombre de vignettes montrées avant « +n ». */
    max?: number;
  }>(),
  { size: 40, max: 4 },
);

const { t } = useI18n();
const shown = computed(() => props.urls.slice(0, props.max));
const viewerOpen = ref(false);
const viewerIndex = ref(0);

function open(i: number) {
  viewerIndex.value = i;
  viewerOpen.value = true;
}

/** Vignette servie par le serveur pour les envois ; image telle quelle sinon. */
function thumb(url: string): string {
  return url.startsWith("/api/v1/uploads/") ? `${url}?w=240` : url;
}
</script>
