<template>
  <!-- En-tête de page : surtitre mono, titre, compteur ; les actions passent dessous sur mobile. -->
  <div class="mb-6 flex flex-wrap items-end justify-between gap-x-4 gap-y-3">
    <div class="flex min-w-0 flex-col gap-1">
      <span
        v-if="eyebrow"
        class="font-mono text-xs uppercase tracking-wider text-muted"
        >{{ eyebrow }}</span
      >
      <div class="flex flex-wrap items-baseline gap-x-3.5 gap-y-1">
        <h1
          class="text-[28px] font-extrabold leading-tight tracking-tight lg:text-[34px]"
        >
          {{ title }}
        </h1>
        <span
          v-if="meta"
          class="font-mono text-xs uppercase tracking-wide text-muted"
          >{{ meta }}</span
        >
      </div>
      <slot name="subtitle" />
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2.5">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ title: string; eyebrow?: string; meta?: string }>();
</script>
