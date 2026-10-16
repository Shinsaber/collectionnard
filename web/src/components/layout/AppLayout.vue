<template>
  <!--
    Coquille de l'application : menu latéral à partir de `lg` (1024 px) ; en dessous, barre du
    haut + barre d'onglets fixe et feuille « Plus » (compte, réglages, administration).
  -->
  <div class="min-h-dvh bg-canvas text-ink lg:flex">
    <a
      href="#contenu"
      class="sr-only z-50 rounded-lg bg-primary px-4 py-2 font-semibold text-primary-ink focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
      >{{ t("layout.skipToContent") }}</a
    >

    <AppSidebar class="hidden! lg:flex!" />

    <div class="flex min-w-0 flex-1 flex-col">
      <AppTopBar class="lg:hidden" @open-more="moreOpen = true" />
      <main
        id="contenu"
        tabindex="-1"
        class="mx-auto w-full max-w-7xl flex-1 px-4 pt-5 pb-28 outline-none lg:px-10 lg:pt-8 lg:pb-12"
      >
        <router-view />
      </main>
    </div>

    <AppTabBar
      class="lg:hidden"
      :more-open="moreOpen"
      @open-more="moreOpen = true"
    />
    <AppMoreSheet v-model="moreOpen" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useOfflineStore } from "@/stores/offline";
import AppMoreSheet from "./AppMoreSheet.vue";
import AppSidebar from "./AppSidebar.vue";
import AppTabBar from "./AppTabBar.vue";
import AppTopBar from "./AppTopBar.vue";

const { t } = useI18n();
const moreOpen = ref(false);
// Copie hors ligne des collections (lot E5) : chargée puis rafraîchie dès l'entrée dans l'application.
onMounted(() => useOfflineStore().start());
</script>
