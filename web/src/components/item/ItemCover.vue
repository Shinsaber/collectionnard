<template>
  <!-- Vignette d'un item : image par défaut du schéma, sinon initiales sur fond doux. -->
  <div
    class="relative shrink-0 overflow-hidden rounded-md bg-soft shadow-cover"
    :style="{ width: `${width}px`, aspectRatio: ratio }"
  >
    <img
      v-if="src"
      :src="src"
      :alt="alt ?? ''"
      loading="lazy"
      class="size-full object-cover"
    />
    <span
      v-else
      class="absolute inset-0 flex items-end p-1.5 font-mono text-[10px] font-medium uppercase leading-none text-muted"
      aria-hidden="true"
      >{{ initials }}</span
    >
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { defaultImage, type Schema } from "@/lib/schema";
import type { Item } from "@/api/types";

const props = withDefaults(
  defineProps<{
    item: Item;
    label: string;
    width?: number;
    ratio?: string;
    /** Texte alternatif ; vide par défaut, le libellé étant affiché à côté. */
    alt?: string;
  }>(),
  { width: 40, ratio: "2 / 3", alt: undefined },
);

const src = computed(() =>
  defaultImage(
    props.item.item_type?.attributes_schema as Schema | undefined,
    props.item.attributes,
  ),
);

/** Deux premières lettres significatives du libellé (« 3. Lost fragment » → « LF »). */
const initials = computed(() =>
  props.label
    .split(/\s+/)
    .map((w) => w.replace(/[^\p{L}\p{N}]/gu, ""))
    .filter((w) => w && !/^\d+$/.test(w))
    .slice(0, 2)
    .map((w) => w[0]!.toUpperCase())
    .join(""),
);
</script>
