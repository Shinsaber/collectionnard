<template>
  <aside
    class="sticky top-0 z-10 flex h-dvh w-62 shrink-0 flex-col gap-6 overflow-y-auto bg-bar px-3.5 py-5 shadow-side"
  >
    <div class="flex items-center justify-between">
      <RouterLink to="/" class="rounded-md px-2 py-1">
        <AppLogo :size="30" />
      </RouterLink>
      <NotificationBell />
    </div>

    <nav :aria-label="t('layout.mainNav')" class="flex flex-col gap-1">
      <RouterLink
        v-for="link in mainLinks"
        :key="link.to"
        :to="link.to"
        class="flex h-11 items-center gap-3 rounded-[10px] px-3 text-[15px] transition-colors"
        :class="linkClass(link.to)"
      >
        <NavIcon :name="link.icon" />
        {{ link.label }}
      </RouterLink>
    </nav>

    <nav
      v-if="adminLinks.length > 0"
      :aria-labelledby="adminTitleId"
      class="flex flex-col gap-1"
    >
      <span
        :id="adminTitleId"
        class="px-3 pb-1.5 font-mono text-[11px] uppercase tracking-wider text-muted"
        >{{ t("layout.adminSection") }}</span
      >
      <RouterLink
        v-for="link in adminLinks"
        :key="link.to"
        :to="link.to"
        class="flex h-10 items-center gap-3 rounded-[10px] px-3 text-sm transition-colors"
        :class="linkClass(link.to)"
      >
        <NavIcon :name="link.icon" :size="18" />
        {{ link.label }}
      </RouterLink>
    </nav>

    <div class="flex-1"></div>
    <AppUserCard />
  </aside>
</template>

<script setup lang="ts">
import { useId } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import AppLogo from "./AppLogo.vue";
import NotificationBell from "./NotificationBell.vue";
import AppUserCard from "./AppUserCard.vue";
import NavIcon from "./NavIcon.vue";
import { isNavActive, useNavigation } from "./navigation";

const route = useRoute();
const { t } = useI18n();
const { mainLinks, adminLinks } = useNavigation();
const adminTitleId = useId();

function linkClass(to: string): string {
  return isNavActive(to, route.path)
    ? "bg-tint font-bold text-tint-ink shadow-tint"
    : "font-medium text-ink/80 hover:bg-soft hover:text-ink";
}
</script>
