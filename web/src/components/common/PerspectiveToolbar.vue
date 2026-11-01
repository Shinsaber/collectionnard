<template>
  <!-- Commandes du redressement (PerspectiveEditor) : rotation, détection, image entière, passer, appliquer. -->
  <div class="flex flex-wrap items-center gap-2">
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
    <ElButton :disabled="busy || !hasImage" @click="emit('detect')">
      <i class="pi pi-objects-column mr-2" aria-hidden="true"></i>
      {{ t("schema.gallery.detectEdges") }}
    </ElButton>
    <ElButton :disabled="busy || !hasSize" @click="emit('reset')">
      {{ t("schema.gallery.wholeImage") }}
    </ElButton>
    <span class="flex-1"></span>
    <ElButton :disabled="busy" @click="emit('skip')">
      {{ t("schema.gallery.skipStraighten") }}
    </ElButton>
    <ElButton
      type="primary"
      :loading="busy"
      :disabled="!hasSize || !hasQuad"
      data-test="perspective-apply"
      @click="emit('apply')"
    >
      <i v-if="!busy" class="pi pi-check mr-2" aria-hidden="true"></i>
      {{ t("schema.gallery.straighten") }}
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
  /** Quatre coins posés (redressement possible). */
  hasQuad: boolean;
}>();
const emit = defineEmits<{
  rotate: [turns: 1 | -1];
  detect: [];
  reset: [];
  skip: [];
  apply: [];
}>();
const { t } = useI18n();
</script>
