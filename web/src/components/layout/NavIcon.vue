<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    class="shrink-0"
  >
    <path :d="NAV_ICONS[name]" />
  </svg>
</template>

<script setup lang="ts">
import { NAV_ICONS, type NavIconName } from "./navigation";

withDefaults(
  defineProps<{ name: NavIconName; size?: number; strokeWidth?: number }>(),
  { size: 20, strokeWidth: 2 },
);
</script>
