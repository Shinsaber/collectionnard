<template>
  <!-- Avatar d'un compte : l'image envoyée (vignette), sinon l'initiale du nom affiché. -->
  <img
    v-if="src"
    :src="src"
    :alt="t('account.avatarAlt', { name })"
    :width="size"
    :height="size"
    class="shrink-0 rounded-full object-cover"
    :style="{ width: `${size}px`, height: `${size}px` }"
  />
  <span
    v-else
    class="flex shrink-0 items-center justify-center rounded-full bg-ink font-bold text-surface"
    :style="{
      width: `${size}px`,
      height: `${size}px`,
      fontSize: `${Math.round(size * 0.4)}px`,
    }"
    aria-hidden="true"
    >{{ initial }}</span
  >
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const props = withDefaults(
  defineProps<{
    user: { username: string; display_name?: string; avatar?: string } | null;
    size?: number;
  }>(),
  { size: 36 },
);
const { t } = useI18n();

const name = computed(
  () => props.user?.display_name || props.user?.username || "",
);
const initial = computed(() => name.value.charAt(0).toUpperCase() || "?");
/** Vignette de 240 px servie par le serveur (seule largeur permise : thumbWidths). */
const src = computed(() => {
  const a = props.user?.avatar;
  return a && a.startsWith("/api/v1/uploads/") ? `${a}?w=240` : "";
});
</script>
