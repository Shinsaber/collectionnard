<template>
  <!-- Barre du haut (mobile) : logo et accès au compte ; la navigation est dans la barre d'onglets. -->
  <header
    class="sticky top-0 z-20 flex items-center justify-between bg-bar px-4 pt-[env(safe-area-inset-top)] shadow-bar"
  >
    <div class="flex h-15 items-center">
      <RouterLink to="/" class="rounded-md">
        <AppLogo :size="28" />
      </RouterLink>
    </div>
    <button
      type="button"
      class="flex size-11 items-center justify-center rounded-full bg-ink text-sm font-bold text-surface shadow-control"
      :aria-label="t('layout.account')"
      aria-haspopup="dialog"
      @click="emit('openMore')"
    >
      <span aria-hidden="true">{{ initial }}</span>
    </button>
  </header>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import AppLogo from "./AppLogo.vue";

const emit = defineEmits<{ openMore: [] }>();
const { t } = useI18n();
const auth = useAuthStore();
const initial = computed(
  () => auth.user?.username?.charAt(0).toUpperCase() ?? "?",
);
</script>
