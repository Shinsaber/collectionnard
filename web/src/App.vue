<template>
  <!-- Locale Element Plus (pagination, sélecteurs…) alignée sur la langue de l'app. -->
  <ElConfigProvider :locale="elementLocale">
    <router-view />
  </ElConfigProvider>
</template>

<script setup lang="ts">
import { useLang } from "@/composables/useLang";

const { elementLocale } = useLang();
</script>
