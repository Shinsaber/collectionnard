<template>
  <!-- Outils du redressement (PerspectiveEditor) : rotation, détection, image entière. -->
  <div class="btn-row flex flex-wrap items-center gap-2">
    <ElButton
      :aria-label="t('schema.gallery.rotateLeft')"
      :disabled="busy || !hasImage"
      data-test="perspective-rotate-left"
      @click="emit('rotate', -1)"
    >
      <i class="pi pi-replay" aria-hidden="true"></i>
    </ElButton>
    <ElButton
      :aria-label="t('schema.gallery.rotateRight')"
      :disabled="busy || !hasImage"
      data-test="perspective-rotate-right"
      @click="emit('rotate', 1)"
    >
      <i class="pi pi-refresh" aria-hidden="true"></i>
    </ElButton>
    <ElButton
      :aria-label="t('schema.gallery.detectEdges')"
      :disabled="busy || !hasImage"
      @click="emit('detect')"
    >
      <i
        class="pi pi-objects-column mr-2 max-md:hidden!"
        aria-hidden="true"
      ></i>
      {{ t("schema.gallery.detect") }}
    </ElButton>
    <ElButton :disabled="busy || !hasSize" @click="emit('reset')">
      {{ t("schema.gallery.wholeImage") }}
    </ElButton>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";

defineProps<{
  busy: boolean;
  /** Image décodée (rotation et détection possibles). */
  hasImage: boolean;
  /** Dimensions connues (image entière possible). */
  hasSize: boolean;
}>();
const emit = defineEmits<{
  rotate: [turns: 1 | -1];
  detect: [];
  reset: [];
}>();
const { t } = useI18n();
</script>
