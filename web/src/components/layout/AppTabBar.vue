<template>
  <!-- Barre d'onglets (mobile) : pastille de fond sur l'onglet actif ; « Plus » ouvre la feuille de réglages. -->
  <nav
    :aria-label="t('layout.mainNav')"
    class="fixed inset-x-0 bottom-0 z-20 grid grid-cols-4 bg-surface pb-[env(safe-area-inset-bottom)] shadow-bar-up"
  >
    <RouterLink
      v-for="link in tabLinks"
      :key="link.to"
      :to="link.to"
      class="flex h-18 flex-col items-center justify-center gap-0.5 text-[11px]"
      :class="tabClass(isNavActive(link.to, route.path))"
    >
      <span
        class="flex h-7.5 w-15 items-center justify-center rounded-full transition-colors"
        :class="pillClass(isNavActive(link.to, route.path))"
      >
        <NavIcon :name="link.icon" :size="22" />
      </span>
      {{ link.label }}
    </RouterLink>
    <button
      type="button"
      class="flex h-18 flex-col items-center justify-center gap-0.5 text-[11px]"
      :class="tabClass(moreActive)"
      aria-haspopup="dialog"
      :aria-expanded="moreOpen"
      @click="emit('openMore')"
    >
      <span
        class="flex h-7.5 w-15 items-center justify-center rounded-full transition-colors"
        :class="pillClass(moreActive)"
      >
        <NavIcon name="more" :size="22" />
      </span>
      {{ t("layout.nav.more") }}
    </button>
  </nav>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import NavIcon from "./NavIcon.vue";
import { isMoreActive, isNavActive, useNavigation } from "./navigation";

const props = defineProps<{ moreOpen: boolean }>();
const emit = defineEmits<{ openMore: [] }>();
const { t } = useI18n();
const route = useRoute();
const { tabLinks } = useNavigation();

const moreActive = computed(() => props.moreOpen || isMoreActive(route.path));

function tabClass(active: boolean): string {
  return active ? "font-bold text-tint-ink" : "font-medium text-muted";
}
function pillClass(active: boolean): string {
  return active ? "bg-tint shadow-tint" : "";
}
</script>
