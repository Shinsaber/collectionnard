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
    <LangSwitch />
  </main>
</template>

<script setup lang="ts">
import AppLogo from "@/components/layout/AppLogo.vue";
import LangSwitch from "@/components/layout/LangSwitch.vue";

defineProps<{ subtitle: string }>();
</script>
