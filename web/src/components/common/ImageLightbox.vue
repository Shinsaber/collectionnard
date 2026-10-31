<template>
  <!--
    Visionneuse plein écran (retour v0.4.0) : les images d'une galerie en grand, avec zoom,
    rotation et passage à la suivante (ElImageViewer, au-dessus des dialogues grâce à `teleported`).
  -->
  <ElImageViewer
    v-if="visible"
    :url-list="urls"
    :initial-index="index"
    :infinite="urls.length > 1"
    hide-on-click-modal
    teleported
    @close="visible = false"
  />
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    /** Images à parcourir (URL d'origine, pas la vignette). */
    urls: string[];
    /** Image montrée à l'ouverture. */
    index?: number;
  }>(),
  { index: 0 },
);
const visible = defineModel<boolean>("visible", { required: true });
</script>
