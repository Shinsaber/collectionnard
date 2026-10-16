<template>
  <span class="inline-flex items-center gap-2 text-ink">
    <svg
      :width="size"
      :height="size"
      viewBox="0 0 56 56"
      :role="wordmark ? undefined : 'img'"
      :aria-label="wordmark ? undefined : 'Collectionnard'"
      :aria-hidden="wordmark ? 'true' : undefined"
      class="shrink-0"
    >
      <!-- Étiquette ambre à œillet ; evenodd : le trou laisse voir le fond, quel qu'il soit. -->
      <path
        :d="TAG_PATH"
        fill="var(--c-accent)"
        fill-rule="evenodd"
        stroke="currentColor"
        :stroke-width="size < 32 ? 4 : 3"
        stroke-linejoin="round"
      />
    </svg>
    <span
      v-if="wordmark"
      class="font-extrabold tracking-tight leading-none"
      :style="{ fontSize: `${Math.round(size * 0.68)}px` }"
      >collectionnard</span
    >
  </span>
</template>

<script setup lang="ts">
// Logo de l'identité « Étiquette » : même tracé que public/favicon.svg.
withDefaults(defineProps<{ size?: number; wordmark?: boolean }>(), {
  size: 28,
  wordmark: true,
});

const TAG_PATH =
  "M8 14a6 6 0 0 1 6-6h18l16 16a4 4 0 0 1 0 5.6L29.6 48a4 4 0 0 1-5.6 0L8 32Z" +
  "M21.5 18a3.5 3.5 0 1 0-7 0a3.5 3.5 0 1 0 7 0Z";
</script>
