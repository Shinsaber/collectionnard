<template>
  <!-- Feuille « Plus » (mobile) : compte, langue, thème et pages d'administration. -->
  <ElDrawer
    v-model="open"
    direction="btt"
    size="auto"
    :title="t('layout.account')"
    class="more-sheet"
  >
    <div class="flex flex-col gap-5 pb-[env(safe-area-inset-bottom)]">
      <AppUserCard />
      <nav
        v-if="auth.isAdmin"
        :aria-labelledby="adminTitleId"
        class="flex flex-col"
      >
        <span
          :id="adminTitleId"
          class="px-1 pb-2 font-mono text-[11px] uppercase tracking-wider text-muted"
          >{{ t("layout.adminSection") }}</span
        >
        <div
          class="flex flex-col overflow-hidden rounded-xl bg-surface shadow-card"
        >
          <RouterLink
            v-for="link in adminLinks"
            :key="link.to"
            :to="link.to"
            class="flex min-h-13 items-center gap-3 border-b border-line px-4 text-[15px] last:border-b-0"
            :class="
              isNavActive(link.to, route.path)
                ? 'bg-tint font-bold text-tint-ink'
                : 'font-medium text-ink'
            "
          >
            <NavIcon :name="link.icon" />
            <span class="flex-1">{{ link.label }}</span>
            <i class="pi pi-angle-right text-muted" aria-hidden="true"></i>
          </RouterLink>
        </div>
      </nav>
    </div>
  </ElDrawer>
</template>

<script setup lang="ts">
import { useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import AppUserCard from "./AppUserCard.vue";
import NavIcon from "./NavIcon.vue";
import { isNavActive, useNavigation } from "./navigation";

const open = defineModel<boolean>({ required: true });
const { t } = useI18n();
const route = useRoute();
const auth = useAuthStore();
const { adminLinks } = useNavigation();
const adminTitleId = useId();

// Une navigation (lien de la feuille ou retour arrière) referme la feuille.
watch(
  () => route.fullPath,
  () => {
    open.value = false;
  },
);
</script>

<style>
/* ElDrawer « btt » : coins arrondis, fond des barres, hauteur bornée (défilement interne). */
.el-drawer.more-sheet {
  max-height: 85dvh;
  border-radius: 22px 22px 0 0;
  background: var(--c-bar);
}
.el-drawer.more-sheet .el-drawer__header {
  margin-bottom: 0;
  font-size: 20px;
  font-weight: 800;
  color: var(--c-ink);
}
</style>
