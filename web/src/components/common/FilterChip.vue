<template>
  <!-- Filtre actif : pastille encre, croix pour le retirer. -->
  <span
    class="inline-flex h-8 items-center gap-1.5 rounded-full bg-ink pr-1 pl-3 text-[13px] font-medium text-canvas"
  >
    <span class="max-w-60 truncate">{{ label }}</span>
    <button
      type="button"
      class="flex size-6 items-center justify-center rounded-full bg-canvas/20 text-canvas hover:bg-canvas/35"
      :aria-label="t('items.search.removeFilter', { label })"
      @click="emit('remove')"
    >
      <i class="pi pi-times text-[10px]" aria-hidden="true"></i>
    </button>
  </span>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";

defineProps<{ label: string }>();
const emit = defineEmits<{ remove: [] }>();
const { t } = useI18n();
</script>
