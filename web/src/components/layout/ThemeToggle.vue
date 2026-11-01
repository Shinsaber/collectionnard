<template>
  <!-- Bascule clair / sombre des pages sans la coquille (publiques) ; mémorisée comme dans la carte utilisateur. -->
  <button
    type="button"
    class="flex size-9 items-center justify-center rounded-full text-muted hover:bg-surface"
    :title="label"
    :aria-label="label"
    data-test="theme-toggle"
    @click="toggle"
  >
    <i
      :class="theme === 'dark' ? 'pi pi-sun' : 'pi pi-moon'"
      aria-hidden="true"
    ></i>
  </button>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useTheme } from "@/composables/useTheme";

const { t } = useI18n();
const { theme, toggle } = useTheme();
const label = computed(() =>
  theme.value === "dark" ? t("layout.lightMode") : t("layout.darkMode"),
);
</script>
