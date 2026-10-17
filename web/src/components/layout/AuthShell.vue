<template>
  <!-- Pages hors session (connexion, inscription, nouveau mot de passe) : carte centrée et langue. -->
  <main
    class="flex min-h-dvh flex-col items-center justify-center gap-6 bg-canvas px-4 py-10 text-ink"
  >
    <div
      class="flex w-full max-w-sm flex-col gap-7 rounded-2xl bg-surface p-7 shadow-raised sm:p-9"
    >
      <div class="flex flex-col items-center gap-2.5 text-center">
        <h1><AppLogo :size="40" /></h1>
        <p class="text-sm text-muted">{{ subtitle }}</p>
      </div>
      <slot />
    </div>

    <!-- Sélecteur de langue (la coquille de l'application n'est pas visible ici) -->
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
  </main>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import AppLogo from "@/components/layout/AppLogo.vue";

defineProps<{ subtitle: string }>();
const { t } = useI18n();
const { lang, setLang, langs, langNames } = useLang();
</script>
