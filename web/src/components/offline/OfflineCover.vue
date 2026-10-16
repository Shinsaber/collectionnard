<template>
  <!-- Couverture hors ligne : vignette gardée sur l'appareil, sinon le réseau, sinon les initiales. -->
  <div
    class="relative w-12 shrink-0 self-start overflow-hidden rounded-md bg-soft shadow-cover"
    style="aspect-ratio: 2 / 3"
  >
    <img
      v-if="src"
      :src="src"
      alt=""
      loading="lazy"
      class="size-full object-cover"
      @error="src = null"
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
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { cachedCover } from "@/lib/offlineCovers";

const props = defineProps<{
  /** Vignette (`…?w=240`), vide si l'item n'a pas d'image envoyée. */
  url: string;
  label: string;
  /** Serveur joignable : la vignette absente du cache est demandée au réseau. */
  online: boolean;
}>();

const src = ref<string | null>(null);
let objectUrl: string | null = null;

function release() {
  if (objectUrl) URL.revokeObjectURL(objectUrl);
  objectUrl = null;
}

watch(
  () => props.url,
  async (url) => {
    release();
    objectUrl = await cachedCover(url).catch(() => null);
    src.value = objectUrl ?? (url && props.online ? url : null);
  },
  { immediate: true },
);
onBeforeUnmount(release);

/** Deux premières lettres significatives du libellé (comme ItemCover). */
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
