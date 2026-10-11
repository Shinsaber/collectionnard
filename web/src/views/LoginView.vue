<template>
  <div
    class="min-h-screen flex items-center justify-center bg-gray-100 dark:bg-gray-900"
  >
    <div
      class="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-8 w-full max-w-sm"
    >
      <h1
        class="text-2xl font-bold text-center text-indigo-600 dark:text-indigo-400 mb-6"
      >
        📦 Collectionnard
      </h1>

      <!-- Sélecteur de langue (l'en-tête de l'application n'est pas visible ici) -->
      <div
        class="flex justify-center gap-1 mb-4"
        role="group"
        :aria-label="t('auth.language')"
      >
        <ElButton
          v-for="l in langs"
          :key="l"
          text
          size="small"
          class="font-mono"
          :type="l === lang ? 'primary' : undefined"
          :title="langNames[l]"
          :aria-label="langNames[l]"
          :aria-pressed="l === lang"
          :lang="l"
          @click="setLang(l)"
          >{{ l.toUpperCase() }}</ElButton
        >
      </div>

      <form class="flex flex-col gap-4" @submit.prevent="handleLogin">
        <div class="flex flex-col gap-1">
          <label
            for="username"
            class="text-sm font-medium text-gray-700 dark:text-gray-300"
            >{{ t("auth.username") }}</label
          >
          <ElInput
            id="username"
            v-model="username"
            :placeholder="t('auth.usernamePlaceholder')"
            autocomplete="username"
            autofocus
          />
        </div>

        <div class="flex flex-col gap-1">
          <label
            for="password"
            class="text-sm font-medium text-gray-700 dark:text-gray-300"
            >{{ t("auth.password") }}</label
          >
          <ElInput
            id="password"
            v-model="password"
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

        <ElButton type="primary" native-type="submit" :loading="loading">
          <i class="pi pi-sign-in mr-2" aria-hidden="true"></i>
          {{ t("auth.login") }}
        </ElButton>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";

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
