<template>
  <main
    class="flex min-h-dvh flex-col items-center justify-center gap-6 bg-canvas px-4 py-10 text-ink"
  >
    <div
      class="flex w-full max-w-sm flex-col gap-7 rounded-2xl bg-surface p-7 shadow-raised sm:p-9"
    >
      <div class="flex flex-col items-center gap-2.5 text-center">
        <h1><AppLogo :size="40" /></h1>
        <p class="text-sm text-muted">{{ t("auth.tagline") }}</p>
      </div>

      <form class="flex flex-col gap-4" @submit.prevent="handleLogin">
        <div class="flex flex-col gap-1.5">
          <label
            for="username"
            class="font-mono text-[11px] uppercase tracking-wider text-muted"
            >{{ t("auth.username") }}</label
          >
          <ElInput
            id="username"
            v-model="username"
            size="large"
            :placeholder="t('auth.usernamePlaceholder')"
            autocomplete="username"
            autofocus
          />
        </div>

        <div class="flex flex-col gap-1.5">
          <label
            for="password"
            class="font-mono text-[11px] uppercase tracking-wider text-muted"
            >{{ t("auth.password") }}</label
          >
          <ElInput
            id="password"
            v-model="password"
            size="large"
            type="password"
            show-password
            autocomplete="current-password"
          />
        </div>

        <ElAlert
          v-if="error"
          type="error"
          :closable="false"
          show-icon
          :title="error"
        />

        <ElButton
          type="primary"
          size="large"
          native-type="submit"
          class="mt-1"
          :loading="loading"
        >
          {{ t("auth.login") }}
        </ElButton>
      </form>
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
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import AppLogo from "@/components/layout/AppLogo.vue";

const auth = useAuthStore();
const router = useRouter();
const { t } = useI18n();
const { lang, setLang, langs, langNames } = useLang();

const username = ref("");
const password = ref("");
const loading = ref(false);
// Clé de catalogue plutôt que texte : le message suit un changement de langue.
const errorKey = ref("");
const error = computed(() => (errorKey.value ? t(errorKey.value) : ""));

async function handleLogin() {
  errorKey.value = "";
  loading.value = true;
  try {
    await auth.login({ username: username.value, password: password.value });
    router.push("/");
  } catch {
    errorKey.value = "auth.invalidCredentials";
  } finally {
    loading.value = false;
  }
}
</script>
