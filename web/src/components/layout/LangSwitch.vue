<template>
  <div class="flex gap-1.5" role="group" :aria-label="t('auth.language')">
    <button
      v-for="l in langs"
      :key="l"
      type="button"
      class="h-9 rounded-full px-3.5 font-mono text-xs font-medium"
      :class="
        l === lang
          ? 'bg-tint text-tint-ink shadow-tint'
          : 'text-muted hover:bg-surface'
      "
      :title="langNames[l]"
      :aria-label="langNames[l]"
      :aria-pressed="l === lang"
      :lang="l"
      @click="setLang(l)"
    >
      {{ l.toUpperCase() }}
    </button>
  </div>
</template>

<script setup lang="ts">
// Sélecteur de langue des pages sans la coquille de l'application (connexion, pages publiques).
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";

const { t } = useI18n();
const { lang, setLang, langs, langNames } = useLang();
</script>
