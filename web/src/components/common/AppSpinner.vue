<template>
  <div class="flex justify-center py-12" role="status" aria-live="polite">
    <i class="pi pi-spin pi-spinner text-3xl text-muted" aria-hidden="true"></i>
    <span class="sr-only">{{ label }}</span>
  </div>
</template>

<script setup lang="ts">
// Indicateur de chargement pleine largeur (remplace le ProgressSpinner de PrimeVue).
withDefaults(defineProps<{ label?: string }>(), { label: "Chargement…" });
</script>
