<template>
  <div class="flex min-h-dvh flex-col bg-canvas text-ink">
    <header
      class="flex items-center justify-between gap-3 bg-bar px-4 py-3 shadow-bar sm:px-6"
    >
      <RouterLink to="/" :aria-label="t('sharing.public.mySpace')">
        <AppLogo :size="28" />
      </RouterLink>
      <div class="flex items-center gap-3">
        <LangSwitch />
        <RouterLink
          :to="auth.isAuthenticated ? '/' : '/login'"
          class="el-button el-button--primary"
          data-test="public-account"
          >{{
            auth.isAuthenticated
              ? t("sharing.public.mySpace")
              : t("sharing.public.signIn")
          }}</RouterLink
        >
      </div>
    </header>
    <main class="mx-auto w-full max-w-6xl flex-1 px-4 py-6 sm:px-6">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
// Coquille des pages publiques (P4-2) : profil, collection, lien de partage — sans menu
// de l'application ni administration, accessibles sans compte.
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import AppLogo from "./AppLogo.vue";
import LangSwitch from "./LangSwitch.vue";

const { t } = useI18n();
const auth = useAuthStore();
</script>
