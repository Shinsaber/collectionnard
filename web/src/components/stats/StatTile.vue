<template>
  <div
    class="flex min-w-0 flex-col gap-1 rounded-2xl bg-surface p-4 shadow-card"
  >
    <span
      class="truncate font-mono text-[11px] uppercase tracking-wider text-muted"
      >{{ label }}</span
    >
    <span class="text-2xl font-extrabold tracking-tight">{{ value }}</span>
    <span v-if="detail" class="text-xs text-muted">{{ detail }}</span>
  </div>
</template>

<script setup lang="ts">
// Chiffre clé : libellé en capitales, valeur, précision facultative.
defineProps<{ label: string; value: string | number; detail?: string }>();
</script>
