<template>
  <!-- Vignettes d'une galerie (photos d'un exemplaire, lot M5) : les premières, puis « +n ». -->
  <div
    v-if="urls.length"
    class="flex flex-wrap items-center gap-1.5"
    data-test="gallery-thumbs"
  >
    <a
      v-for="(url, i) in shown"
      :key="url"
      :href="url"
      target="_blank"
      rel="noopener noreferrer"
      class="block overflow-hidden rounded-md shadow-cover"
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
    </a>
    <span v-if="urls.length > shown.length" class="font-mono text-xs text-muted"
      >+{{ urls.length - shown.length }}</span
    >
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

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

const shown = computed(() => props.urls.slice(0, props.max));

/** Vignette servie par le serveur pour les envois ; image telle quelle sinon. */
function thumb(url: string): string {
  return url.startsWith("/api/v1/uploads/") ? `${url}?w=240` : url;
}
</script>
